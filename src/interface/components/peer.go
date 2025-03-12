package components

import (
	"bytes"
	"errors"
	gen "github.com/touka-aoi/low-level-server/gen/proto"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/encoding/protodelim"
	"io"
	"net/netip"
)

const MaxOSFileDescriptor = 1 << 20
const maxBufferSize = 1 << 20
const maxBufferSizeMask = maxBufferSize - 1

type Peer struct {
	Fd        int32
	Ip        netip.AddrPort
	Buffer    []byte
	WriteChan chan *Peer
	head      int
	tail      int
}

func NewReadPeer(fd int32, ip netip.AddrPort) *Peer {
	return &Peer{
		Fd:        fd,
		Ip:        ip,
		WriteChan: make(chan *Peer, 1),
		Buffer:    make([]byte, maxBufferSize),
	}
}

func (p *Peer) Write(b []byte) (int, error) {
	if p.head > p.tail {
		if len(b) > p.head-p.tail {
			return 0, ErrWouldBlock
		}
	} else {
		if len(b) > maxBufferSize-(p.tail-p.head) {
			return 0, ErrWouldBlock
		}
	}
	if (p.tail)+len(b) >= maxBufferSize {
		copy(p.Buffer[p.tail:], b[:maxBufferSize-p.tail])
		copy(p.Buffer[0:], b[maxBufferSize-p.tail:])
		p.tail = len(b) - (maxBufferSize - p.tail&maxBufferSizeMask)
	} else {
		copy(p.Buffer[p.tail:], b)
		p.tail = (p.tail & maxBufferSizeMask) + len(b)
	}
	return len(b), nil
}

func (p *Peer) ReadMessage() (*gen.Envelope, error) {
	if len(p.Buffer) == 0 {
		return nil, ErrWouldBlock
	}
	var message gen.Envelope
	err := protodelim.UnmarshalFrom(bytes.NewReader(p.Buffer), &message)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return nil, ErrWouldBlock
		}
		return nil, err
	}
	return &message, nil
}

func (p *Peer) Flush() error {
	return nil
}

func (p *Peer) Close() error {
	err := unix.Close(int(p.Fd))
	if err != nil {
		return err
	}
	p.WriteChan = nil
	return nil
}
