package internal

import (
	"context"
	"github.com/touka-aoi/low-level-server/interface/server"
	"sync"
)

type Server2 struct {
	cfg      server.ServerConfig
	listener *Socket
	network  string // iouring, epoll, kqueeeの実装が入ってきても許されるIFと交換
	wg       sync.WaitGroup
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

func (s *Server2) Serve(ctx context.Context) {
	s.listener.Accept()

	wg := sync.WaitGroup{}
	go func() {
		defer wg.Done()
		for {
			if s.state.Load() == ServerPrepareClosing {
				// Acceptを止める命令を出す
				// 一度だけ出したいけど、どうするかここは無限にループするので
			}
			// データを受けとる
			data := s.listener.ReceiveData() // receivedata内でサーバーステータスの情報を処理するしか...
			// でもlistenerは別構造体か
			s.handleEvent(data) // handleEventの中でイベントごとにhandler.handleData2()<-名前募集を入れたらいいか

			// がなくなったら終了させたい
			if ctx.Done() {
				break
			}

			// Acceptを受け取る

			// ↑イベントをハンドリングしている

			// データをハンドリングする
			// フレーミングのためにPeerにする
			// 論理処理はイベントキューに詰める
			// 処理を書いてるんじゃなくてIOを書いてる <- これ

			// データを書き込む

			event := s.listener.receiveEvent()
			data := s.handleEvent(event)
			s.handler.handleData(data)
		}
	}()

	if s.state.Load() <= ServerPrepareClosing {
		s.state.Store(ServerClosed)
	}

	s.wg.Wait()
}
