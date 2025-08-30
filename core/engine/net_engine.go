//go:build linux

package engine

import (
	"context"
	"net/netip"

	"github.com/google/uuid"
	"github.com/touka-aoi/low-level-server/core/buffer"
	"github.com/touka-aoi/low-level-server/core/event"
)

type NetEvent struct {
	EventType  event.EventType
	Fd         int32
	Data       []byte
	RemoteAddr netip.AddrPort
}

type Peer struct {
	SessionID  string
	Fd         int32
	LocalAddr  netip.AddrPort
	RemoteAddr netip.AddrPort
	buf        []byte
	ring       *buffer.RingBuffer
}

func NewPeer(fd int32, localAddr netip.AddrPort, remoteAddr netip.AddrPort) *Peer {
	sessionID := uuid.NewString()
	return &Peer{
		SessionID:  sessionID,
		Fd:         fd,
		LocalAddr:  localAddr,
		RemoteAddr: remoteAddr,
		ring:       buffer.NewRingBuffer(4096),
	}
}

func (p *Peer) Feed(data []byte) error {
	_, err := p.ring.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (p *Peer) Advance(n int) {
	p.ring.Consume(n)
}

func (p *Peer) Peek(b []byte) bool {
	return p.ring.Peek(b)
}

func (p *Peer) View(n int) ([]byte, []byte, bool) {
	return p.ring.View(n)
}

type NetEngine interface {
	Accept(ctx context.Context, listener Listener) error
	RecvFrom(ctx context.Context, listener Listener) error
	ReceiveData(ctx context.Context) ([]*NetEvent, error)
	RegisterRead(ctx context.Context, peer *Peer) error
	Write(ctx context.Context, fd int32, data []byte) error
	PrepareClose() error
	GetSockAddr(ctx context.Context, fd int32) (*SockAddr, error)
	Close() error
}
