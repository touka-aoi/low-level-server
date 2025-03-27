package internal

import (
	"context"
	"github.com/touka-aoi/low-level-server/interface/server"
	"sync"
	"sync/atomic"
)

const (
	ServerInitialized = iota
	ServerStarted
	ServerPrepareClosing
	ServerClosed
)

type Server2 struct {
	cfg      server.ServerConfig
	listener *Socket
	network  string // iouring, epoll, kqueeeの実装が入ってきても許されるIFと交換
	wg       sync.WaitGroup
	state    atomic.Int32
	once     sync.Once
}

func NewServer2() *Server2 {
	return &Server2{}
}

func (s *Server2) Listen() error {
	listener, err := Listen(s.cfg.Protocol, s.cfg.ExternalAddress, s.cfg.ListenMaxConnection)
	if err != nil {
		return err
	}
	s.listener = listener
	return nil
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
func (s *Server2) handleData(event eventType, data []byte) {
	// peerを取得する
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

func (s *Server2) Serve(ctx context.Context) {
	s.listener.Accept()
	for {
		if s.state.Load() == ServerPrepareClosing {
			s.once.Do(s.listener.PrepareClose())
		}
		// この時点でdecoder, encoderがセットされてないとだめなのか
		peer := s.listener.ReceiveData()
		s.handleData(peer)

		// peerのcloseを待つ感じかな
		if ctx.Done() {
			break
		}
	}

	if s.state.Load() <= ServerPrepareClosing {
		s.state.Store(ServerClosed)
	}
}
