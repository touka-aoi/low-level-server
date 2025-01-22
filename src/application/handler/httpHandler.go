package handler

import (
	"bufio"
	"context"
	"fmt"
	"github.com/touka-aoi/low-level-server/interface/components"
	"log/slog"
	"net/http"
)

type HttpHandler struct {
}

func (h *HttpHandler) OnRead(ctx context.Context, peer *components.Peer, buffer []byte) error {

	// リクエストに対して他の関数に移譲する

	req, err := http.ReadRequest(bufio.NewReader(peer))
	if err != nil {
		slog.Error("ReadRequest", "err", err)
	}
	slog.Info("Read", "method", req.Method, "url", req.URL, "req", req)

	// 200 OK を返す
	body := "hello! I'm go server !"
	contentLen := len(body)
	response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\n\r\n%s", contentLen, body)
	peer.Write([]byte(response))
	return nil
}

func NewHttpHandler() Handler {
	return &HttpHandler{}
}
