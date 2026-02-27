# Axon — Tech Stack Specification
## The "Superbike" Configuration

**Version:** 1.0  
**Date:** February 2026  
**Status:** Recommended for Implementation

---

## Core Philosophy

> *"Lightweight, brutally efficient, no bloat."*

The tech stack prioritizes:
- **Raw performance** — Every millisecond matters
- **Minimal dependencies** — Fewer things = fewer failure points
- **Deploy anywhere** — Single binary, no runtime needed
- **Future-proof** — Battle-tested technologies with long-term support

---

## Primary Stack

### Language: Go 1.22+

**Why Go?**
- Compiled — no interpreted overhead
- Native concurrency — goroutines handle thousands of sessions
- Single binary deployment — `go build` = one executable
- Battle-tested — used by Google, Cloudflare, Docker, Kubernetes

**Version Requirements:**
```bash
go version >= 1.22.0
```

---

### HTTP Framework: Fiber v2

**Why Fiber?**
- Inspired by Express.js (familiar for Node devs)
- 10k+ requests/second throughput
- 40x faster than FastAPI
- Extremely low memory footprint
- Native HTTP/2 and WebSocket support

**Installation:**
```go
import "github.com/gofiber/fiber/v2"
```

---

### Browser Control: Rod

**Why Rod?**
- Pure Go Chrome DevTools Protocol (CDP) client
- No Python/Node.js bridge required
- Automatic browser management (download, launch)
- Built-in wait mechanisms
- jQuery-like selector API

**Installation:**
```go
import "github.com/go-rod/rod"
```

**Key Features:**
- Automatic Chromium download/install
- Bi-directional WebSocket communication
- Element tracking and caching
- Event listeners on DOM changes
- Screenshot and PDF generation

---

### Storage: BadgerDB

**Why BadgerDB?**
- Pure Go key-value store
- 10x faster than SQLite for our use case
- Zero-copy reads
- TTL support for session expiry
- Streamable exports for audit logs

**Installation:**
```go
import "github.com/dgraph-io/badger/v4"
```

**Data Stored:**
- Session metadata and state
- Element memory (learned selectors per domain)
- Audit log entries
- Action history

---

### Optional: AI Intent Classifier (Python Microservice)

**Architecture:**
```
Axon (Go) <--gRPC--> Intent Service (Python)
```

**Why separate?**
- Keep Axon core fast and simple
- Python has better ML/AI libraries (transformers, ONNX)
- Can be deployed separately or on GPU
- gRPC is 10x faster than REST

**Technologies:**
- Python 3.11+
- ONNX Runtime (inference)
- sentence-transformers (embeddings)
- FastAPI (if HTTP needed)

---

## Dependency Tree

```
axon/
├── github.com/gofiber/fiber/v2        # HTTP server
├── github.com/go-rod/rod               # Browser CDP
├── github.com/go-rod/rod/lib/proto    # CDP protocol definitions
├── github.com/dgraph-io/badger/v4      # Key-value store
├── github.com/golang/protobuf/proto    # gRPC protobuf
├── google.golang.org/grpc              # gRPC runtime
├── github.com/google/uuid             # Session IDs
├── github.com/rs/zerolog              # Logging
├── github.com/spf13/viper              # Config management
└── golang.org/x/sync/errgroup         # Concurrent operations
```

---

## Development Tools

### Required

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.22+ | Language runtime |
| Git | 2.0+ | Version control |
| Chrome/Chromium | Latest | Browser engine |

### Recommended

| Tool | Purpose |
|------|---------|
| VS Code + Go extension | IDE |
| Air | Live reload during development |
| golangci-lint | Linting |
| golang/mock | Mocking for tests |
| Swagger/Insomnia | API testing |

---

## File Structure (Go Project)

```
axon/
├── cmd/
│   └── axon/
│       └── main.go              # Entry point
├── internal/
│   ├── config/
│   │   └── config.go           # Configuration loading
│   ├── server/
│   │   ├── server.go           # Fiber app setup
│   │   ├── routes.go           # HTTP handlers
│   │   └── middleware.go       # Security middleware
│   ├── browser/
│   │   ├── pool.go            # Browser pool manager
│   │   ├── session.go          # Session management
│   │   └── snapshot.go         # Snapshot extraction
│   ├── security/
│   │   ├── ssrf.go             # SSRF protection
│   │   ├── injection.go        # Prompt injection detection
│   │   └── reversibility.go    # Action classification
│   ├── storage/
│   │   ├── badger.go           # Database operations
│   │   ├── session.go          # Session storage
│   │   └── audit.go            # Audit log storage
│   ├── intent/
│   │   ├── classifier.go       # Element intent classification
│   │   └── resolver.go         # Intent-based element finding
│   └── grpc/
│       ├── client.go           # gRPC client to Python
│       └── proto/              # Generated protobuf files
├── pkg/
│   ├── types/
│   │   └── types.go           # Shared type definitions
│   └── utils/
│       └── utils.go           # Helper functions
├── api/
│   └── openapi.yaml           # OpenAPI specification
├── configs/
│   └── config.yaml            # Default configuration
├── test/
│   ├── integration/           # Integration tests
│   └── mocks/                 # Mock implementations
├── go.mod
├── go.sum
└── Makefile
```

---

## Build Configuration

### Go Build Flags

```bash
# Production build
CGO_ENABLED=0 go build -ldflags="-s -w" -o axon ./cmd/axon

# Cross-compilation
GOOS=linux GOARCH=amd64 go build -o axon-linux-amd64 ./cmd/axon
GOOS=darwin GOARCH=amd64 go build -o axon-darwin-amd64 ./cmd/axon
```

### Output

- **Binary Size**: ~15-20MB (compressed)
- **Memory Usage**: ~50MB idle (without browser)
- **Startup Time**: <100ms

---

## Docker Configuration

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o axon ./cmd/axon

FROM alpine:3.19
RUN apk --no-cache add chromium
COPY --from=builder /app/axon /usr/local/bin/axon
EXPOSE 8020
ENTRYPOINT ["axon"]
```

---

## Performance Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| HTTP Request Latency | <5ms | P99 |
| Session Startup | <50ms | Cold start |
| Snapshot Extraction | <20ms | DOM to semantic |
| Throughput | >10k req/sec | Sustained |
| Memory (idle) | <50MB | Without browser |
| Binary Size | <20MB | Compressed |

---

## Next Steps

1. Initialize Go project: `go mod init github.com/superclaw/axon`
2. Set up Fiber server with basic routes
3. Integrate Rod for browser control
4. Add BadgerDB for session storage
5. Implement security middleware

---

*Axon Tech Stack v1.0 | February 2026*
