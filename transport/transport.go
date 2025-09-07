package transport

import (
	"context"

	"github.com/touka-aoi/low-level-server/server/peer"
)

type Transport interface {
	OnConnect(ctx context.Context, peer *peer.Peer) error
	OnData(ctx context.Context, peer *peer.Peer, data []byte) ([]byte, error)
	OnDisconnect(ctx context.Context, peer *peer.Peer) error
}
