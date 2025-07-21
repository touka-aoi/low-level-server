package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// DefaultHandlers returns a router with default handlers
func DefaultHandlers() *Router {
	router := NewRouter()

	// Home handler
	router.GET("/", func(req *Request) ([]byte, error) {
		return NewResponse().
			Text("hello! I'm go server !").
			Build(), nil
	})

	// Ping handler
	router.POST("/ping", func(req *Request) ([]byte, error) {
		return NewResponse().
			Text("pong").
			Build(), nil
	})

	// Echo handler
	router.POST("/echo", func(req *Request) ([]byte, error) {
		if len(req.Body) == 0 {
			return NewResponse().
				Status(400).
				Text("No body provided").
				Build(), nil
		}

		return NewResponse().
			Header("X-Echo-Length", fmt.Sprintf("%d", len(req.Body))).
			Body(req.Body).
			Build(), nil
	})

	// JSON API example
	router.GET("/api/status", func(req *Request) ([]byte, error) {
		status := map[string]interface{}{
			"status": "ok",
			"server": "low-level-server",
			"version": "1.0.0",
		}

		data, err := json.Marshal(status)
		if err != nil {
			return nil, err
		}

		return NewResponse().
			JSON(data).
			Build(), nil
	})

	// File upload handler (example)
	router.POST("/upload", func(req *Request) ([]byte, error) {
		contentType := req.Headers["Content-Type"]
		slog.Info("Upload request", "contentType", contentType, "size", len(req.Body))

		// TODO: Handle multipart/form-data
		// For now, just acknowledge
		response := map[string]interface{}{
			"message": "Upload endpoint (not implemented)",
			"size":    len(req.Body),
		}

		data, _ := json.Marshal(response)
		return NewResponse().
			JSON(data).
			Build(), nil
	})

	// Media serving example
	router.GET("/media/*", func(req *Request) ([]byte, error) {
		// Extract file path
		// For example: /media/video.m3u8
		
		// TODO: Implement actual file serving
		// For now, return 404
		return NewResponse().
			Status(404).
			Text("Media serving not implemented yet").
			Build(), nil
	})

	// Health check
	router.GET("/health", func(req *Request) ([]byte, error) {
		return NewResponse().
			Header("Cache-Control", "no-cache").
			Text("OK").
			Build(), nil
	})

	return router
}