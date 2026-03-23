.PHONY: all proto build clean deps run-server run-client lint vet test

# Binary output directory.
BIN_DIR := bin

# Default target.
all: proto build

## ──────────────────────────────────────────────
## Proto Generation
## ──────────────────────────────────────────────

# Generate Go code from proto files using buf.
proto:
	@echo "==> Generating proto..."
	buf generate
	@echo "==> Done."

# Generate code for a specific proto path.
# Usage: make proto-path PROTO_PATH=proto/storage/v1
proto-path:
	@echo "==> Generating proto for $(PROTO_PATH)..."
	buf generate --path $(PROTO_PATH)
	@echo "==> Done."

# Lint proto files.
lint-proto:
	@echo "==> Linting proto files..."
	buf lint
	@echo "==> Done."

## ──────────────────────────────────────────────
## Build
## ──────────────────────────────────────────────

# Build server and client binaries.
build:
	@echo "==> Building server..."
	go build -o $(BIN_DIR)/server ./cmd/server
	@echo "==> Building client..."
	go build -o $(BIN_DIR)/client ./cmd/client
	@echo "==> Done."

## ──────────────────────────────────────────────
## Run
## ──────────────────────────────────────────────

# Run the gRPC server.
run-server:
	go run ./cmd/server

# Run the gRPC client demo.
run-client:
	go run ./cmd/client

## ──────────────────────────────────────────────
## Quality
## ──────────────────────────────────────────────

# Run all tests (excludes helper-only packages like testutil).
test: proto
	@echo "==> Running tests..."
	go test $$(go list ./cmd/... ./internal/... | grep -v -E '/testutil|cmd/(client|server)$$') -race -v -count=1
	@echo "==> Done."

# Run go vet.
vet:
	go vet ./cmd/... ./internal/...

# Run linter (requires golangci-lint).
lint:
	golangci-lint run ./cmd/... ./internal/...

## ──────────────────────────────────────────────
## Cleanup
## ──────────────────────────────────────────────

# Remove build artifacts and generated files.
clean:
	rm -rf $(BIN_DIR)
	rm -rf pkg/gen
	rm -rf pkg/gen-ts
	rm -rf pkg/gen-php

## ──────────────────────────────────────────────
## Dependencies
## ──────────────────────────────────────────────

# Install required tools.
deps:
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
