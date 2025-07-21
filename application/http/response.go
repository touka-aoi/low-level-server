package http

import (
	"fmt"
	"time"
)

// ResponseBuilder helps build HTTP responses
type ResponseBuilder struct {
	status  int
	headers map[string]string
	body    []byte
}

// NewResponse creates a new response builder
func NewResponse() *ResponseBuilder {
	return &ResponseBuilder{
		status:  200,
		headers: make(map[string]string),
	}
}

// Status sets the response status code
func (r *ResponseBuilder) Status(code int) *ResponseBuilder {
	r.status = code
	return r
}

// Header adds a response header
func (r *ResponseBuilder) Header(key, value string) *ResponseBuilder {
	r.headers[key] = value
	return r
}

// Body sets the response body
func (r *ResponseBuilder) Body(data []byte) *ResponseBuilder {
	r.body = data
	r.headers["Content-Length"] = fmt.Sprintf("%d", len(data))
	return r
}

// Text sets a text response body
func (r *ResponseBuilder) Text(text string) *ResponseBuilder {
	r.headers["Content-Type"] = "text/plain; charset=utf-8"
	return r.Body([]byte(text))
}

// JSON sets a JSON response body
func (r *ResponseBuilder) JSON(data []byte) *ResponseBuilder {
	r.headers["Content-Type"] = "application/json"
	return r.Body(data)
}

// HTML sets an HTML response body
func (r *ResponseBuilder) HTML(html string) *ResponseBuilder {
	r.headers["Content-Type"] = "text/html; charset=utf-8"
	return r.Body([]byte(html))
}

// Build creates the final HTTP response bytes
func (r *ResponseBuilder) Build() []byte {
	// Set default headers
	if _, ok := r.headers["Date"]; !ok {
		r.headers["Date"] = time.Now().UTC().Format(time.RFC1123)
	}
	if _, ok := r.headers["Server"]; !ok {
		r.headers["Server"] = "low-level-server/1.0"
	}

	// Build response
	statusText := statusTexts[r.status]
	response := fmt.Sprintf("HTTP/1.1 %d %s\r\n", r.status, statusText)

	// Add headers
	for key, value := range r.headers {
		response += fmt.Sprintf("%s: %s\r\n", key, value)
	}

	// End headers
	response += "\r\n"

	// Add body if present
	if len(r.body) > 0 {
		response += string(r.body)
	}

	return []byte(response)
}