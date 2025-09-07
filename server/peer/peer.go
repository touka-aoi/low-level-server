package peer

import (
	"net/netip"
	"sync/atomic"

	"github.com/google/uuid"
)

type Peer struct {
	SessionID  string
	fd         int32
	localAddr  netip.AddrPort
	remoteAddr netip.AddrPort
	status     atomic.Int32
	LastActive atomic.Int64

	Reader *RingReader
	Writer *RingWriter
}

func NewPeer(fd int32, localAddr netip.AddrPort, remoteAddr netip.AddrPort) *Peer {
	sessionID := uuid.NewString()
	return &Peer{
		SessionID:  sessionID,
		fd:         fd,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
		Reader:     NewRingReader(4096),
		Writer:     NewRingWriter(4096),
	}
}

func (p *Peer) Fd() int32 {
	return p.fd
}

func (p *Peer) LocalAddr() netip.AddrPort {
	return p.localAddr
}

func (p *Peer) RemoteAddr() netip.AddrPort {
	return p.remoteAddr
}

func (p *Peer) Status() string {
	s := p.status.Load()
	return ConnState(s).String()
}
