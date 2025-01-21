package server

import (
	"golang.org/x/sys/unix"
	"net/netip"
)

const maxOSFileDescriptor = 1 << 20

type Peer struct {
	Fd        int32
	Ip        netip.AddrPort
	Buffer    []byte
	writeChan chan *Peer
}

func (p *Peer) Read(b []byte) (int, error) {
	copy(b, p.Buffer)
	return len(p.Buffer), nil
}

func (p *Peer) Write(b []byte) (int, error) {
	p.Buffer = b
	p.writeChan <- p
	return len(b), nil
}

func (p *Peer) Close() error {
	//close(p.writeChan)
	err := unix.Close(int(p.Fd))
	if err != nil {
		return err
	}
	return nil
}

type PeerAcceptor [maxOSFileDescriptor]*Peer

func NewPeerAcceptor() *PeerAcceptor {
	return &PeerAcceptor{}
}

func (p *PeerAcceptor) GetPeer(fd int32) *Peer {
	return p[fd&maxOSFileDescriptor]
}

func (p *PeerAcceptor) RegisterPeer(fd int32, peer *Peer) {
	p[fd&maxOSFileDescriptor] = peer
}

func (p *PeerAcceptor) UnregisterPeer(fd int32) {
	peer := p[fd&maxOSFileDescriptor]
	if peer == nil {
		return
	}
	peer.Close()
	p[fd&maxOSFileDescriptor] = nil
}
