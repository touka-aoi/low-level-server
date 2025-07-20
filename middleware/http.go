package middleware

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
)

type HTTPRequest struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    []byte
}

func HTTPParserMiddleware(ctx *Context, next NextFunc) error {
	if !isHTTPRequest(ctx.Data) {
		return next(ctx)
	}

	request, err := parseHTTPRequest(ctx.Data)
	if err != nil {
		slog.Error("Failed to parse HTTP request", "error", err)
		return err
	}

	ctx.Request = request
	slog.Debug("Parsed HTTP request", "method", request.Method, "path", request.Path)

	return next(ctx)
}

func HTTPResponseMiddleware(ctx *Context, next NextFunc) error {
	if err := next(ctx); err != nil {
		return err
	}

	if httpReq, ok := ctx.Request.(*HTTPRequest); ok {
		response := createHTTPResponse(httpReq)
		ctx.Response = response
		slog.Debug("Created HTTP response", "method", httpReq.Method, "path", httpReq.Path)
	}

	return nil
}

func isHTTPRequest(data []byte) bool {
	methods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH"}
	dataStr := string(data)

	for _, method := range methods {
		if strings.HasPrefix(dataStr, method+" ") {
			return true
		}
	}
	return false
}

func parseHTTPRequest(data []byte) (*HTTPRequest, error) {
	lines := strings.Split(string(data), "\r\n")
	if len(lines) < 1 {
		return nil, fmt.Errorf("invalid HTTP request")
	}

	requestLine := strings.Split(lines[0], " ")
	if len(requestLine) < 3 {
		return nil, fmt.Errorf("invalid request line")
	}

	method := requestLine[0]
	path := requestLine[1]

	headers := make(map[string]string)
	bodyStartIndex := -1

	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			bodyStartIndex = i + 1
			break
		}

		headerParts := strings.SplitN(line, ": ", 2)
		if len(headerParts) == 2 {
			headers[headerParts[0]] = headerParts[1]
		}
	}

	var body []byte
	if bodyStartIndex != -1 && bodyStartIndex < len(lines) {
		bodyLines := lines[bodyStartIndex:]
		body = []byte(strings.Join(bodyLines, "\r\n"))
	}

	return &HTTPRequest{
		Method:  method,
		Path:    path,
		Headers: headers,
		Body:    body,
	}, nil
}

func createHTTPResponse(req *HTTPRequest) []byte {
	var response bytes.Buffer

	switch req.Path {
	case "/":
		response.WriteString("HTTP/1.1 200 OK\r\n")
		response.WriteString("Content-Type: text/plain\r\n")
		body := "hello! I'm go server !"
		response.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(body)))
		response.WriteString("\r\n")
		response.WriteString(body)
	case "/ping":
		if req.Method == "POST" {
			response.WriteString("HTTP/1.1 200 OK\r\n")
			response.WriteString("Content-Type: text/plain\r\n")
			body := "pong"
			response.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(body)))
			response.WriteString("\r\n")
			response.WriteString(body)
		} else {
			response.WriteString("HTTP/1.1 405 Method Not Allowed\r\n")
			response.WriteString("\r\n")
		}
	case "/echo":
		if req.Method == "POST" {
			response.WriteString("HTTP/1.1 200 OK\r\n")
			response.WriteString("Content-Type: text/plain\r\n")
			response.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(req.Body)))
			response.WriteString("\r\n")
			response.Write(req.Body)
		} else {
			response.WriteString("HTTP/1.1 405 Method Not Allowed\r\n")
			response.WriteString("\r\n")
		}
	default:
		response.WriteString("HTTP/1.1 404 Not Found\r\n")
		response.WriteString("\r\n")
	}

	return response.Bytes()
}
