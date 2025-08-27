package buffer

import "errors"

var (
	ErrBufferFull = errors.New("buffer is full")
)

type RingBuffer struct {
	buf  []byte
	mask uint64
	head uint64
	tail uint64
}

func nextPow2(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return n + 1
}

func NewRingBuffer(size int) *RingBuffer {
	capacity := nextPow2(size)
	return &RingBuffer{
		buf:  make([]byte, capacity),
		mask: uint64(capacity) - 1,
	}
}

func (r *RingBuffer) length() int {
	return int(r.tail - r.head)
}

func (r *RingBuffer) capacity() int {
	return len(r.buf)
}

func (r *RingBuffer) free() int {
	return r.capacity() - r.length()
}

func (r *RingBuffer) advance(n int) {
	r.tail += uint64(n)
}

func (r *RingBuffer) Write(b []byte) (int, error) {
	if len(b) > r.free() {
		return 0, ErrBufferFull
	}
	i := int(r.tail & r.mask)
	n1 := copy(r.buf[i:], b)
	n2 := copy(r.buf, b[n1:])
	r.head += uint64(n1 + n2)
	return n1 + n2, nil
}

func (r *RingBuffer) Peek(dst []byte) bool {
	if len(dst) > r.length() {
		return false
	}
	i := int(r.head & r.mask)
	n1 := copy(dst, r.buf[i:])
	if n1 < len(dst) {
		copy(dst[n1:], r.buf[:len(dst)-n1])
	}
	return true
}

func (r *RingBuffer) View(n int) (a, b []byte, ok bool) {
	if n > r.length() {
		return nil, nil, false
	}
	i := int(r.tail & r.mask)
	if i+n <= len(r.buf) {
		return r.buf[i : i+n : i+n], nil, true
	}
	n1 := len(r.buf) - i
	return r.buf[i:len(r.buf):len(r.buf)], r.buf[: n-n1 : n-n1], true
}

func (r *RingBuffer) Consume(n int) {
	r.advance(n)
}
