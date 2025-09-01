//go:build linux

package engine

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"time"
	"unsafe"

	"github.com/touka-aoi/low-level-server/core/core"
	toukaerrors "github.com/touka-aoi/low-level-server/core/errors"
	"github.com/touka-aoi/low-level-server/core/event"
	"golang.org/x/sys/unix"
)

type userData struct {
	eventType event.EventType
	fd        int32
}

type SockAddr struct {
	Fd         int32
	LocalAddr  netip.AddrPort
	RemoteAddr netip.AddrPort
}

type UringNetEngine struct {
	uring *core.Uring
}

func (e *UringNetEngine) CancelAccept(ctx context.Context, listener Listener) error {
	return nil
}

func (e *UringNetEngine) ClosePeer(ctx context.Context, peer *Peer) error {
	return nil
}

func (e *UringNetEngine) WaitEvent() error {
	return e.uring.WaitEvent()
}

func (e *UringNetEngine) WaitEventWithTimeout(d time.Duration) error {
	return e.uring.WaitEventWithTimeout(d)
}

func NewUringNetEngine() *UringNetEngine {
	uring := core.CreateUring(4096)
	uring.RegisterRingBuffer(256, core.MaxBufferSize, 1)
	return &UringNetEngine{
		uring: uring,
	}
}

func (e *UringNetEngine) Accept(ctx context.Context, listener Listener) error {
	op := e.uring.AcceptMultishot(listener.Fd(), e.encodeUserData(event.EVENT_TYPE_ACCEPT, listener.Fd()))
	e.uring.Submit(op)
	return nil
}

func (e *UringNetEngine) RecvFrom(ctx context.Context, listener Listener) error {
	op := e.uring.RecvFrom(listener.Fd(), e.encodeUserData(event.EVENT_TYPE_RECVMSG, listener.Fd()))
	e.uring.Submit(op)
	return nil
}

// ReceiveData関数は一つのCQEイベントを処理して、イベントとして返します
// ここでIO_URINGの依存関係を打ち切ります
func (e *UringNetEngine) ReceiveData(ctx context.Context) ([]*NetEvent, error) {
	cqeEvents, err := e.uring.PeekBatchEvents(64)
	if err != nil {
		return nil, err
	}

	if len(cqeEvents) == 0 {
		return nil, toukaerrors.ErrWouldBlock // ここwouldBlockの方がいい そんなことあります
	}

	// slog.DebugContext(ctx, "Received CQE events", "cqeEvents", cqeEvents)

	netEvents := make([]*NetEvent, 0, len(cqeEvents))

	for cqeEvent := range slices.Values(cqeEvents) {
		userData := e.decodeUserData(cqeEvent.UserData)
		if cqeEvent.Res < 0 {
			slog.ErrorContext(ctx, "Error in CQE event", "eventType", userData.eventType, "fd", userData.fd, "error", cqeEvent.Res)
			panic("CQE event error") // ここはpanicしない方がいいかもしれません
		}

		switch userData.eventType {
		case event.EVENT_TYPE_ACCEPT:
			netEvents = append(netEvents, &NetEvent{
				EventType: event.EVENT_TYPE_ACCEPT,
				Fd:        cqeEvent.Res,
				Data:      nil,
			})
		case event.EVENT_TYPE_READ:
			if cqeEvent.Flags&core.IORING_CQE_F_MORE == 0 {
				// 再度Readイベントを起こす
				// 再度必要用のイベントで返せばいいか？？？考え中...
			}
			if cqeEvent.Res == -ENOBUFS {
				// これってどういう状況？
				slog.WarnContext(ctx, "No buffer available for read", "fd", userData.fd)
				return nil, nil
			}

			if cqeEvent.Flags&core.IORING_CQE_F_BUFFER == 0 {
				slog.WarnContext(ctx, "Read event without buffer flag", "fd", userData.fd, "flags", cqeEvent.Flags)
				return nil, nil
			}
			idx := cqeEvent.Flags >> core.IORING_CQE_BUFFER_SHIFT
			buff := e.uring.GetRingBuffer(uint16(idx))
			// recvの場合ここからvalidationが必要
			slog.DebugContext(ctx, "Read buffer", "fd", userData.fd, "bufferIndex", idx, "dataLength", len(buff))
			// engineが持っているバッファ領域にコピーしてあげたいが今回は新しく作っておく
			b := make([]byte, cqeEvent.Res)
			copy(b, buff[:cqeEvent.Res])
			slog.DebugContext(ctx, "Read event", "fd", userData.fd, "bytesRead", cqeEvent.Res, "flags", cqeEvent.Flags)
			netEvents = append(netEvents, &NetEvent{
				EventType: event.EVENT_TYPE_READ,
				Fd:        userData.fd,
				Data:      b,
			})
		case event.EVENT_TYPE_WRITE:
			// writeイベントはCQEを発行しない
		case event.EVENT_TYPE_RECVMSG:
			if cqeEvent.Flags&core.IORING_CQE_F_BUFFER == 0 {
				slog.WarnContext(ctx, "Read event without buffer flag", "fd", userData.fd, "flags", cqeEvent.Flags)
				return nil, nil
			}
			idx := cqeEvent.Flags >> core.IORING_CQE_BUFFER_SHIFT
			buff := e.uring.GetRingBuffer(uint16(idx))
			b := make([]byte, cqeEvent.Res)
			copy(b, buff[:cqeEvent.Res])

			addrBytes := unsafe.Slice(e.uring.Msghdr.Name, e.uring.Msghdr.Namelen)
			family := binary.LittleEndian.Uint16(addrBytes[0:2])
			var remoteAddr netip.AddrPort
			if family == unix.AF_INET && len(addrBytes) >= 16 {
				// IPv4の場合
				port := binary.BigEndian.Uint16(addrBytes[2:4])
				ip := net.IPv4(addrBytes[4], addrBytes[5], addrBytes[6], addrBytes[7])
				addr := fmt.Sprintf("%s:%d", ip, port)
				remoteAddr = netip.MustParseAddrPort(addr)
			}
			if cqeEvent.Flags&core.IORING_CQE_F_MORE == 0 {
				// F_MOREの原因はどうやって判定したらいいのか
				slog.DebugContext(ctx, "F_MORE flag not set, submitting new recvmsg operation", "fd", userData.fd)
				op := e.uring.RecvFrom(userData.fd, e.encodeUserData(event.EVENT_TYPE_RECVMSG, userData.fd))
				e.uring.Submit(op)
			}
			netEvents = append(netEvents, &NetEvent{
				EventType:  event.EVENT_TYPE_RECVMSG,
				Fd:         userData.fd,
				Data:       b,
				RemoteAddr: remoteAddr,
			})
		case event.EVENT_TYPE_TIMEOUT:
			slog.DebugContext(ctx, "Timeout event")
		default:
			slog.WarnContext(ctx, "Unknown event type", "eventType", userData.eventType)
			// 他のイベントタイプはここで処理する必要があります
			// なんかエラーを出したいなぁという気分ではあります。
			return nil, nil
		}
	}
	return netEvents, nil
}

func (e *UringNetEngine) PrepareClose() error {
	ud := e.encodeUserData(event.EVENT_TYPE_TIMEOUT, 0)
	e.uring.Timeout(0, ud)
	slog.Debug("PrepareClose")
	return nil
}

func (e *UringNetEngine) RegisterRead(ctx context.Context, peer *Peer) error {
	ud := e.encodeUserData(event.EVENT_TYPE_READ, peer.Fd)
	op := e.uring.ReadMultishot(peer.Fd, ud)
	slog.DebugContext(ctx, "Registering read operation", "fd", peer.Fd, "userData", ud)
	e.uring.Submit(op)
	return nil
}

func (e *UringNetEngine) Close() error {
	return e.uring.Close()
}

func (e *UringNetEngine) encodeUserData(ev event.EventType, fd int32) uint64 {
	ud := uint64(ev)<<32 | uint64(fd)
	return ud
}

func (e *UringNetEngine) decodeUserData(data uint64) *userData {
	return &userData{
		eventType: event.EventType(data >> 32),
		fd:        int32(data & 0xFFFFFFFF),
	}
}

func (e *UringNetEngine) GetSockAddr(ctx context.Context, fd int32) (*SockAddr, error) {
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

	return &SockAddr{
		Fd:         fd,
		LocalAddr:  localAddrPort,
		RemoteAddr: remoteAddrPort,
	}, nil
}

func (e *UringNetEngine) Write(ctx context.Context, fd int32, data []byte) error {
	userData := e.encodeUserData(event.EVENT_TYPE_WRITE, fd)
	e.uring.Write(fd, data, userData)
	slog.DebugContext(ctx, "Submitted write operation", "fd", fd, "dataLength", len(data))
	return nil
}
