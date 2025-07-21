# CLAUDE.md

会話は日本語で行ってください。

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **low-level HTTP server** written in Go that implements high-performance networking using direct Linux system calls and io_uring. The project is currently **work in progress** with major refactoring ongoing.

**Current Status**: Missing main application entry point (`main.go`). Infrastructure is largely complete but application logic is incomplete.

## Architecture

The codebase follows a layered architecture separating concerns:

### Core Components

- **`internal/engine/`** - Network engine abstraction layer
  - `NetEngine` interface provides abstraction over different I/O mechanisms
  - `io_uring_engine.go` implements io_uring-based high-performance I/O
  - `listener.go` handles TCP listener operations

- **`internal/io/`** - Low-level I/O operations  
  - `socket.go` - Raw socket operations using direct Linux system calls
  - `uring.go` - Complete io_uring implementation with submission/completion queues

- **`internal/event/`** - Event system for handling network events (ACCEPT, READ, WRITE)

- **`internal/errors/`** - Custom error types (e.g., `ErrWouldBlock`)

### Key Technical Characteristics

- **Linux-specific**: All Go files have `//go:build linux` build constraints
- **System call usage**: Direct syscalls (SYS_SOCKET, SYS_BIND, SYS_LISTEN) instead of Go standard library
- **io_uring integration**: Complete implementation with buffer rings and multishot operations
- **Lock-free programming**: Uses atomic operations for concurrent access
- **Manual memory management**: Custom buffer ring management for performance

## Development Commands

Since this is a standard Go project, use these commands from the `/src` directory:

```bash
# Build the project
go build

# Run with Go (when main.go exists)
go run .

# Manage dependencies  
go mod tidy

# Format code
go fmt ./...

# Run static analysis
go vet ./...
```

## API Endpoints (Planned)

- `POST /ping` - Returns "pong"
- `POST /echo` - Echo functionality  
- `GET /` - Returns "hello! I'm go server !"

## Development Notes

- The project currently lacks a `main.go` entry point
- Many functions have TODO comments or empty implementations
- Focus on removing panic calls as noted in the TODO list
- All networking bypasses Go's standard `net` package for maximum performance
- Testing infrastructure is not yet implemented