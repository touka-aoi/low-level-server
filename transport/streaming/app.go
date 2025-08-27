package streaming

import (
	"context"
	"encoding/binary"
	"errors"
	"log/slog"

	"github.com/touka-aoi/low-level-server/core/engine"
	"github.com/touka-aoi/low-level-server/transport"
)

type LiveStreamingApp struct {
}

func (l LiveStreamingApp) OnConnect(ctx context.Context, peer *engine.Peer) error {
	//TODO implement me
	// 認証情報の検証や接続管理などをここに入れたい
	panic("implement me")
}

func (l LiveStreamingApp) OnData(ctx context.Context, peer *engine.Peer, data []byte) ([]byte, error) {
	peer.Feed(data)
	for {
		header, ok := peer.Peek(HeaderSize)
		if !ok {
			return nil, errors.New("invalid header")
		}
		if binary.BigEndian.Uint16(header[0:2]) != MagicNumber {
			return nil, errors.New("ErrBadMagic")
		}
		length := binary.BigEndian.Uint32(header[3:7])
		total := HeaderSize + int(length)
		// if total > maxFrameSize {...}
		full, ok := peer.Peek(total)
		if !ok {
			return nil, errors.New("ErrNeedMore")
		}
		peer.Advance(total)
		frame, err := ParseFrame(full)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to parse frame", "error", err)
		}
		l.processFrame(ctx, frame)
	}
}

func (l *LiveStreamingApp) processFrame(ctx context.Context, frame *Frame) {
	switch frame.Type {
	case TYPE_DATA:
		slog.InfoContext(ctx, "Received data frame", "length", len(frame.Payload))
	case TYPE_CONTROL:
		slog.InfoContext(ctx, "Received control frame", "length", len(frame.Payload))
	case TYPE_HEARTBEAT:
		slog.InfoContext(ctx, "Received heartbeat frame")
	}
}

func (l LiveStreamingApp) OnDisconnect(ctx context.Context, peer *engine.Peer) error {
	//TODO implement me
	// 接続管理を入れる
	panic("implement me")
}

func (l LiveStreamingApp) handleControl() {
}

func (l LiveStreamingApp) handleData() {
}

func (l LiveStreamingApp) handleHeartbeat() {
}

var _ transport.Transport = (*LiveStreamingApp)(nil)

func NewLiveStreamingApp() *LiveStreamingApp {
	return &LiveStreamingApp{}
}
