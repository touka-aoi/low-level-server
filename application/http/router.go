package http

import (
	"strings"
	"sync"
)

type HandlerFunc func(*Request) ([]byte, error)

type Router struct {
	mu     sync.RWMutex
	routes map[string]map[string]HandlerFunc // method -> path -> handler
}

// NewRouter creates a new router
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]map[string]HandlerFunc),
	}
}

// Handle registers a handler for the given method and path
func (r *Router) Handle(method, path string, handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.routes[method] == nil {
		r.routes[method] = make(map[string]HandlerFunc)
	}
	r.routes[method][path] = handler
}

// GET registers a GET handler
func (r *Router) GET(path string, handler HandlerFunc) {
	r.Handle("GET", path, handler)
}

// POST registers a POST handler
func (r *Router) POST(path string, handler HandlerFunc) {
	r.Handle("POST", path, handler)
}

// PUT registers a PUT handler
func (r *Router) PUT(path string, handler HandlerFunc) {
	r.Handle("PUT", path, handler)
}

// DELETE registers a DELETE handler
func (r *Router) DELETE(path string, handler HandlerFunc) {
	r.Handle("DELETE", path, handler)
}

// Match finds a handler for the given method and path
func (r *Router) Match(method, path string) HandlerFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Exact match
	if methodRoutes, ok := r.routes[method]; ok {
		if handler, ok := methodRoutes[path]; ok {
			return handler
		}

		// Simple wildcard matching (e.g., /media/*)
		for pattern, handler := range methodRoutes {
			if strings.HasSuffix(pattern, "/*") {
				prefix := pattern[:len(pattern)-2]
				if strings.HasPrefix(path, prefix) {
					return handler
				}
			}
		}
	}

	return nil
}
