package middleware

import "github.com/touka-aoi/low-level-server/core/engine"

type Context struct {
	Data     []byte
	Request  interface{}
	Response []byte
	Fd       int32
	Metadata map[string]interface{}
	Peer     engine.Peer
}

type NextFunc func(*Context) error
type MiddlewareFunc func(*Context, NextFunc) error

type Pipeline struct {
	middlewares []MiddlewareFunc
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		middlewares: make([]MiddlewareFunc, 0),
	}
}

func (p *Pipeline) Use(middleware MiddlewareFunc) *Pipeline {
	p.middlewares = append(p.middlewares, middleware)
	return p
}

func (p *Pipeline) Execute(ctx *Context) error {
	return p.executeMiddleware(0, ctx)
}

func (p *Pipeline) executeMiddleware(index int, ctx *Context) error {
	if index >= len(p.middlewares) {
		return nil
	}

	next := func(ctx *Context) error {
		return p.executeMiddleware(index+1, ctx)
	}

	return p.middlewares[index](ctx, next)
}

func NewContext(data []byte, fd int32, peer engine.Peer) *Context {
	return &Context{
		Data:     data,
		Fd:       fd,
		Peer:     peer,
		Metadata: make(map[string]interface{}),
	}
}
