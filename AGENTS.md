# AGENTS.md

## Guidelines for Agentic Coding Agents

This file contains instructions for AI agents (like opencode, Cursor, etc.) working in this repository. Follow these strictly to maintain consistency.

## 1. Build, Lint, Test Commands

### Dependencies
- `make deps` - Installs buf, protoc-gen-go, protoc-gen-go-grpc, golangci-lint
- `go mod tidy` - Update dependencies

### Proto Generation
- `make proto` or `buf generate` - Generate all Go/TS/PHP code from protos
- `make proto-path PROTO_PATH=proto/xxx/v1` - Generate for specific proto
- `make lint-proto` or `buf lint` - Lint proto files
- Config: buf.yaml, buf.gen.yaml

### Build
- `make build` - Builds server and client binaries to ./bin/
- `go build -o bin/server ./cmd/server`
- `go build ./cmd/client`

### Run
- `make run-server` or `go run ./cmd/server`
- `make run-client` or `go run ./cmd/client`

### Quality Checks (MUST RUN AFTER CHANGES)
- `make lint` - golangci-lint run ./cmd/... ./internal/...
- `make vet` - go vet ./cmd/... ./internal/...
- `make test` - Runs tests with -race (excludes testutil, cmd/client/server)
- Test coverage: `make test-cover` (generates coverage.txt with atomic mode + race)
- Full: `go test ./... -race -count=1`

### Running a Single Test (Important)
- `go test -run TestSayHello ./internal/service/greeter -v`
- `go test -run '^TestSpecificName$' ./internal/server/interceptor -count=1 -v`
- With race: `go test -run TestName ./path -race`
- Specific package only: `go test ./internal/logging -run TestXXX`
- Coverage for one: `go test -run TestFoo ./pkg -coverprofile=coverage.txt`

### CI Commands
- See .github/workflows/test.yaml: go vet, go test with race and coverage
- Use `go test $(go list ./cmd/... ./internal/... | grep -v -E '/testutil|cmd/(client|server)$') -race`

### Cleanup
- `make clean` - Removes binaries and generated pkg/gen*

**IMPORTANT**: After any code changes, ALWAYS run `make lint`, `make vet`, `make test` (or equivalent single test). If no Makefile command, ask user to add to this file.

## 2. Code Style Guidelines

### Copyright Header (Required in EVERY .go file)
```go
// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.
```

### Package Documentation
- Always add package comment explaining purpose.
- For services: "Package xxx provides the Xxx gRPC service implementation."

### Imports
- Standard library first (sorted)
- Blank line
- Third-party (github.com, google.golang.org, etc.)
- Blank line
- Local/internal packages and generated pb (use alias `pb "path/to/pkg/gen/..."`)
- Example from greeter.go:1:18
- Use goimports to organize

### Naming Conventions
- Types: PascalCase (Service, Handler)
- Functions: camelCase, exported if public (NewService, Register)
- Variables: camelCase, short but descriptive (ctx, req, svc)
- Constants: UPPER_SNAKE_CASE if unexported use lower
- Interface methods: match gRPC patterns
- Test files: package xxx_test (black-box testing)
- Methods on services: match proto RPC names (SayHello, SayHelloServerStream)

### Types and Structs
- Embed UnimplementedXXXServer from generated pb
- Use functional options pattern for configuration (see internal/server/option.go:1)
- Logger is injected via constructor: logging.Handler
- Keep structs minimal

### Error Handling
- Wrap errors: `fmt.Errorf("context: %w", err)`
- gRPC errors: `status.Errorf(codes.InvalidArgument, "msg: %v", err)` or `status.Error`
- In interceptors: use codes.Internal for panics/recovery (see recovery.go:33)
- Never panic in production code
- Log errors with structured logging using logger.Error(msg, "key", value)
- In tests: use require.NoError(t, err), assert.Equal

### Logging
- Use internal/logging.Handler (not direct slog)
- Methods: .Info(), .Debug(), .Warn(), .Error(msg string, args ...any)
- Always include context like "method", req fields
- See internal/logging/logging.go:28
- Services receive logger via NewService(l logging.Handler)

### Testing Style
- Use bufconn for in-memory gRPC tests (see internal/testutil/grpctest.go)
- Helper functions with t.Helper()
- Use testify: assert, require
- t.Cleanup for teardown
- Black box tests in _test package
- Test both unary and streaming
- See greeter_test.go:47 and server_test.go
- No tests in cmd/ packages

### Formatting and Comments
- Run `gofmt -s` and goimports
- Add godoc comments for all exported types/functions and package docs (required; see `internal/server/interceptor/`)
- Avoid unnecessary inline `//` comments inside function bodies
- Keep functions < 50 lines when possible
- Use early returns
- Prefer explicit over implicit

### Server and Interceptors
- Use functional options: server.WithXXX()
- Register services via RegisterService(registrar func(*grpc.Server))
- Interceptors in internal/server/interceptor/
- Always chain interceptors properly: Recovery(), Logging(), etc.
- See server.go:42 for New(), option.go for options
- For new interceptors: implement both unary and stream versions

### Proto and Generated Code
- NEVER edit files in pkg/gen/
- Edit only .proto files in proto/*/v1/*.proto
- Then run make proto
- Follow existing proto patterns (see proto/helloworld/v1/helloworld.proto)
- Use google.golang.org/genproto conventions

### General Go Best Practices
- Go 1.26+
- Use context everywhere
- Graceful shutdown in server
- Avoid global state except for logging.Default()
- Error strings should not be capitalized or end with punctuation
- Use %w for wrapping in fmt.Errorf
- Struct tags for any JSON if needed (though rare in gRPC)

## Additional Rules
- Follow existing patterns exactly (look at similar files first using tools)
- When editing, read file first with Read tool
- Use edit tool for changes, never assume
- Mimic code style, naming, error handling from neighboring files
- Check go.mod before adding new deps - NEVER assume libraries
- For new services: follow adding new service section in README.md
- Security: never log sensitive data, no secrets in code
- Always verify with lint/test after changes

## Cursor / Copilot Rules
- No .cursorrules or .github/copilot-instructions.md found. Follow this file instead.

This ensures all agents produce consistent, high-quality code matching the project's professional standards.
