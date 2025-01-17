package server

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"unsafe"

	"golang.org/x/sys/unix"
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

func (a *Acceptor) serverLoop(ctx context.Context) {
	// ここ関数化したいけど、関数化するとsockAddrのポインタがめんどいことになるな...
	// というかこのsockAddr並行安全性がないので子のループは同時並行ではない
	a.sockAddr = &sockAddr{}
	a.sockLen = uint32(unsafe.Sizeof(a.sockAddr))
	// sockAddrが一つしかないので並行処理できない
	a.uring.Accpet(a.socket, a.sockAddr, &a.sockLen)

	slog.InfoContext(ctx, "MainLoop start")
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP

		default:

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

			//TODO: cqeからデータを受け取ることがわかるもっといい名前
			res, eventType, sourceFd := a.uring.Wait()
			slog.InfoContext(ctx, "CQE", "cqe res", res, "event Type", eventType, "fd", sourceFd)

			// handle eventとして関数化したいな
			switch eventType {
			case EVENT_TYPE_ACCEPT:
				sockaddr, err := unix.Getpeername(int(res))
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
						Fd:        res,
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
			case EVENT_TYPE_READ:
				peer := a.peerAcceptor.GetPeer(sourceFd)
				// peerが持っているbuffer領域にコピーする
				buffer := make([]byte, res)
				a.uring.Read(buffer)
				peer.Buffer = buffer
				req, err := http.ReadRequest(bufio.NewReader(peer))
				if err != nil {
					slog.Error("ReadRequest", "err", err)
				}
				slog.Info("Request", "method", req.Method, "url", req.URL, "req", req)

				// 200 OK を返す
				body := "hello! I'm go server !"
				contentLen := len(body)
				response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\n\r\n%s", contentLen, body)
				peer.Write([]byte(response))
				//slog.InfoContext(ctx, "Write", "peer", peer, "res", res)

			case EVENT_TYPE_WRITE:
				peer := a.peerAcceptor.GetPeer(sourceFd)
				slog.ErrorContext(ctx, "Write", "peer", peer, "res", res)
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
