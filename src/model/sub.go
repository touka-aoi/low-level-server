package model

import "context"

type AppInterface interface {
	OnUpdate()
	OnStart()
}

type dataHandler interface {
	OnRead(ctx context.Context, peer *Peer, buffer []byte) error
	OnHandle(ctx context.Context, peer *Peer, buffer []byte) error
}

type Peer struct {
	Fd     int32
	Buffer []byte
}

// サービスでもあり、データハンドラでもある
// うまく動くのか？
type ioURingHandler struct {
	OnUpdate func()
	OnStart  func(ctx context.Context)
	OnRead   func(ctx context.Context, peer *Peer, buffer []byte) error
	OnHandle func(ctx context.Context, peer *Peer, buffer []byte) error
}

func NewIoURingHandler(
	onRead func(ctx context.Context, peer *Peer, buffer []byte) error,
	onHandle func(ctx context.Context, peer *Peer, buffer []byte) error) *ioURingHandler {
	return &ioURingHandler{
		OnRead:   onRead,
		OnHandle: onHandle,
	}
}

func (h *ioURingHandler) OnHandle(ctx context.Context, peer *Peer, buffer []byte) error {
	return h.onHandle(ctx, peer, buffer)
}

func (h *ioURingHandler) OnRead(ctx context.Context, peer *Peer, buffer []byte) error {
	// ioURINGが持つデータ処理を行う
	return h.onRead(ctx, peer, buffer)
}

type Application struct {
	service []*AppInterface
}

func NewApplication() *Application {

	return &Application{
		service: service,
	}
}

func (a *Application) RunContext(ctx context.Context) {
	for _, s := range a.service {
		s.Onstart(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			for _, s := range a.service {
				s.OnUpdate()
			}
			return
		}
	}
}
