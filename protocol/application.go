package protocol

import (
	"context"

	"github.com/touka-aoi/low-level-server/core/engine"
)

type Application interface {
	OnConnect(ctx context.Context, peer *engine.Peer) error
	OnData(ctx context.Context, peer *engine.Peer, data []byte) ([]byte, error)
	OnDisconnect(ctx context.Context, peer *engine.Peer) error
}
