package components

import (
	"golang.org/x/sys/unix"
	"net/netip"
)

const MaxOSFileDescriptor = 1 << 20

type Peer struct {
	Fd        int32
	Ip        netip.AddrPort
	Buffer    []byte
	WriteChan chan *Peer
}

func (p *Peer) Read(b []byte) (int, error) {
	copy(b, p.Buffer)
	return len(p.Buffer), nil
}

func (p *Peer) Write(b []byte) (int, error) {
	p.Buffer = b
	p.WriteChan <- p
	return len(b), nil
}

func (p *Peer) Close() error {
	err := unix.Close(int(p.Fd))
	if err != nil {
		return err
	}
	p.WriteChan = nil
	return nil
}
