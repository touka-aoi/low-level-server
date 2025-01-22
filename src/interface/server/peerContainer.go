package server

type PeerContainer [maxOSFileDescriptor]*Peer

func NewPeerContainer() *PeerContainer {
	return &PeerContainer{}
}

func (p *PeerContainer) GetPeer(fd int32) *Peer {
	return p[fd&maxOSFileDescriptor]
}

func (p *PeerContainer) RegisterPeer(fd int32, peer *Peer) {
	p[fd&maxOSFileDescriptor] = peer
}

func (p *PeerContainer) UnregisterPeer(fd int32) {
	peer := p[fd&maxOSFileDescriptor]
	if peer == nil {
		return
	}
	peer.Close()
	p[fd&maxOSFileDescriptor] = nil
}
