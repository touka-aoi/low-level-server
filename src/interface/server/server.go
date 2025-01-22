package server

import (
	"context"
	"github.com/touka-aoi/low-level-server/application/handler"
	"github.com/touka-aoi/low-level-server/interface/components"
	"github.com/touka-aoi/low-level-server/internal"
	"golang.org/x/sys/unix"
	"log/slog"
	"net/netip"
)

const maxConnection = 4096

type Server struct {
	uring         *internal.Uring
	socket        *internal.Socket
	AcceptChan    chan *components.Peer
	maxConnection int
	writeChan     chan *components.Peer
	peerContainer *components.PeerContainer
	handler       handler.Handler
}

func NewAcceptor() *Server {
	socket := internal.CreateSocket()
	//TODO: touka-aoi refactor option structure
	ID := 1
	uring := internal.CreateUring(maxConnection)
	uring.RegisterRingBuffer(maxConnection, ID)
	httpHandler := handler.NewHttpHandler()
	return &Server{
		socket:        socket,
		uring:         uring,
		maxConnection: maxConnection,
		peerContainer: components.NewPeerContainer(),
		writeChan:     make(chan *components.Peer, components.MaxOSFileDescriptor),
		handler:       httpHandler,
	}
}

func (a *Server) Close() {
	_ = a.socket.Close()
	_ = a.uring.Close()
}

func (a *Server) Listen(address string) error {
	addr, err := netip.ParseAddrPort(address)
	if err != nil {
		return err
	}

	a.socket.Bind(addr)
	a.socket.Listen(a.maxConnection)

	return nil
}

func (a *Server) Serve(ctx context.Context) {
	go a.serverLoop(ctx)

	<-ctx.Done()
}

func (a *Server) accept() {
	a.uring.AccpetMultishot(a.socket)
}

func (a *Server) serverLoop(ctx context.Context) {
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
			case internal.EVENT_TYPE_ACCEPT:
				a.handleAccept(ctx, cqe)
			case internal.EVENT_TYPE_WRITE:
				a.handleWrite(ctx, cqe)
			case internal.EVENT_TYPE_READ:
				a.handleRead(ctx, cqe)
			}
		}
	}
}

func (a *Server) watchPeer(peer *components.Peer) {
	a.uring.WatchReadMultiShot(peer.Fd)
}

func (a *Server) handleAccept(ctx context.Context, cqe *internal.UringCQE) {
	sockaddr, err := unix.Getpeername(int(cqe.Res))
	if err != nil {
		slog.ErrorContext(ctx, "Getpeername", "err", err)
		return
	}

	// IORING_CQE_F_MOREフラグをチェクし、何か問題が起きていないか確認する
	// 問題が起きていた場合、再度Accpet_MultiShotを行う

	switch sa := sockaddr.(type) {
	case *unix.SockaddrInet4:
		addr := netip.AddrFrom4(sa.Addr)
		ip := netip.AddrPortFrom(addr, uint16(sa.Port))

		peer := &components.Peer{
			Fd:        cqe.Res,
			Ip:        ip,
			WriteChan: a.writeChan,
		}
		slog.InfoContext(ctx, "Accept", "fd", peer.Fd, "ip", peer.Ip)

		a.watchPeer(peer)
		a.peerContainer.RegisterPeer(peer.Fd, peer)
	}
}

func (a *Server) handleRead(ctx context.Context, cqe *internal.UringCQE) {
	_, sourceFD := a.uring.DecodeUserData(cqe.UserData)
	if cqe.Res < 1 {
		slog.InfoContext(ctx, "EOF", "fd", sourceFD)
		a.peerContainer.UnregisterPeer(sourceFD)
		return
	}
	peer := a.peerContainer.GetPeer(sourceFD)
	buffer := make([]byte, cqe.Res)
	a.uring.Read(buffer)
	peer.Buffer = buffer

	err := a.handler.OnRead(ctx, peer, buffer)
	if err != nil {
		slog.ErrorContext(ctx, "Handler OnRead", "err", err)
	}
}

func (a *Server) handleWrite(ctx context.Context, cqe *internal.UringCQE) {
	_, fd := a.uring.DecodeUserData(cqe.UserData)
	peer := a.peerContainer.GetPeer(fd)
	slog.ErrorContext(ctx, "Write", "peer", peer, "res", cqe.Res)
}
