package components

type PeerContainer [MaxOSFileDescriptor]*Peer

func NewPeerContainer() *PeerContainer {
	return &PeerContainer{}
}

func (p *PeerContainer) GetPeer(fd int32) *Peer {
	return p[fd&MaxOSFileDescriptor]
}

func (p *PeerContainer) RegisterPeer(fd int32, peer *Peer) {
	p[fd&MaxOSFileDescriptor] = peer
}

func (p *PeerContainer) UnregisterPeer(fd int32) {
	peer := p[fd&MaxOSFileDescriptor]
	if peer == nil {
		return
	}
	peer.Close()
	p[fd&MaxOSFileDescriptor] = nil
}
