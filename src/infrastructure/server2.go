package infrastructure

import (
	"context"
	"sync"
	"sync/atomic"

	netEngine "github.com/touka-aoi/low-level-server/internal/engine"
)

const (
	ServerInitialized = iota
	ServerStarted
	ServerPrepareClosing
	ServerClosed
)

// --------------------------------------------------
// ここから下はServer2の実装
// --------------------------------------------------

type Server2 struct {
	listener  *netEngine.Listener
	network   string // iouring, epoll, kqueeeの実装が入ってきても許されるIFと交換
	wg        sync.WaitGroup
	state     atomic.Int32
	once      sync.Once
	netEngine netEngine.NetEngine
}

func NewServer2() *Server2 {
	return &Server2{}
}

func (s *Server2) Listen() error {
	listener, err := netEngine.Listen("tcp", "localhost:8080", 1000)
	if err != nil {
		return err
	}
	s.listener = &listener
	return nil
}

func (s *Server2) Serve(ctx context.Context) {
	s.netEngine.Accept(ctx, s.listener)
	s.state.Store(ServerStarted)
	s.netEngine.Up(ctx)

	for {
		if s.state.Load() == ServerPrepareClosing {
			s.netEngine.PrepareClose()
		}

		data, err := s.netEngine.ReceiveData()
		if err != nil {
			// handle error
		}
		s.handleEvent(event, data)

	}

	if s.state.Load() <= ServerPrepareClosing {
		s.state.Store(ServerClosed)
	}
}

// interfaceを満たしてたら入れれるようにしたらよさそう
func (s *Server2) AddCodec(c any) {
	s.ReadCodec = c
	s.WriteCodec = c
}

func (s *Server2) AddHandler(h any) {
	s.handler = h
}

// ここでフレーミングを吸収する
func (s *Server2) handleData(data []byte) {
	// ここわからん
	event, err := s.ReadCodec.Decode(event)
	if err != nil {
		// handle error

	}
	s.handleEvent(event)
}

// ここに来る頃にはすでにパースされている
func (s *Server2) handleEvent(event any) {
	switch event.(type) {
	case EVENT_TYPE_ACCEPT:
		s.handleAccept(event)
	case EVENT_TYPE_WRITE:
		// 何もしない
	case EVENT_TYPE_READ:
		s.handler.handleData(event)
	case EVENT_TYPE_CLOSE:
	}
}
