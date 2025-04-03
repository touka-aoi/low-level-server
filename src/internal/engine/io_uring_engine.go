//go:build linux

package engine

import "github.com/touka-aoi/low-level-server/internal/io"

type UringNetEngine struct {
	uring *io.Uring
}

func NewUringNetEngine() *UringNetEngine {
	uring := io.CreateUring(4096)
	return &UringNetEngine{
		uring: uring,
	}
}

func (e *UringNetEngine) ReceiveData() ([]*NetEvent, error) {

	return nil, nil
}
