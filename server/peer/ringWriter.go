package peer

import (
	"github.com/touka-aoi/low-level-server/core/buffer"
	toukaerrors "github.com/touka-aoi/low-level-server/core/errors"
)

type RingWriter struct {
	ring       *buffer.RingBuffer
	queuedByte int
}

func NewRingWriter(size int) *RingWriter {
	if size <= 0 {
		size = 4096
	}
	return &RingWriter{
		ring: buffer.NewRingBuffer(size),
	}
}

// NOTE: FeedじゃなくてReadにしてもいいなぁと思っている
func (p *RingWriter) Feed(data []byte) error {
	_, err := p.ring.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (p *RingWriter) Advance(n int) {
	p.queuedByte -= n
	p.ring.Advance(n)
}

func (p *RingWriter) Advance2(n int) {
	p.queuedByte += n
}

func (p *RingWriter) QueuedByte() int {
	return p.queuedByte
}

func (p *RingWriter) Peek(b []byte) bool {
	return p.ring.Peek(b)
}

func (p *RingWriter) PeekOut() []byte {
	return p.ring.PeekOut()
}

func (p *RingWriter) View(n int) ([]byte, []byte, bool) {
	return p.ring.View(n)
}

func (p *RingWriter) Length() int {
	return p.ring.Length()
}

func (p *RingWriter) Write(b []byte) (int, error) {
	if len(b) > p.ring.Free() {
		return 0, toukaerrors.ErrWouldBlock
	}
	_, err := p.ring.Write(b)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}
