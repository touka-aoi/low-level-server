package handler

import (
	"context"
	"log/slog"
	"slices"

	"github.com/touka-aoi/low-level-server/core/engine"
	"github.com/touka-aoi/low-level-server/core/event"
)

const (
	maxConnections = 65535
)

type SessionManager struct {
	engine      engine.NetEngine
	connections map[int32]engine.Peer
}

func NewSessionManager(netEngine engine.NetEngine) *SessionManager {
	return &SessionManager{
		engine:      netEngine,
		connections: make(map[int32]engine.Peer),
	}
}

func (sm *SessionManager) Serve(ctx context.Context) {
	// ちょっとwaitする必要があるなぁとおもいつつ、、、
	for {
		netEvents, err := sm.engine.ReceiveData(ctx)
		if err != nil {
			// エラー処理
			continue
		}

		for NetEvent := range slices.Values(netEvents) {
			switch NetEvent.EventType {
			case event.EVENT_TYPE_ACCEPT:
				sm.handleAccept(ctx, NetEvent)
			// case event.EVENT_TYPE_READ:
			// 	sm.handleRead(NetEvent)
			// case event.EVENT_TYPE_WRITE:
			// 	sm.handleWrite(NetEvent)
			default:
				// 未知のイベントタイプの処理
			}
		}
	}
}

func (sm *SessionManager) handleAccept(ctx context.Context, event *engine.NetEvent) {
	newFd := event.Fd
	if newFd < 0 {
		slog.WarnContext(ctx, "Invalid file descriptor for new connection", "fd", newFd)
		return
	}

	peer, err := sm.engine.GetPeerName(ctx, newFd)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get peer name", "fd", newFd, "error", err)
		return
	}

	slog.DebugContext(ctx, "Accepted new connection", "fd", newFd, "localAddr", peer.LocalAddr, "remoteAddr", peer.RemoteAddr)

	sm.connections[newFd] = *peer

	// 新しい接続に対してREAD操作を登録
	if err := sm.engine.RegisterRead(ctx, peer); err != nil {
		slog.ErrorContext(ctx, "Failed to register read operation", "fd", newFd, "error", err)
		delete(sm.connections, newFd)
		return
	}
}
