package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/touka-aoi/low-level-server/core/engine"
	toukaerrors "github.com/touka-aoi/low-level-server/core/errors"
	"github.com/touka-aoi/low-level-server/core/event"
	"github.com/touka-aoi/low-level-server/middleware"
	"github.com/touka-aoi/low-level-server/server/peer"
	"github.com/touka-aoi/low-level-server/transport"
)

const (
	maxConnections = 65535
)

type NetworkServerConfig struct {
	Protocol string
	Address  string
	Port     int
}

type SrvStatus int

const (
	Running SrvStatus = iota
	Draining
	Stopped
)

var stateName = map[SrvStatus]string{
	Running:  "running",
	Draining: "draining",
	Stopped:  "stopped",
}

func (s SrvStatus) String() string {
	return stateName[s]
}

type NetworkServer struct {
	engine      engine.NetEngine
	listener    engine.Listener
	config      NetworkServerConfig
	connections map[int32]*peer.Peer
	pipeline    *middleware.Pipeline
	app         transport.Transport
	status      SrvStatus
}

func NewNetworkServer(netEngine engine.NetEngine, config NetworkServerConfig, pipeline *middleware.Pipeline, app transport.Transport) *NetworkServer {
	return &NetworkServer{
		engine:      netEngine,
		config:      config,
		connections: make(map[int32]*peer.Peer),
		pipeline:    pipeline,
		app:         app,
		//oreore:      oreore, オレオレも所有してオレオレする必要がありそう
	}
}

func (ns *NetworkServer) Serve(ctx context.Context) {
	ns.status = Running
	var drainingDeadline time.Time

	go func() {
		select {
		case <-ctx.Done():
			err := ns.engine.PrepareClose()
			if err != nil {
				slog.ErrorContext(ctx, "Failed to prepare close", "error", err)
			}
		}
	}()

	for {
		netEvents, recvError := ns.engine.ReceiveData(ctx)
		if recvError != nil && !errors.Is(recvError, toukaerrors.ErrWouldBlock) {
			slog.ErrorContext(ctx, "Failed to receive data", "error", recvError)
			continue
		}

		for NetEvent := range slices.Values(netEvents) {
			switch NetEvent.EventType {
			case event.EVENT_TYPE_ACCEPT:
				ns.handleAccept(ctx, NetEvent)
			case event.EVENT_TYPE_READ:
				ns.handleRead(ctx, NetEvent)
			//case event.EVENT_TYPE_WRITE:
			//	ns.handleWrite(NetEvent)
			case event.EVENT_TYPE_RECVMSG:
				slog.DebugContext(ctx, "Received data from peer", "fd", NetEvent.Fd, "dataLength", len(NetEvent.Data), "data", string(NetEvent.Data))
			default:
				// 未知のイベントタイプの処理
			}
		}

		if ns.status == Running && ctx.Err() != nil {
			ns.status = Draining
			drainingDeadline = time.Now().Add(10 * time.Second)
			err := ns.PrepareClose(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to prepare shutdown", "error", err)
			}
			for _, conn := range ns.connections {
				if conn.Status() == peer.StateIdle.String() {
					if err := ns.app.OnDisconnect(ctx, conn); err != nil {
						slog.ErrorContext(ctx, "Application error", "error", err)
					}
					//TODO: update peer status compare and swap
					err := ns.engine.ClosePeer(ctx, conn.Fd())
					if err != nil {
						slog.WarnContext(ctx, "Failed to close peer", "error", err)
					}
				}
			}
		}

		if ns.status == Draining {
			var unCloseConnections int
			for _, conn := range ns.connections {
				if conn.Status() != peer.StateClosed.String() {
					unCloseConnections++
				}
			}
			//if unCloseConnections == 0 {
			//	ns.status = Stopped
			//	return
			//}
			if time.Now().After(drainingDeadline) {
				ns.status = Stopped
				slog.WarnContext(ctx, "Draining timeout exceeded")
				return
			}
		}

		// checkPeerStatus

		if errors.Is(recvError, toukaerrors.ErrWouldBlock) {
			err := ns.engine.WaitEvent()
			//slog.DebugContext(ctx, "Wait event Done")
			if err != nil {
				slog.ErrorContext(ctx, "Failed to wait event", "error", err)
			}
		}

	}
}

func (ns *NetworkServer) PrepareClose(ctx context.Context) error {
	slog.InfoContext(ctx, "Server Prepare to close")
	if ns.config.Protocol == "tcp" {
		slog.DebugContext(ctx, "Shut Prepare to close")
		err := ns.engine.CancelAccept(ctx, ns.listener)
		if err != nil {
			slog.ErrorContext(context.Background(), "Failed to cancel accept", "error", err)
			return err
		}
	}
	for _, conn := range ns.connections {
		if conn.Status() != peer.StateClosed.String() {
			// 閉じることを送信する
			// オレオレプロトコルここに来れないわ...困っち
		}
	}
	return nil
}

func (ns *NetworkServer) Listen(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", ns.config.Address, ns.config.Port)
	listener, err := engine.Listen(ns.config.Protocol, addr, 1024)
	if err != nil {
		return err
	}
	ns.listener = listener

	switch ns.config.Protocol {
	case "tcp":
		if err := ns.engine.Accept(ctx, listener); err != nil {
			slog.ErrorContext(ctx, "Failed to start accepting connections", "error", err)
			return err
		}
	case "udp":
		if err := ns.engine.RecvFrom(ctx, listener); err != nil {
			slog.ErrorContext(ctx, "Failed to start receiving data", "error", err)
			return err
		}
	}

	slog.Info("Listening on", "address", addr)
	return nil
}

func (ns *NetworkServer) handleAccept(ctx context.Context, event *engine.NetEvent) {
	newFd := event.Fd
	if newFd < 0 {
		slog.WarnContext(ctx, "Invalid file descriptor for new connection", "fd", newFd)
		return
	}

	sockAddr, err := ns.engine.GetSockAddr(ctx, newFd)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get peer name", "fd", newFd, "error", err)
		return
	}
	connPeer := peer.NewPeer(sockAddr.Fd, sockAddr.LocalAddr, sockAddr.RemoteAddr)
	slog.DebugContext(ctx, "Accepted new connection", "fd", newFd, "localAddr", connPeer.LocalAddr, "remoteAddr", connPeer.RemoteAddr)

	ns.connections[newFd] = connPeer

	// Applicationに通知
	if ns.app != nil {
		if err := ns.app.OnConnect(ctx, connPeer); err != nil {
			slog.ErrorContext(ctx, "Application rejected connection", "fd", newFd, "error", err)
			delete(ns.connections, newFd)
			return
		}
	}

	// 新しい接続に対してREAD操作を登録
	if err := ns.engine.RegisterRead(ctx, connPeer.Fd()); err != nil {
		slog.ErrorContext(ctx, "Failed to register read operation", "fd", newFd, "error", err)
		delete(ns.connections, newFd)
		return
	}
}

func (ns *NetworkServer) handleRead(ctx context.Context, event *engine.NetEvent) {
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

	peer, ok := ns.connections[fd]
	if !ok {
		slog.Warn("Peer not found for read event", "fd", fd)
		return
	}

	// ミドルウェア実行（ログ等）
	if ns.pipeline != nil {
		// あんまこの設計良くないな
		ctx := middleware.NewContext(data, fd, peer)
		if err := ns.pipeline.Execute(ctx); err != nil {
			slog.Error("Pipeline execution failed", "fd", fd, "error", err)
			return
		}
	}

	// Applicationに処理を委譲
	if ns.app != nil {
		response, err := ns.app.OnData(ctx, peer, data)
		if err != nil {
			slog.Error("Application error", "fd", fd, "error", err)
			return
		}

		// レスポンスがあれば送信
		if len(response) > 0 {
			if err := ns.engine.Write(ctx, fd, response); err != nil {
				slog.Error("Failed to send response", "fd", fd, "error", err)
			}
		}
	}
}
