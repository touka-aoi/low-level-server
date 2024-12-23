package server

import (
	"net/netip"
)

type Peer struct {
	Fd     int32
	Ip     netip.AddrPort
	Buffer []byte
}

const maxOSFileDescriptor = 1 << 20

type PeerAcceptor [maxOSFileDescriptor]*Peer

func NewPeerAcceptor() *PeerAcceptor {
	return &PeerAcceptor{}
}

func (p *PeerAcceptor) GetPeer(fd int32) *Peer {
	return p[fd&maxOSFileDescriptor]
}

func (p *PeerAcceptor) SetPeer(fd int32, peer *Peer) {
	p[fd&maxOSFileDescriptor] = peer
}
