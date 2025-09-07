//go:build linux

package engine

import (
	"context"
)

type NetEngine interface {
	Accept(ctx context.Context, listener Listener) error
	CancelAccept(ctx context.Context, listener Listener) error
	RecvFrom(ctx context.Context, listener Listener) error
	ReceiveData(ctx context.Context) ([]*NetEvent, error)
	WaitEvent() error
	RegisterRead(ctx context.Context, fd int32) error
	Write(ctx context.Context, fd int32, data []byte) error
	PrepareClose() error
	GetSockAddr(ctx context.Context, fd int32) (*SockAddr, error)
	ClosePeer(ctx context.Context, fd int32) error
	Close() error
}
