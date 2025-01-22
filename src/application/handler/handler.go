package handler

import (
	"context"
	"github.com/touka-aoi/low-level-server/interface/components"
)

type Handler interface {
	OnRead(ctx context.Context, peer *components.Peer, buffer []byte) error
}
