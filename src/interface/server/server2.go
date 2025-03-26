package server

import (
	"context"
	"github.com/touka-aoi/low-level-server/internal"
	"log/slog"
)

type ServerConfig struct {
	ExternalAddress     string
	Protocol            string
	ListenMaxConnection int
}

type Server struct {
	cfg            ServerConfig
	socket         *internal.Socket
	internalServer *internal.Server2
}

func NewServer2() *Server {
	return &Server{}
}

func (s *Server) ListenAndServe(ctx context.Context) {

	err := s.internalServer.Listen()
	if err != nil {
		slog.ErrorContext(ctx, "socket Listen", "err", err)
	}

	s.internalServer.AddCodec()   // protocol buffer, http などのコーデックが追加できる
	s.internalServer.AddHandler() // ハンドラが追加できる

	s.internalServer.Serve(ctx) // サーバーを起動する

}
