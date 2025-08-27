# CLAUDE.md

会話は日本語で行ってください。

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **low-level HTTP server** written in Go that implements high-performance networking using direct Linux system calls and io_uring. The project follows a reactor pattern with a single-threaded event loop architecture designed to handle 10,000+ concurrent connections.

**Current Status**: Work in progress with HTTP server implementation functional. Major refactoring ongoing for multi-protocol support (TCP/UDP).

## Architecture

The codebase follows a layered architecture with strict separation of concerns:

### Directory Structure

- **`cmd/`** - Application entry points
  - `http-server/` - Main HTTP server binary
  - `debug/` - Debug server binary with UDP support

- **`core/`** - Core networking infrastructure
  - `engine/` - Network engine abstraction layer
    - `NetEngine` interface for I/O mechanisms
    - `io_uring_engine.go` - io_uring-based high-performance I/O
    - `listener.go` - TCP/UDP listener operations
  - `io/` - Low-level I/O operations  
    - `socket.go` - Direct Linux system calls (SYS_SOCKET, SYS_BIND, etc.)
    - `uring.go` - Complete io_uring implementation with submission/completion queues
  - `event/` - Event system for network events (ACCEPT, READ, WRITE, RECVMSG)
  - `errors/` - Custom error types

- **`transport/`** - Protocol implementations
  - `http/` - HTTP/1.1 protocol handling
    - `app.go` - HTTP application logic
    - `parser.go` - HTTP request parsing
    - `router.go` - Request routing
    - `handlers.go` - Default HTTP handlers

- **`server/`** - Server orchestration layer
  - `network_server.go` - Main server event loop (supports both TCP/UDP)

- **`middleware/`** - Middleware pipeline system
  - `pipeline.go` - Middleware chaining

- **`application/`** - Application-level features (planned)
  - `room/` - Room management for real-time features

### Key Technical Characteristics

- **Linux-specific**: All core files use `//go:build linux` build constraints
- **Zero-copy I/O**: Direct system calls bypass Go's standard library
- **io_uring integration**: Buffer rings and multishot operations for maximum throughput
- **Lock-free design**: Atomic operations for concurrent access
- **Single event loop**: One thread handles all I/O operations
- **Fixed memory pools**: Predictable memory usage (~100MB for 10K connections)

## Development Commands

```bash
# Build the HTTP server
go build ./cmd/http-server

# Run the HTTP server (default port 8080)
go run ./cmd/http-server/main.go

# Run with custom host and port
go run ./cmd/http-server/main.go -host 0.0.0.0 -port 3000

# Run with debug logging
go run ./cmd/http-server/main.go -debug

# Build debug server (UDP support)
go build ./cmd/debug

# Run debug server
go run ./cmd/debug/main.go

# Format code
go fmt ./...

# Run static analysis
go vet ./...

# Manage dependencies
go mod tidy

# Build for Linux deployment (from Makefile)
make build  # Builds debug binary for Linux AMD64

# Deploy and run on remote server (from Makefile)
make run    # Deploys to configured VM and runs
```

## API Endpoints

Current HTTP endpoints:

- `GET /` - Returns "hello! I'm go server !"
- `POST /ping` - Returns "pong"
- `POST /echo` - Echoes request body back to client
- `GET /api/status` - Returns JSON status information
- `GET /health` - Health check endpoint
- `POST /upload` - File upload (not yet implemented)
- `GET /media/*` - Media serving (not yet implemented)

## Performance Architecture

### Event Loop Model
```
Single Event Loop (I/O) → Worker Pool (CPU-intensive tasks)
- Event loop handles: Accept, Read, Write, State transitions
- Workers handle: Parsing, Business logic, Template rendering
```

### Connection State Machine
```
INITIAL → READING → PARSING → PROCESSING → WRITING → KEEP_ALIVE/CLOSED
```

### Memory Model
- Connection state: ~52 bytes per connection
- Buffer pools: 4KB read + 4KB write per active connection
- Total: ~100MB fixed memory for 10,000 connections

### io_uring Implementation Details
- Buffer size: 20KB per buffer
- Batch processing: Up to 64 events per iteration
- Multishot operations with F_MORE flag for continuous reading
- Manual buffer management for zero-copy performance

## Development Notes

- Tests are not yet implemented
- Some functions contain TODO comments for future improvements
- The project intentionally bypasses Go's `net` package for performance
- All I/O operations use direct Linux system calls
- Buffer management is manual for zero-copy performance
- UDP protocol support added for debug server
- Error handling uses panic in some places (to be improved)