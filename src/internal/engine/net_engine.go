//go:build linux

package engine

import (
	"context"

	"github.com/touka-aoi/low-level-server/internal/event"
)

type NetEvent struct {
	EventType event.EventType
	Fd        int32
	Data      []byte
}

type NetEngine interface {
	Accept(ctx context.Context, listener Listener) error
	ReceiveData(ctx context.Context) ([]*NetEvent, error)
	PrepareClose() error
	Close() error
}
