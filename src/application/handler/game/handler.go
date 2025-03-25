package game

import "context"

type GameHandler interface {
	DataHandler(ctx context.Context, data []byte) (interface{}, error)
	Tick(ctx context.Context)
}

func HandlerFromFunc(
	dataHandler func(ctx context.Context, data []byte) (interface{}, error),
	tickHandler func(ctx context.Context),
) GameHandler {
	return &Handler{
		dataHandler: dataHandler,
		tickHandler: tickHandler,
	}
}

type Handler struct {
	dataHandler func(ctx context.Context, data []byte) (interface{}, error)
	tickHandler func(ctx context.Context)
}

var _ GameHandler = (*Handler)(nil)

func (h *Handler) DataHandler(ctx context.Context, data []byte) (interface{}, error) {
	_, _ = h.dataHandler(ctx, data)
}

func (h *Handler) Tick(ctx context.Context) {
	h.tickHandler(ctx)
}
