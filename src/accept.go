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

func NewAcceptor() *Acceptor {
	socket := CreateSocket()
	//TODO: touka-aoi refactor option structure
	ID := 1
	uring := CreateUring(maxConnection)
	uring.RegisterRingBuffer(ID)
	return &Acceptor{
		socket:        socket,
		uring:         uring,
		maxConnection: maxConnection,
		peerAcceptor:  NewPeerAcceptor(),
		writeChan:     make(chan *Peer, maxOSFileDescriptor),
	}
}

func (a *Acceptor) Close() {
	_ = a.socket.Close()
	_ = a.uring.Close()
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
	go a.serverLoop(ctx)

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
	for {
		select {
		case <-ctx.Done():
			return

		case writer := <-a.writeChan:
			a.uring.Write(writer.Fd, writer.Buffer)

		default:
			cqe, err := a.uring.WaitEvent()
			if err != nil {
				slog.ErrorContext(ctx, "WaitEvent", "err", err)
				continue
			}
			eventType, sourceFD := a.uring.DecodeUserData(cqe.UserData)
			slog.InfoContext(ctx, "CQE", "cqe res", cqe.Res, "event Type", eventType, "fd", sourceFD)

			switch eventType {
			case EVENT_TYPE_ACCEPT:
				sockaddr, err := unix.Getpeername(int(cqe.Res))
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
						Fd:        cqe.Res,
						Ip:        ip,
						writeChan: a.writeChan,
					}
					slog.InfoContext(ctx, "Accept", "fd", peer.Fd, "ip", peer.Ip)

					a.watchPeer(peer)
					a.peerAcceptor.RegisterPeer(peer.Fd, peer)
				}
			case EVENT_TYPE_WRITE:
				peer := a.peerAcceptor.GetPeer(sourceFD)
				slog.ErrorContext(ctx, "Write", "peer", peer, "res", cqe.Res)

			case EVENT_TYPE_READ:
				// https://manpages.debian.org/unstable/manpages-dev/pread.2.en.html
				// a return of zero indicates end of file
				if cqe.Res < 1 {
					slog.InfoContext(ctx, "EOF", "fd", sourceFD)
					a.peerAcceptor.UnregisterPeer(sourceFD)
					continue
				}
				peer := a.peerAcceptor.GetPeer(sourceFD)
				buffer := make([]byte, cqe.Res)
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

			}
		}
	}
}

func (a *Acceptor) watchPeer(peer *Peer) {
	a.uring.WatchReadMultiShot(peer.Fd)
}
