//go:build linux

package engine

import (
	"context"

	"github.com/touka-aoi/low-level-server/internal/event"
	"github.com/touka-aoi/low-level-server/internal/io"
	"golang.org/x/exp/slog"
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
	e.uring.AccpetMultishot(listener.Fd())
	return nil
}

func (e *UringNetEngine) ReceiveData(ctx context.Context) ([]*NetEvent, error) {
	cqEvent, err := e.uring.WaitEvent()
	if err != nil {
		return nil, err
	}
	if cqEvent == nil {
		return nil, nil
	}

	if cqEvent.UserData == 0 {
		slog.WarnContext(ctx, "UserData is nil")
		return nil, nil
	}

	userData := e.decodeUserData(cqEvent.UserData)

	netEvents := make([]*NetEvent, 0)

	return nil, nil
}

func (e *UringNetEngine) encodeUserData() {

}

func (e *UringNetEngine) decodeUserData(data uint64) *userData {
	return &userData{
		eventType: event.EventType(data >> 32),
		fd:        int32(data & 0xFFFFFFFF),
	}
}
