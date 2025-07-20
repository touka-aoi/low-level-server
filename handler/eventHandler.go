package handler

import (
	"context"
	"log/slog"
	"slices"

	"github.com/touka-aoi/low-level-server/core/engine"
	"github.com/touka-aoi/low-level-server/core/event"
	"github.com/touka-aoi/low-level-server/middleware"
)

const (
	maxConnections = 65535
)

type SessionManager struct {
	engine      engine.NetEngine
	connections map[int32]engine.Peer
	pipeline    *middleware.Pipeline
}

func NewSessionManager(netEngine engine.NetEngine, pipeline *middleware.Pipeline) *SessionManager {
	return &SessionManager{
		engine:      netEngine,
		connections: make(map[int32]engine.Peer),
		pipeline:    pipeline,
	}
}

func (sm *SessionManager) Serve(ctx context.Context) {
	// ちょっとwaitする必要があるなぁとおもいつつ、、、
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "Session manager shutting down")
			return
		default:
			netEvents, err := sm.engine.ReceiveData(ctx)
			if err != nil {
				// エラー処理
				continue
			}

			for NetEvent := range slices.Values(netEvents) {
				switch NetEvent.EventType {
				case event.EVENT_TYPE_ACCEPT:
					sm.handleAccept(ctx, NetEvent)
				case event.EVENT_TYPE_READ:
					sm.handleRead(NetEvent)
				// case event.EVENT_TYPE_WRITE:
				// 	sm.handleWrite(NetEvent)
				default:
					// 未知のイベントタイプの処理
				}
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

func (sm *SessionManager) handleRead(event *engine.NetEvent) {
	fd := event.Fd
	data := event.Data

	if fd < 0 {
		slog.Warn("Invalid file descriptor for read event", "fd", fd)
		return
	}

	if len(data) == 0 {
		slog.Warn("Received empty data for read event", "fd", fd)
		return
	}

	slog.Debug("Received data from peer", "fd", fd, "dataLength", len(data), "data", string(data))

	peer := sm.connections[fd]
	if peer.Fd == 0 {
		slog.Warn("Peer not found for read event", "fd", fd)
		return
	}

	ctx := middleware.NewContext(data, fd, peer)
	if err := sm.pipeline.Execute(ctx); err != nil {
		slog.Error("Pipeline execution failed", "fd", fd, "error", err)
		return
	}

	if len(ctx.Response) > 0 {
		sm.sendResponse(fd, ctx.Response)
	}
}

func (sm *SessionManager) sendResponse(fd int32, response []byte) {
	slog.Debug("Sending response", "fd", fd, "responseLength", len(response))
	if err := sm.engine.Write(context.Background(), fd, response); err != nil {
		slog.Error("Failed to send response", "fd", fd, "error", err)
	}
}
