//go:build linux

package engine

import (
	"context"

	"github.com/touka-aoi/low-level-server/internal/event"
	"github.com/touka-aoi/low-level-server/internal/io"
)

type userData struct {
	eventType event.EventType
	fd        int32
}

type UringNetEngine struct {
	uring *io.Uring
}

func NewUringNetEngine() *UringNetEngine {
	uring := io.CreateUring(4096)
	return &UringNetEngine{
		uring: uring,
	}
}

func (e *UringNetEngine) Accept(ctx context.Context, listener Listener) error {
	op := e.uring.AccpetMultishot(listener.Fd(), e.encodeUserData(event.EVENT_TYPE_ACCEPT, listener.Fd()))
	e.uring.Submit(op)
	return nil
}

func (e *UringNetEngine) ReceiveData(ctx context.Context) ([]*NetEvent, error) {
	// seen cqe
	cqEvent, err := e.uring.PeekBatchEvents(1)
	if err != nil {
		return nil, err
	}

	for _, event := range cqEvent {
		if event.Res < 0 {

		}

	}

	// advance cqe

	return nil, nil
}

func (e *UringNetEngine) handleEvent() error {
	return nil
}

func (e *UringNetEngine) PrepareClose() error {
	return nil
}

func (e *UringNetEngine) Close() error {
	return e.uring.Close()
}

func (e *UringNetEngine) encodeUserData(ev event.EventType, fd int32) uint64 {
	userData := uint64(ev)<<32 | uint64(fd)
	return userData
}

func (e *UringNetEngine) decodeUserData(data uint64) *userData {
	return &userData{
		eventType: event.EventType(data >> 32),
		fd:        int32(data & 0xFFFFFFFF),
	}
}
