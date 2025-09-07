package peer

import "github.com/touka-aoi/low-level-server/core/buffer"

type RingReader struct {
	ring *buffer.RingBuffer
}

func NewRingReader(size int) *RingReader {
	if size <= 0 {
		size = 4096
	}
	return &RingReader{
		ring: buffer.NewRingBuffer(size),
	}
}

// NOTE: FeedじゃなくてReadにしてもいいなぁと思っている
func (p *RingReader) Feed(data []byte) error {
	_, err := p.ring.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (p *RingReader) Advance(n int) {
	p.ring.Advance(n)
}

func (p *RingReader) Peek(b []byte) bool {
	return p.ring.Peek(b)
}

func (p *RingReader) View(n int) ([]byte, []byte, bool) {
	return p.ring.View(n)
}
