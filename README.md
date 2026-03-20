# gRPC Template

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.26-blue?logo=go)](https://go.dev/dl/)
[![Go Reference](https://pkg.go.dev/badge/github.com/H0llyW00dzZ/grpc-template.svg)](https://pkg.go.dev/github.com/H0llyW00dzZ/grpc-template)
[![Go Report Card](https://goreportcard.com/badge/github.com/H0llyW00dzZ/grpc-template)](https://goreportcard.com/report/github.com/H0llyW00dzZ/grpc-template)

A production-ready Go gRPC template/boilerplate for bootstrapping new gRPC projects. Designed as a template repository for any Git code hosting (e.g., GitHub).

> **Actively maintained** — I built this template from my own experience with high-performance and critical systems that rely on gRPC. Proto definitions are added as I encounter real-world patterns worth templating. Use this repo as a template to bootstrap your next project without writing boilerplate from scratch.

## Features

- **Proto-first** — [Buf](https://buf.build/) for proto linting and code generation
- **Multi-language** — generates Go server & client stubs, TypeScript/JavaScript and PHP client code
- **Functional Options** — clean, extensible server configuration
- **TLS / mTLS** — secure connections with a single option
- **Built-in Interceptors** — logging (slog) and panic recovery for both unary and streaming RPCs
- **Health Checks** — standard [gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md)
- **Server Reflection** — debug with [grpcurl](https://github.com/fullstorydev/grpcurl) out of the box
- **Graceful Shutdown** — handles `SIGINT`/`SIGTERM` and drains connections
- **Proto Collection** — ready-to-use proto templates for common patterns
- **Example RPCs** — unary, server streaming, client streaming, and bidirectional

## Why gRPC in 2026?

In 2026, gRPC is the clear winner for service-to-service communication — and **especially for AI / AI-tool workloads**:

| | REST / JSON (HTTP/1.1) | gRPC (HTTP/2 + Protobuf) |
|---|---|---|
| **Serialization** | Text-based JSON — parse overhead on every call | Binary Protobuf — 5-10× smaller payloads, near-zero parse cost |
| **Transport** | One request per connection (or clunky keep-alive) | Multiplexed streams over a single HTTP/2 connection |
| **Streaming** | Workarounds (SSE, WebSockets, chunked transfer) | Native bidirectional streaming, first-class support |
| **Latency** | Higher per-call overhead from headers + JSON encoding | Minimal framing; ideal for high-frequency AI inference calls |
| **Code generation** | Manual client SDKs or OpenAPI generators | Strongly-typed stubs generated from `.proto` files for any language |

Modern AI systems — LLM orchestrators, inference pipelines, tool-calling agents (MCP), embedding services — make **thousands of low-latency calls** between components. The overhead of REST/JSON serialization and HTTP/1.1 connection management adds up fast. gRPC eliminates that overhead with binary serialization, persistent multiplexed connections, and native streaming, making it the natural transport layer for AI-native architectures.

## Showcase

The demo below shows the [`cmd/server`](cmd/server) and [`cmd/client`](cmd/client) in action — unary and server-streaming RPCs over gRPC:

![gRPC Go Demo](assets/image/grpc-go.gif)

With streaming interceptors enabled, both unary and streaming RPCs are logged with method, duration, and error details:

![gRPC Streaming Interceptor Demo](assets/image/grpc-go-streaming-Interceptor.gif)

## Project Structure

```text
grpc-template/
├── cmd/
│   ├── server/main.go          # Server entry point
│   └── client/main.go          # Client demo
├── internal/
│   ├── server/                 # gRPC server lifecycle
│   │   ├── server.go           # Server with graceful shutdown
│   │   ├── option.go           # Functional options (TLS, mTLS)
│   │   └── interceptor.go      # Logging & recovery interceptors
│   ├── service/
│   │   └── greeter/            # Example service implementation
│   │       ├── greeter.go      # Greeter service
│   │       └── greeter_test.go # Greeter service tests
│   └── testutil/
│       └── grpctest.go         # Shared bufconn test helpers
├── proto/
│   ├── analytics/v1/           # Event tracking & aggregation
│   ├── audit/v1/               # Audit logging & compliance
│   ├── auth/v1/                # Multi-credential auth
│   ├── config/v1/              # Remote config & feature flags
│   ├── crud/v1/                # CRUD with pagination & field masks
│   ├── discovery/v1/           # Service registry & discovery
│   ├── echo/v1/                # All 4 RPC patterns
│   ├── geo/v1/                 # Geospatial operations
│   ├── helloworld/v1/          # Greeter (unary + server streaming)
│   ├── identity/v1/            # User management & RBAC
│   ├── kv/v1/                  # Key-value store with watch
│   ├── media/v1/               # Media processing pipelines
│   ├── messaging/v1/           # Real-time messaging / pub-sub
│   ├── notification/v1/        # Push notifications & events
│   ├── queue/v1/               # Message queue with DLQ
│   ├── ratelimit/v1/           # Rate limiting & quota enforcement
│   ├── scheduler/v1/           # Cron / scheduled job management
│   ├── search/v1/              # Full-text search & indexing
│   ├── secret/v1/              # Vault / secret management
│   ├── storage/v1/             # Streaming file upload/download
│   ├── task/v1/                # Async job queue with progress
│   └── workflow/v1/            # State machine / orchestration
├── pkg/gen/                    # Generated Go code (do not edit)
├── pkg/gen-ts/                 # Generated TypeScript/JS code (do not edit)
├── pkg/gen-php/                # Generated PHP code (do not edit)
├── buf.yaml                    # Buf module config
├── buf.gen.yaml                # Buf generation config
├── Makefile                    # Build automation
└── README.md
```

## Getting Started

Clone this repository to bootstrap your new project:

```bash
git clone https://github.com/H0llyW00dzZ/grpc-template.git my-grpc-project
cd my-grpc-project
```

Then update the Go module path to match your own project:

```bash
go mod edit -module github.com/yourorg/yourproject
```

## Prerequisites

- [Go](https://go.dev/) 1.26+
- [Buf CLI](https://buf.build/docs/installation)

Install tools:

```bash
make deps
```

## Quick Start

### 1. Generate proto code

```bash
make proto
```

### 2. Run the server

```bash
make run-server
```

### 3. Run the client (in another terminal)

```bash
make run-client
```

## Proto Collection

This template ships with ready-to-use proto definitions so you never have to write them from scratch:

| Proto | Package | What It Covers |
|-------|---------|----------------|
| `helloworld/v1` | Greeter | Unary + server-streaming RPCs |
| `echo/v1` | Echo | All 4 RPC patterns (unary, server stream, client stream, bidirectional) |
| `crud/v1` | CrudService | Create, Get, List (pagination), Update (field mask), Delete |
| `auth/v1` | AuthService | Multi-credential login (`oneof`: password, API key, OAuth), refresh, validate, logout |
| `messaging/v1` | MessagingService | Send, subscribe (server stream), full-duplex streaming, channels, metadata |
| `storage/v1` | StorageService | Chunked upload (client stream), download (server stream), object info, list |
| `task/v1` | TaskService | Submit, status, watch (server stream for progress), cancel, list with filters |
| `notification/v1` | NotificationService | Send to recipients/topics, subscribe (server stream), acknowledge, list |
| `kv/v1` | KvService | Get, set (TTL), delete, batch ops, watch (server stream), optimistic locking |
| `discovery/v1` | DiscoveryService | Register, deregister, lookup, heartbeat, watch topology changes |
| `ratelimit/v1` | RateLimitService | Check (allow/deny/throttle), report usage, get quota, manage rules |
| `config/v1` | ConfigService | Get/set/delete config, watch changes, feature flag evaluation |
| `audit/v1` | AuditService | Log events (single/batch), query with filters, stream real-time audit trail |
| `scheduler/v1` | SchedulerService | Create/update/delete schedules, pause/resume, cron expressions, execution history |
| `search/v1` | SearchService | Index, search (facets/filters/sort), suggest (autocomplete), batch index |
| `workflow/v1` | WorkflowService | Start, signal, query, cancel, list, watch state transitions |
| `geo/v1` | GeoService | Nearby search, geocode, reverse geocode, geofencing, route, location tracking |
| `media/v1` | MediaService | Transcode, resize, job status, watch progress (server stream), cancel |
| `secret/v1` | SecretService | Get/put/delete secrets, version history, rotation, watch rotation events |
| `identity/v1` | IdentityService | User CRUD, assign/revoke roles, check permissions (RBAC) |
| `analytics/v1` | AnalyticsService | Track events (single + client stream batch), aggregation queries, reports |
| `queue/v1` | QueueService | Publish, consume (server stream), ack/nack, DLQ, visibility timeout |

Pick what you need, delete what you don't. Each proto is self-contained under `proto/<service>/v1/`.

## Testing

Tests use [bufconn](https://pkg.go.dev/google.golang.org/grpc/test/bufconn) for in-memory gRPC connections — no TCP ports needed, fast and hermetic.

```bash
make test
```

Shared test helpers live in `internal/testutil/`. See `internal/service/greeter/greeter_test.go` for a working example of unary and server-streaming RPC tests.

## Adding a New Service

1. **Define a proto** — Create a new `.proto` file under `proto/yourservice/v1/`
2. **Generate code** — Run `make proto`
3. **Implement the service** — Create a new package in `internal/service/<yourservice>/` implementing the generated server interface
4. **Register the service** — Add to `srv.RegisterService(...)` in `cmd/server/main.go`

```go
srv.RegisterService(
    greeterSvc.Register,
    authSvc.Register,
    yourSvc.Register,
)
```

## Customization

| What | Where | How |
|------|-------|-----|
| Server port | `cmd/server/main.go` | `server.WithPort("8080")` |
| Enable TLS | `cmd/server/main.go` | `server.WithTLS("cert.pem", "key.pem")` |
| Enable mTLS | `cmd/server/main.go` | `server.WithMutualTLS("cert.pem", "key.pem", "ca.pem")` |
| Unary interceptors | `cmd/server/main.go` | `server.WithUnaryInterceptors(...)` |
| Stream interceptors | `cmd/server/main.go` | `server.WithStreamInterceptors(...)` |
| Enable reflection | `cmd/server/main.go` | `server.WithReflection()` |
| Set keepalives | `cmd/server/main.go` | `server.WithKeepalive(...)` |
| Set max msg size | `cmd/server/main.go` | `server.WithMaxMsgSize(1024 * 1024 * 50)` |
| Stream limits | `cmd/server/main.go` | `server.WithMaxConcurrentStreams(1000)` |
| Custom listener | `cmd/server/main.go` | `server.WithListener(lis)` |
| Proto output path | `buf.gen.yaml` | Change `out` field |
| Go module path | `go.mod` | `go mod edit -module your/module` |

## Make Targets

| Target | Description |
|--------|-------------|
| `make proto` | Generate Go + TypeScript/JS + PHP code from proto files |
| `make proto-path PROTO_PATH=proto/storage/v1` | Generate code for a specific proto package |
| `make build` | Build server and client binaries |
| `make run-server` | Run the gRPC server |
| `make run-client` | Run the client demo |
| `make test` | Run all tests |
| `make vet` | Run `go vet` |
| `make lint` | Run `golangci-lint` |
| `make clean` | Remove binaries and generated code (Go + TS + PHP) |
| `make deps` | Install required tools |

## License

BSD 3-Clause License — see [LICENSE](LICENSE) for details.
