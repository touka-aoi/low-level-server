package http

import (
	"bytes"
	"fmt"
	"strings"
)

// Request represents an HTTP request
type Request struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
	Body    []byte
}

// ParseHTTPRequest parses raw HTTP request data
func ParseHTTPRequest(data []byte) (*Request, error) {
	// Find the end of headers
	headerEnd := bytes.Index(data, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		return nil, fmt.Errorf("invalid HTTP request: no header terminator")
	}

	// Split headers and body
	headerData := data[:headerEnd]
	bodyData := data[headerEnd+4:]

	// Parse headers line by line
	lines := bytes.Split(headerData, []byte("\r\n"))
	if len(lines) < 1 {
		return nil, fmt.Errorf("invalid HTTP request: no request line")
	}

	// Parse request line
	requestLine := string(lines[0])
	parts := strings.Split(requestLine, " ")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid request line: %s", requestLine)
	}

	req := &Request{
		Method:  parts[0],
		Path:    parts[1],
		Version: parts[2],
		Headers: make(map[string]string),
		Body:    bodyData,
	}

	// Parse headers
	for i := 1; i < len(lines); i++ {
		line := string(lines[i])
		if line == "" {
			break
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])
		req.Headers[key] = value
	}

	return req, nil
}