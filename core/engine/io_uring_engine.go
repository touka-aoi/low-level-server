//go:build linux

package engine

import (
	"context"
	"log/slog"
	"net/netip"
	"slices"

	"github.com/touka-aoi/low-level-server/core/event"
	"github.com/touka-aoi/low-level-server/core/io"
	"golang.org/x/sys/unix"
)

type userData struct {
	eventType event.EventType
	fd        int32
}

type UringNetEngine struct {
	uring *io.Uring
}

func NewUringNetEngine() *UringNetEngine {
	uring := io.CreateUring(4096)
	return &UringNetEngine{
		uring: uring,
	}
}

func (e *UringNetEngine) Accept(ctx context.Context, listener Listener) error {
	op := e.uring.AccpetMultishot(listener.Fd(), e.encodeUserData(event.EVENT_TYPE_ACCEPT, listener.Fd()))
	e.uring.Submit(op)
	return nil
}

// ReceiveData関数は一つのCQEイベントを処理して、イベントとして返します
// ここでIO_URINGの依存関係を打ち切ります
func (e *UringNetEngine) ReceiveData(ctx context.Context) ([]*NetEvent, error) {
	// 一度の呼びだしで溜まっているCQEイベントを全て消費します (1ループ60fpsで処理できるイベントの数は考え中です)
	// ここはチャンネルとかの方がいいのか？
	cqeEvents, err := e.uring.PeekBatchEvents(1)
	if err != nil {
		return nil, err
	}

	if len(cqeEvents) == 0 {
		return nil, nil // ここwouldBlockの方がいいか？ そんなことないか
	}

	netEvents := make([]*NetEvent, 0, len(cqeEvents))

	for cqeEvent := range slices.Values(cqeEvents) {
		if cqeEvent.Res < 0 {
			// ここではCQEエラーを処理します
			// Acceptの場合とかReadの場合などmultishotをうまく処理しないといけません
			slog.ErrorContext(ctx, "Error in CQE event", "error", cqeEvent.Res)
			panic("CQE event error") // ここはpanicしない方がいいかもしれません
		}

		userData := e.decodeUserData(cqeEvent.UserData)

		switch userData.eventType {
		case event.EVENT_TYPE_ACCEPT:
			netEvents = append(netEvents, &NetEvent{
				EventType: event.EVENT_TYPE_ACCEPT,
				Fd:        cqeEvent.Res,
				Data:      nil,
			})
		case event.EVENT_TYPE_READ:
			if cqeEvent.Res == 0 {
				// end of file ?
			}
			//TODO: makeしてるのはよくないのでリングバッファにしたい
			data := make([]byte, 0, cqeEvent.Res) // cqeEvent.Resは受信したバイト
			// Readイベントの場合はDataに受信したデータを格納します
			e.uring.Read(data)
			netEvents = append(netEvents, &NetEvent{
				EventType: event.EVENT_TYPE_READ,
				Fd:        userData.fd,
				Data:      data,
			})
		case event.EVENT_TYPE_WRITE:
			// writeイベントはCQEを発行しない
		default:
			// 他のイベントタイプはここで処理する必要があります
			// なんかエラーを出したいなぁという気分ではあります。
			return nil, nil
		}
	}
	return netEvents, nil
}

func (e *UringNetEngine) handleEvent() error {
	return nil
}

func (e *UringNetEngine) PrepareClose() error {
	return nil
}

func (e *UringNetEngine) RegisterRead(ctx context.Context, peer *Peer) error {
	userData := e.encodeUserData(event.EVENT_TYPE_READ, peer.Fd)
	op := e.uring.ReadMultishot(peer.Fd, userData)
	e.uring.Submit(op)
	return nil
}

func (e *UringNetEngine) Close() error {
	return e.uring.Close()
}

func (e *UringNetEngine) encodeUserData(ev event.EventType, fd int32) uint64 {
	userData := uint64(ev)<<32 | uint64(fd)
	return userData
}

func (e *UringNetEngine) decodeUserData(data uint64) *userData {
	return &userData{
		eventType: event.EventType(data >> 32),
		fd:        int32(data & 0xFFFFFFFF),
	}
}

func (e *UringNetEngine) GetPeerName(ctx context.Context, fd int32) (*Peer, error) {
	localSockAddr, err := unix.Getsockname(int(fd))
	if err != nil {
		return nil, err
	}

	remoteSockAddr, err := unix.Getpeername(int(fd))
	if err != nil {
		return nil, err
	}

	var localAddrPort netip.AddrPort
	switch addr := localSockAddr.(type) {
	case *unix.SockaddrInet4:
		ip := netip.AddrFrom4([4]byte{addr.Addr[0], addr.Addr[1], addr.Addr[2], addr.Addr[3]})
		localAddrPort = netip.AddrPortFrom(ip, uint16(addr.Port))
	case *unix.SockaddrInet6:
		ip := netip.AddrFrom16(addr.Addr)
		localAddrPort = netip.AddrPortFrom(ip, uint16(addr.Port))
	default:
		return nil, unix.EAFNOSUPPORT
	}

	var remoteAddrPort netip.AddrPort
	switch addr := remoteSockAddr.(type) {
	case *unix.SockaddrInet4:
		ip := netip.AddrFrom4([4]byte{addr.Addr[0], addr.Addr[1], addr.Addr[2], addr.Addr[3]})
		remoteAddrPort = netip.AddrPortFrom(ip, uint16(addr.Port))
	case *unix.SockaddrInet6:
		ip := netip.AddrFrom16(addr.Addr)
		remoteAddrPort = netip.AddrPortFrom(ip, uint16(addr.Port))
	default:
		return nil, unix.EAFNOSUPPORT
	}

	return &Peer{
		Fd:         fd,
		LocalAddr:  localAddrPort,
		RemoteAddr: remoteAddrPort,
	}, nil
}
