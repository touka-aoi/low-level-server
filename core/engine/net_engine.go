//go:build linux

package engine

import (
	"context"
	"net/netip"

	"github.com/touka-aoi/low-level-server/core/event"
)

type NetEvent struct {
	EventType  event.EventType
	Fd         int32
	Data       []byte
	RemoteAddr netip.AddrPort
}

type Peer struct {
	Fd         int32
	LocalAddr  netip.AddrPort
	RemoteAddr netip.AddrPort
}

type NetEngine interface {
	Accept(ctx context.Context, listener Listener) error
	RecvFrom(ctx context.Context, listener Listener) error
	ReceiveData(ctx context.Context) ([]*NetEvent, error)
	RegisterRead(ctx context.Context, peer *Peer) error
	Write(ctx context.Context, fd int32, data []byte) error
	PrepareClose() error
	GetPeerName(ctx context.Context, fd int32) (*Peer, error)
	Close() error
}
