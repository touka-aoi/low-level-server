package live

import (
	"context"
	"log/slog"

	"github.com/touka-aoi/low-level-server/transport/streaming"
)

type LiveApp struct {
	receiveChannel <-chan streaming.FrameMessage
	config         LiveConfig
}

type LiveConfig struct {
	Fps int
}

func NewLiveApp(config LiveConfig, chr <-chan streaming.FrameMessage) *LiveApp {
	return &LiveApp{
		receiveChannel: chr,
		config:         config,
	}
}

func (l *LiveApp) Run(ctx context.Context) {
	go l.processLoop(ctx)
}

func (l *LiveApp) processLoop(ctx context.Context) {
	slog.InfoContext(ctx, "LiveApp started")
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case frame := <-l.receiveChannel:
			l.processNextFrame(ctx, frame)
		}
	}
	slog.InfoContext(ctx, "LiveApp stopped")
}

func (l *LiveApp) processNextFrame(ctx context.Context, frame streaming.FrameMessage) {
	slog.DebugContext(ctx, "process Next Frame", "type", frame.Frame.Type, "payload", frame.Frame.Payload)
}
