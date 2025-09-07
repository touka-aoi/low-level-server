package streaming

import (
	"context"
	"encoding/binary"
	"errors"
	"log/slog"

	"github.com/touka-aoi/low-level-server/server/peer"
	"github.com/touka-aoi/low-level-server/transport"
	"github.com/touka-aoi/low-level-server/transport/protocol"
)

// これはトランスポート層とアプリケーション層の橋渡しをする
// だのでこれが両方から依存されるはず...
type LiveStreamingApp struct {
	handler protocol.LiveProtocol
	chw     chan<- FrameMessage
}

type FrameMessage struct {
	Frame     *protocol.Frame
	SessionID string
}

func NewLiveStreaming(chw chan<- FrameMessage) *LiveStreamingApp {
	return &LiveStreamingApp{
		chw: chw,
	}
}

func (l *LiveStreamingApp) SetHandler(handler protocol.LiveProtocol) {
	l.handler = handler
}

func (l LiveStreamingApp) OnConnect(ctx context.Context, peer *peer.Peer) error {
	//TODO implement me
	// 認証情報の検証や接続管理などをここに入れたい
	//panic("implement me")
	return nil
}

func (l LiveStreamingApp) OnData(ctx context.Context, peer *peer.Peer, data []byte) ([]byte, error) {
	if err := peer.Reader.Feed(data); err != nil {
		return nil, err
	}
	// 遅延パースの可能性
	for {
		header := make([]byte, protocol.HeaderSize)
		ok := peer.Reader.Peek(header)
		if !ok {
			slog.DebugContext(ctx, "Need more data")
			return nil, errors.New("invalid header")
		}
		if binary.BigEndian.Uint16(header[0:2]) != protocol.MagicNumber {
			slog.DebugContext(ctx, "Invalid magic number", "magic", binary.BigEndian.Uint16(header[0:2]))
			return nil, errors.New("ErrBadMagic")
		}
		length := binary.BigEndian.Uint32(header[3:7])
		total := protocol.HeaderSize + int(length)
		// if total > maxFrameSize {...}
		b := make([]byte, total)
		ok = peer.Reader.Peek(b)
		if !ok {
			return nil, errors.New("ErrNeedMore")
		}
		frame, err := protocol.ParseFrame(b)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to parse frame", "error", err)
		}
		frameMessage := FrameMessage{Frame: frame, SessionID: peer.SessionID}
		slog.DebugContext(ctx, "Received frame", "type", frameMessage.Frame.Type, "payload", frameMessage.Frame.Payload)
		//l.processFrame(ctx, frame)
		if l.chw != nil {
			l.chw <- frameMessage
		}
		peer.Reader.Advance(total)
	}
}

func (l *LiveStreamingApp) processFrame(ctx context.Context, frame *protocol.Frame) {
	switch frame.Type {
	case protocol.TYPE_DATA:
		if l.handler != nil {
			l.handler.ReceiveData()
		}
		slog.InfoContext(ctx, "Received data frame", "length", len(frame.Payload))
	case protocol.TYPE_CONTROL:
		if l.handler != nil {
			l.handler.ReceiveControl()
		}
		slog.InfoContext(ctx, "Received control frame", "length", len(frame.Payload))
	case protocol.TYPE_HEARTBEAT:
		if l.handler != nil {
			l.handler.ReceiveHeartbeat()
		}
		slog.InfoContext(ctx, "Received heartbeat frame")
	}
}

func (l LiveStreamingApp) OnDisconnect(ctx context.Context, peer *peer.Peer) error {
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
