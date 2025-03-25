package internal

import (
  "bufio"
  "context"
	"errors"
	"github.com/touka-aoi/low-level-server/application/handler"
	"github.com/touka-aoi/low-level-server/application/handler/game"
	gen "github.com/touka-aoi/low-level-server/gen/proto"
	"github.com/touka-aoi/low-level-server/interface/components"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"
	"io"
	"log/slog"
	"net/netip"
)

const maxConnection = 4096
const retryCapacity = 4096

type Server struct {
	uring         *Uring
	socket        *Socket
	AcceptChan    chan *components.Peer
	maxConnection int
	writeChan     chan *components.Peer
	retryChan     chan *UringSQE
	peerContainer *components.PeerContainer
	handler       game.Handler
}

func NewAcceptor() *Server {
	socket := CreateSocket()
	//TODO: touka-aoi refactor option structure
	ID := 1
	uring := CreateUring(maxConnection)
	uring.RegisterRingBuffer(maxConnection, ID)
	httpHandler := handler.NewHttpHandler()
	return &Server{
		socket:        socket,
		uring:         uring,
		maxConnection: maxConnection,
		peerContainer: components.NewPeerContainer(),
		writeChan:     make(chan *components.Peer, components.MaxOSFileDescriptor),
		retryChan:     make(chan *UringSQE, retryCapacity),
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
	slog.InfoContext(ctx, "serverLoop start")

	a.uring.AccpetMultishot(a.socket)
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
				a.handleAccept(ctx, cqe)
			case EVENT_TYPE_WRITE:
				a.handleWrite(ctx, cqe)
			case EVENT_TYPE_READ:
				a.handleRead(ctx, cqe)
			}
		}

		//select {
		//case op := <-a.retryChan:
		//	a.uring.Submit(op)
		//default:
		//}
	}
}

func (a *Server) handleAccept(ctx context.Context, cqe *UringCQE) {
	sockaddr, err := unix.Getpeername(int(cqe.Res))
	if err != nil {
		slog.ErrorContext(ctx, "Getpeername", "err", err)
		return
	}

	if cqe.Flags&IORING_CQE_F_MORE == 0 {
		eventType, sourceFD := a.uring.DecodeUserData(cqe.UserData)
		slog.WarnContext(ctx, "IORING_CQE_F_MORE", "res", cqe.Res, "eventType", eventType, "sourceFD", sourceFD)
	}

	// ここは大げさすぎるな 必要なのはremoteAddrだけ
	// peerがwriteしたいものを持つのは良さそう chanしたらしんどすぎる
	//
	switch sa := sockaddr.(type) {
	case *unix.SockaddrInet4:
		addr := netip.AddrFrom4(sa.Addr)
		ip := netip.AddrPortFrom(addr, uint16(sa.Port))

		peer := &components.Peer{
      r: bufio.NewReader(何を入れたらいいかはわかってない),
      decode: s.ReadCodec
      encoder: s.WriteCodec
      w: 書く先
			Fd:        cqe.Res,
			Ip:        ip,
			WriteChan: a.writeChan,
		}
		slog.InfoContext(ctx, "Accept", "fd", peer.Fd, "ip", peer.Ip)

		a.uring.WatchReadMultiShot(peer.Fd)
		a.peerContainer.RegisterPeer(peer.Fd, peer)
	}
}

// ここ10ms以下にしたい
func (a *Server) handleRead(ctx context.Context, cqe *UringCQE) {
	evenType, sourceFD := a.uring.DecodeUserData(cqe.UserData)
	if cqe.Res < 1 {
		slog.InfoContext(ctx, "EOF", "fd", sourceFD)
		a.peerContainer.UnregisterPeer(sourceFD)
		return
	}

	if cqe.Flags&IORING_CQE_F_MORE == 0 {
		slog.WarnContext(ctx, "IORING_CQE_F_MORE", "res", cqe.Res, "eventType", evenType, "sourceFD", sourceFD)
	}

	peer := a.peerContainer.GetPeer(sourceFD)
	buffer := make([]byte, cqe.Res)
	a.uring.Read(buffer)
	peer.Buffer = buffer
	var message gen.Envelope
	// bufferはio.Bufferedにしないと
	err := protodelim.UnmarshalFrom(buffer, &message)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			// 何もしない
			return
		}
		slog.ErrorContext(ctx, "UnmarshalFrom", "err", err)
		// UnmarshalFromでエラーが起こった時どうしたらいいんだろうか...
	}

	switch message.WhichPayload() {
	case gen.Envelope_PlayerAction_case:
		// 何かする
		playerAction := message.GetPlayerAction()
		// ロジックサーバーに処理をおくる
		// resutlを返す
		var result gen.ActionResult
		data, err := proto.Marshal(&result)
		if err != nil {
			slog.ErrorContext(ctx, "proto.Marshal", "err", err)
		}
		peer.Buffer = data
		a.writeChan <- peer
	case gen.Envelope_StatusRequest_case:
		statusRequest := message.GetStatusRequest()
	// 何かする
	case gen.Envelope_Payload_not_set_case:
		// 何もしない
	}

}

func (a *Server) handleWrite(ctx context.Context, cqe *UringCQE) {
	_, fd := a.uring.DecodeUserData(cqe.UserData)
	peer := a.peerContainer.GetPeer(fd)
	slog.ErrorContext(ctx, "Write", "peer", peer, "res", cqe.Res)
}
