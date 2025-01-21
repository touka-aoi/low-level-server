package server

import (
	"bufio"
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"log/slog"
	"net/http"
	"net/netip"
	"unsafe"
)

const maxConnection = 4096

type Acceptor struct {
	uring         *Uring
	socket        *Socket
	AcceptChan    chan *Peer
	sockAddr      *sockAddr
	sockLen       uint32
	maxConnection int
	writeChan     chan *Peer
	peerAcceptor  *PeerAcceptor
}

// TODO: touka-aoi change the name
func NewAcceptor() *Acceptor {
	//TODO: touka-aoi add error handling to CreateSocket
	socket := CreateSocket()
	uring := CreateUring(maxConnection)
	uring.RegisterRingBuffer(1)
	return &Acceptor{
		socket:        socket,
		uring:         uring,
		maxConnection: maxConnection,
		peerAcceptor:  NewPeerAcceptor(),
		writeChan:     make(chan *Peer, maxOSFileDescriptor),
	}
}

func (a *Acceptor) Close() {
	a.socket.Close()
	a.uring.Close()
}

func (a *Acceptor) Listen(address string) error {
	addr, err := netip.ParseAddrPort(address)
	if err != nil {
		return err
	}

	a.socket.Bind(addr)
	a.socket.Listen(a.maxConnection)

	return nil
}

func (a *Acceptor) Serve(ctx context.Context) {
	// ここでサーバーループを回す
	go a.serverLoop(ctx)

	// ほんとは上の関数をメインで走らせたい
	<-ctx.Done()
}

func (a *Acceptor) accept() {
	a.sockAddr = &sockAddr{}
	a.sockLen = uint32(unsafe.Sizeof(a.sockAddr))
	a.uring.AccpetMultishot(a.socket, a.sockAddr, &a.sockLen)
}

func (a *Acceptor) serverLoop(ctx context.Context) {
	slog.InfoContext(ctx, "serverLoop start")

	a.accept()
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP

		default:

			// このループは別建てでも別にええな
		LOOP2:
			for {
				select {
				case writer := <-a.writeChan:
					//TODO: 一度に複数のデータをopに変換してSQを一度だけ呼び出す
					a.uring.Write(writer.Fd, writer.Buffer)
				default:
					break LOOP2
				}
			}

			cqe := a.uring.WaitEvent()
			cqe2 := cqe.DecodeUserData()
			slog.InfoContext(ctx, "CQE", "cqe res", cqe2.Res, "event Type", cqe2.EventType, "fd", cqe2.SourceFD)

			switch cqe2.EventType {
			case EVENT_TYPE_ACCEPT:
				sockaddr, err := unix.Getpeername(int(cqe2.Res))
				if err != nil {
					slog.ErrorContext(ctx, "Getpeername", "err", err)
					continue
				}

				// IORING_CQE_F_MOREフラグをチェクし、何か問題が起きていないか確認する
				// 問題が起きていた場合、再度Accpet_MultiShotを行う

				switch sa := sockaddr.(type) {
				case *unix.SockaddrInet4:
					addr := netip.AddrFrom4(sa.Addr)
					ip := netip.AddrPortFrom(addr, uint16(sa.Port))

					peer := &Peer{
						Fd:        cqe2.Res,
						Ip:        ip,
						writeChan: a.writeChan,
					}
					slog.InfoContext(ctx, "Accept", "fd", peer.Fd, "ip", peer.Ip)

					err := a.watchPeer(peer)
					if err != nil {
						slog.ErrorContext(ctx, "WatchRead", "err", err)
					}

					a.peerAcceptor.RegisterPeer(peer.Fd, peer)
				}
			case EVENT_TYPE_WRITE:
				peer := a.peerAcceptor.GetPeer(cqe2.SourceFD)
				slog.ErrorContext(ctx, "Write", "peer", peer, "res", cqe2.Res)

			case EVENT_TYPE_READ:
				// https://manpages.debian.org/unstable/manpages-dev/pread.2.en.html
				// a return of zero indicates end of file
				if cqe.Res < 1 {
					slog.InfoContext(ctx, "EOF", "fd", cqe2.SourceFD)
					err := a.peerAcceptor.GetPeer(cqe2.SourceFD).Close()
					if err != nil {
						slog.ErrorContext(ctx, "Close", "err", err)
					}
					continue
				}
				peer := a.peerAcceptor.GetPeer(cqe2.SourceFD)
				buffer := make([]byte, cqe2.Res)
				a.uring.Read(buffer)
				peer.Buffer = buffer

				// applicaition handler に渡す
				req, err := http.ReadRequest(bufio.NewReader(peer))
				if err != nil {
					slog.Error("ReadRequest", "err", err)
				}
				slog.Info("Read", "method", req.Method, "url", req.URL, "req", req)

				// 200 OK を返す
				body := "hello! I'm go server !"
				contentLen := len(body)
				response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\n\r\n%s", contentLen, body)
				peer.Write([]byte(response))
				//slog.InfoContext(ctx, "Write", "peer", peer, "res", res)

			}
		}
	}

	slog.InfoContext(ctx, "MainLoop end")
}

func (a *Acceptor) HandleData(r *bufio.Reader) {
	//req, err := http.ReadRequest(r)
	//if err != nil {
	//	slog.Error("ReadRequest", "err", err)
	//}
	//slog.Info("Request", "method", req.Method, "url", req.URL, "req", req)
	//
	//// fdからwriterを作成しないといけない
	//rw := NewBufioResponseWriter()
	//mux := http.NewServeMux()
	//mux.ServeHTTP(rw, req)
	//
	//if err := a.writer.Flush(); err != nil {
	//	fmt.Fprintf(os.Stderr, "Flush error: %v\n", err)
	//}
}

func (a *Acceptor) watchPeer(peer *Peer) error {
	err := a.uring.WatchRead(peer.Fd)
	if err != nil {
		return err
	}

	return nil
}

// type bufioResponseWriter struct {
// 	writer      *bufio.Writer
// 	headers     http.Header
// 	status      int
// 	wroteHeader bool
// }

// func NewBufioResponseWriter(w *bufio.Writer) *bufioResponseWriter {
// 	return &bufioResponseWriter{
// 		writer:  w,
// 		headers: make(http.Header),
// 		status:  http.StatusOK,
// 	}
// }

// func (w *bufioResponseWriter) Header() http.Header {
// 	return w.headers
// }

// func (w *bufioResponseWriter) Write(data []byte) (int, error) {
// 	if !w.wroteHeader {
// 		w.WriteHeader(http.StatusOK)
// 	}
// 	return w.writer.Write(data)
// }

// func (w *bufioResponseWriter) WriteHeader(statusCode int) {
// 	if w.wroteHeader {
// 		return
// 	}
// 	w.status = statusCode
// 	fmt.Fprintf(w.writer, "HTTP/1.1 %d %s\r\n", w.status, http.StatusText(w.status))
// 	w.headers.Write(w.writer)
// 	w.writer.WriteString("\r\n")
// 	w.wroteHeader = true
// }
