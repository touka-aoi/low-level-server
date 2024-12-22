package server

import (
	"context"
	"encoding/binary"
	"golang.org/x/sys/unix"
	"log/slog"
	"net/netip"
	"unsafe"
)

const maxConnection = 4096

type Peer struct {
	Fd int32
	Ip netip.AddrPort
}

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
	a.serverLoop(ctx)
}

func (a *Acceptor) serverLoop(ctx context.Context) {
	// ここ関数化したいけど、関数化するとsockAddrのポインタがめんどいことになるな...
	// というかこのsockAddr並行安全性がないので子のループは同時並行ではない
	a.sockAddr = &sockAddr{}
	a.sockLen = uint32(unsafe.Sizeof(a.sockAddr))
	//TODO: アクセプトの命令を出すことがわかる関数名 ( 実際にAccpetするわけではない ) PrepareAccept?
	a.uring.Accpet(a.socket, a.sockAddr, &a.sockLen)

	slog.InfoContext(ctx, "MainLoop start")
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		default:

			//TODO: cqeからデータを受け取ることがわかるもっといい名前
			cqe := a.uring.Wait()
			slog.InfoContext(ctx, "CQE", "cqe res", cqe.Res, "cqe user data", cqe.UserData, "cqe flags", cqe.Flags)

			// handle eventとして関数化したいな
			switch cqe.UserData {
			case EVENT_TYPE_ACCEPT:
				switch a.sockAddr.Family {
				case unix.AF_INET:
					port := binary.BigEndian.Uint16(a.sockAddr.Data[0:2])
					addr := netip.AddrFrom4([4]byte(a.sockAddr.Data[2:6]))

					ip := netip.AddrPortFrom(addr, port)

					peer := &Peer{
						Fd: int32(cqe.Res),
						Ip: ip,
					}
					slog.InfoContext(ctx, "Accept", "fd", peer.Fd, "ip", peer.Ip)
					err := a.uring.WatchRead(peer)
					if err != nil {
						slog.ErrorContext(ctx, "WatchRead", "err", err)
					}
				}
			case EVENT_TYPE_READ:
				data := a.uring.Read(cqe)
				slog.InfoContext(ctx, "Read", "data", string(data))
				//a.service.HandleData(data)
			}
		}
	}

	slog.InfoContext(ctx, "MainLoop end")
}
