### これはなに？
go server using io-uring

### できること
http
- POST /ping : pongが返ります
- POST /echo : オウム返しをします
- GET / : hello! I'm go server ! を返します

### TODO
- [ ] io-uring core層のリファクタリング
- [ ] 接続している全Peerに対して送信するbroadcastの実装
- [ ] アプリケーション層の実装
- こんな感じの関数型ハンドラとして実装するのがいいみたい
```go
 type PacketHandler func(session *GameSession, data []byte) []byte

  func GameMiddleware(handler PacketHandler) MiddlewareFunc {
      return func(ctx *Context, next NextFunc) error {
          session := ctx.Metadata["session"].(*GameSession)
          ctx.Response = handler(session, ctx.Data)
          return nil
      }
  }

```
- [ ] webRTC対応
- [ ] udp対応
- [ ] http/2対応
- [ ] http/3対応

### 問題点
- AIにコードを書かせているので、正しい設計をしてるかどうかわからない
- 考慮不足が多くて最善の設計をしているかわからない。
- io-uringのロック、op周りの設計が甘くて壊れそう

### ファイルアップローダー (第一章)
- 
- 


## Architecture Design

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ Application Layer                                               │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐       │
│  │ HTTP Handler  │  │ gRPC Handler  │  │ Custom Proto  │       │
│  │               │  │               │  │ Handler       │       │
│  └───────────────┘  └───────────────┘  └───────────────┘       │
└─────────────────────────────────────────────────────────────────┘
                              │
                        ProtocolHandler Interface
                              │
┌─────────────────────────────────────────────────────────────────┐
│ Reactor Layer (Single Event Loop)                              │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ State Machine Engine                                        │ │
│  │  - Connection state management (10,000+ connections)       │ │
│  │  - Protocol-specific state transitions                     │ │
│  │  - Event dispatching and action execution                  │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐                      │
│  │ FD → State Map  │  │ Write Command   │                      │
│  │ fd:4 → Reading  │  │ Queue           │                      │
│  │ fd:5 → Writing  │  │ (Non-blocking)  │                      │
│  │ fd:6 → Processing│  │                 │                      │
│  └─────────────────┘  └─────────────────┘                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                        Single Thread I/O
                              │
┌─────────────────────────────────────────────────────────────────┐
│ I/O Backend Layer                                               │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ io_uring Integration                                        │ │
│  │  - Submission Queue (SQ) for I/O operations               │ │
│  │  - Completion Queue (CQ) for I/O results                  │ │
│  │  - Buffer rings for zero-copy operations                  │ │
│  │  - Multishot operations for high throughput               │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
                        Linux Syscalls
                              │
┌─────────────────────────────────────────────────────────────────┐
│ System Layer                                                    │
│  - Raw socket operations (SYS_SOCKET, SYS_BIND, etc.)         │
│  - Direct memory management                                     │
│  - Hardware-level optimizations                                │
└─────────────────────────────────────────────────────────────────┘
```

### Core Design Principles

#### 1. Single-Threaded Event Loop + Worker Pool
```
┌─────────────────────────────────────────────────────────────────┐
│ Worker Pool (CPU-intensive tasks only)                         │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │Worker 1  │ │Worker 2  │ │Worker 3  │ │Worker 4  │          │
│  │(Parsing) │ │(Business)│ │(Template)│ │(Crypto)  │          │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
└─────────────────────────────────────────────────────────────────┘
                              ↑
                        Work Queue (Channel)
                              ↑
┌─────────────────────────────────────────────────────────────────┐
│ Single Event Loop Thread (I/O only)                            │
│  - Handles 10,000+ concurrent connections                      │
│  - Zero goroutines per connection                              │
│  - Pure I/O: Read/Write/Accept                                 │
│  - State machine transitions                                   │
│  - Buffer management                                            │
└─────────────────────────────────────────────────────────────────┘
```

#### 2. State Machine + Reactor Pattern
```
Connection State Machine Example (HTTP):

    ┌─────────────┐
    │   INITIAL   │
    └─────┬───────┘
          │ OnAccept
          ▼
    ┌─────────────┐  OnRead(partial)  ┌─────────────┐
 ┌─→│   READING   │◄─────────────────│   READING   │
 │  └─────┬───────┘                  └─────────────┘
 │        │ OnRead(complete)
 │        ▼                          
 │  ┌─────────────┐  OffloadAction   ┌─────────────┐
 │  │  PARSING    │ ────────────────→│ PROCESSING  │
 │  └─────┬───────┘                  └─────┬───────┘
 │        │ WriteAction                     │ WorkerDone
 │        ▼                                 ▼
 │  ┌─────────────┐  OnWrite(partial) ┌─────────────┐
 │  │   WRITING   │◄─────────────────│   WRITING   │
 │  └─────┬───────┘                  └─────────────┘
 │        │ OnWrite(complete)
 │        ▼
 │  ┌─────────────┐
 └──│ KEEP_ALIVE  │ (HTTP/1.1 pipeline)
    └─────┬───────┘
          │ CloseAction / Timeout
          ▼
    ┌─────────────┐
    │   CLOSED    │
    └─────────────┘
```

#### 3. Multi-Protocol Support (Fixed Binding)
```
Protocol Configuration:

┌─────────────────────────────────────────────────────────────────┐
│ Fixed Protocol Binding (No Auto-Detection)                     │
│                                                                 │
│ Listen Endpoints:                                               │
│ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│ │ tcp:8080        │ │ tcp:8443        │ │ udp:53          │   │
│ │ Protocol: HTTP  │ │ Protocol: HTTPS │ │ Protocol: DNS   │   │
│ │ Handler: Static │ │ Handler: Static │ │ Handler: Static │   │
│ └─────────────────┘ └─────────────────┘ └─────────────────┘   │
│                                                                 │
│ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│ │ tcp:9000        │ │ udp:3478        │ │ tcp:50051       │   │
│ │ Protocol:       │ │ Protocol:       │ │ Protocol:       │   │
│ │ ProtocolBuffer  │ │ WebRTC          │ │ gRPC            │   │
│ └─────────────────┘ └─────────────────┘ └─────────────────┘   │
└─────────────────────────────────────────────────────────────────┘

Benefits:
✓ Zero detection overhead
✓ Predictable performance  
✓ Clear separation of concerns
✓ Protocol-specific optimizations
```

#### 4. Memory Efficiency
```
Memory Usage (10,000 connections):
┌─────────────────────────────────────────────────────────────────┐
│ ConnectionState: ~52 bytes × 10,000 = ~520KB                   │
│ Buffer pools: 4KB × 20,000 (read+write) = ~80MB                │
│ Protocol states: Variable by protocol type                     │
│ Total: ~100MB fixed memory usage                               │
│                                                                 │
│ Key Features:                                                   │
│ - Fixed memory pools                                            │
│ - Zero-copy buffer management                                   │
│ - Connection state reuse                                        │
│ - Predictable memory growth                                     │
└─────────────────────────────────────────────────────────────────┘
```

#### 5. Write Command Flow
```
Application → Reactor Write Flow:

Application Thread          WriteQueue           Reactor           io_uring
     │                         │                   │                 │
     │ ctx.Write(data) ────────→│                   │                 │
     │                         │ Enqueue           │                 │
     │                         │ (Non-blocking)    │                 │
     │                         │                   │                 │
     │ Return immediately      │                   │                 │
     │                         │                   │                 │
     │                         │ Poll Queue ←──────│                 │
     │                         │                   │                 │
     │                         │ Commands ─────────→│                 │
     │                         │                   │                 │
     │                         │                   │ SubmitWrite ────→│
     │                         │                   │                 │
     │                         │                   │ Write CQE ←─────│
     │ OnWriteComplete ←───────│ Callback ←────────│                 │
```

### Performance Characteristics

- **Scalability**: 10,000+ concurrent connections on single thread
- **Memory**: Linear growth, ~100MB for 10K connections
- **CPU**: Single core for I/O, additional cores for CPU-intensive tasks
- **Latency**: Minimal context switches, direct syscall access
- **Throughput**: io_uring multishot operations for maximum efficiency

### Extensibility

The architecture is designed for easy protocol extension:

```go
// Adding new protocol
reactor.RegisterProtocol("custom", &CustomProtocolFactory{})

// Configuration-based setup
listeners:
  - address: "0.0.0.0:9999"
    transport: tcp
    protocol: custom
    config:
      custom_setting: value
```

Future protocols can be added without modifying the core Reactor implementation.
