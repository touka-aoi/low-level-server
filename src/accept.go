package server

import (
	"bufio"
	"context"
	"encoding/binary"
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
}

// TODO: touka-aoi change the name
func NewAcceptor() *Acceptor {
	//TODO: touka-aoi add error handling to CreateSocket
	socket := CreateSocket()
	uring := CreateUring(maxConnection)
	return &Acceptor{
		socket:        socket,
		uring:         uring,
		maxConnection: maxConnection,
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

func (a *Acceptor) Accept(ctx context.Context) {
	//for peer := range a.AcceptChan {
	//	// io-uring で read する待機列に入れる
	//	//a.watchPeer(&peer)
	//}
}

func (a *Acceptor) watchPeer(peer *Peer) {
	err := a.uring.WatchRead(peer)
	if err != nil {
		slog.Debug("WatchRead", "err", err)
	}
}

func (a *Acceptor) Serve(ctx context.Context) {
	// ここでサーバーループを回す
	go a.serverLoop(ctx)

	// ほんとは上の関数をメインで走らせたい
	<-ctx.Done()
}

func (a *Acceptor) writeLoop(ctx context.Context) {
	for writer := range a.writerChan {
		a.uring.Write(peer, data)
	}
}

func (a *Acceptor) serverLoop(ctx context.Context) {
	// ここ関数化したいけど、関数化するとsockAddrのポインタがめんどいことになるな...
	// というかこのsockAddr並行安全性がないので子のループは同時並行ではない
	a.sockAddr = &sockAddr{}
	a.sockLen = uint32(unsafe.Sizeof(a.sockAddr))
	//TODO: アクセプトの命令を出すことがわかる関数名 ( 実際にAccpetするわけではない ) PrepareAccept?
	a.uring.Accpet(a.socket, a.sockAddr, &a.sockLen)
	peerAcceptor := NewPeerAcceptor()

	slog.InfoContext(ctx, "MainLoop start")
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		default:

			//TODO: cqeからデータを受け取ることがわかるもっといい名前
			res, eventType, fd := a.uring.Wait()
			//slog.InfoContext(ctx, "CQE", "cqe res", cqe.Res, "cqe user data", cqe.UserData, "cqe flags", cqe.Flags)

			// handle eventとして関数化したいな
			switch eventType {
			case EVENT_TYPE_ACCEPT:
				switch a.sockAddr.Family {
				case unix.AF_INET:
					port := binary.BigEndian.Uint16(a.sockAddr.Data[0:2])
					addr := netip.AddrFrom4([4]byte(a.sockAddr.Data[2:6]))

					ip := netip.AddrPortFrom(addr, port)

					peer := &Peer{
						Fd: res,
						Ip: ip,
					}
					slog.InfoContext(ctx, "Accept", "fd", peer.Fd, "ip", peer.Ip)
					err := a.uring.WatchRead(peer)
					if err != nil {
						slog.ErrorContext(ctx, "WatchRead", "err", err)
					}

					peerAcceptor.SetPeer(peer.Fd, peer)
				}
			case EVENT_TYPE_READ:
				peer := peerAcceptor.GetPeer(fd)
				// peerが持っているbuffer領域にコピーする
				peer.Buffer = make([]byte, res)
				a.uring.Read(peer)

				http.ReadRequest(bufio.NewReader(peer.Buffer))

				//a.HandleData(data)
				slog.DebugContext(ctx, "Read", "peer", peer)
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

type bufioResponseWriter struct {
	writer      *bufio.Writer
	headers     http.Header
	status      int
	wroteHeader bool
}

func NewBufioResponseWriter(w *bufio.Writer) *bufioResponseWriter {
	return &bufioResponseWriter{
		writer:  w,
		headers: make(http.Header),
		status:  http.StatusOK,
	}
}

func (w *bufioResponseWriter) Header() http.Header {
	return w.headers
}

func (w *bufioResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.writer.Write(data)
}

func (w *bufioResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.status = statusCode
	fmt.Fprintf(w.writer, "HTTP/1.1 %d %s\r\n", w.status, http.StatusText(w.status))
	w.headers.Write(w.writer)
	w.writer.WriteString("\r\n")
	w.wroteHeader = true
}
