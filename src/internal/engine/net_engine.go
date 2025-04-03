//go:build linux

package engine

import (
	"context"

	"github.com/touka-aoi/low-level-server/internal"
)

type NetEvent struct {
	EventType internal.EventType
	Fd        int32
	Data      []byte
}

type NetEngine interface {
	Accept(ctx context.Context, listener Listener) error
	Up(ctx context.Context) error
	ReceiveData(ctx context.Context) ([]*NetEvent, error)
	PrepareClose() error
	Close() error
}
