package http

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/touka-aoi/low-level-server/core/engine"
	"github.com/touka-aoi/low-level-server/protocol"
)

type HTTPApplication struct {
	router *Router
}

func NewHTTPApplication(router *Router) protocol.Application {
	return &HTTPApplication{
		router: router,
	}
}

// OnConnect is called when a new connection is established
func (h *HTTPApplication) OnConnect(ctx context.Context, peer *engine.Peer) error {
	slog.DebugContext(ctx, "HTTP connection established",
		"peer", peer.RemoteAddr,
		"local", peer.LocalAddr)
	return nil
}

// OnData processes HTTP requests
func (h *HTTPApplication) OnData(ctx context.Context, peer *engine.Peer, data []byte) ([]byte, error) {
	// Parse HTTP request
	req, err := ParseHTTPRequest(data)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to parse HTTP request", "error", err)
		return createErrorResponse(400, "Bad Request"), nil
	}

	slog.DebugContext(ctx, "HTTP request received",
		"method", req.Method,
		"path", req.Path,
		"peer", peer.RemoteAddr)

	// Route the request
	handler := h.router.Match(req.Method, req.Path)
	if handler == nil {
		return createErrorResponse(404, "Not Found"), nil
	}

	// Execute handler
	response, err := handler(req)
	if err != nil {
		slog.ErrorContext(ctx, "Handler error", "error", err)
		return createErrorResponse(500, "Internal Server Error"), nil
	}

	return response, nil
}

// OnDisconnect is called when a connection is closed
func (h *HTTPApplication) OnDisconnect(ctx context.Context, peer *engine.Peer) error {
	slog.DebugContext(ctx, "HTTP connection closed", "peer", peer.RemoteAddr)
	return nil
}

func createErrorResponse(status int, message string) []byte {
	statusText := statusTexts[status]
	return []byte(fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Length: %d\r\n\r\n%s",
		status, statusText, len(message), message))
}

var statusTexts = map[int]string{
	200: "OK",
	400: "Bad Request",
	404: "Not Found",
	500: "Internal Server Error",
}
