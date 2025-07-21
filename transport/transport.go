package transport

import (
	"context"

	"github.com/touka-aoi/low-level-server/core/engine"
)

type Transport interface {
	OnConnect(ctx context.Context, peer *engine.Peer) error
	OnData(ctx context.Context, peer *engine.Peer, data []byte) ([]byte, error)
	OnDisconnect(ctx context.Context, peer *engine.Peer) error
}
