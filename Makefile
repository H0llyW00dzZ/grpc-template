.PHONY: all proto proto-path lint-proto build run-server run-client test test-cover vet lint clean deps header

# Binary output directory.
BIN_DIR := bin

# Print ASCII art banner.
header:
	@printf '%s\n' '       ______ ______  _____   _____                        _         _        '
	@printf '%s\n' '       | ___ \| ___ \/  __ \ |_   _|                      | |       | |       '
	@printf '%s\n' '  __ _ | |_/ /| |_/ /| /  \/   | |  ___  _ __ ___   _ __  | |  __ _ | |_  ___ '
	@printf '%s\n' ' / _` ||    / |  __/ | |       | | / _ \| '"'"'_ ` _ \ | '"'"'_ \ | | / _` || __|/ _ \'
	@printf '%s\n' '| (_| || |\ \ | |    | \__/\   | ||  __/| | | | | || |_) || || (_| || |_|  __/'
	@printf '%s\n' ' \__, |\_| \_|\_|     \____/   \_/ \___||_| |_| |_|| .__/ |_| \__,_| \__|\___|'
	@printf '%s\n' '  __/ |                                            | |                        '
	@printf '%s\n' ' |___/   by H0llyW00dzZ (@github.com/H0llyW00dzZ)  |_|                        '
	@echo ""

# Default target.
all: header proto build

## ──────────────────────────────────────────────
## Proto Generation
## ──────────────────────────────────────────────

# Generate Go code from proto files using buf.
proto: header
	@echo "==> Generating proto..."
	buf generate
	@echo "==> Done."

# Generate code for a specific proto path.
# Usage: make proto-path PROTO_PATH=proto/storage/v1
proto-path: header
	@echo "==> Generating proto for $(PROTO_PATH)..."
	buf generate --path $(PROTO_PATH)
	@echo "==> Done."

# Lint proto files.
lint-proto: header
	@echo "==> Linting proto files..."
	buf lint
	@echo "==> Done."

## ──────────────────────────────────────────────
## Build
## ──────────────────────────────────────────────

# Build server and client binaries.
build: header
	@echo "==> Building server..."
	go build -o $(BIN_DIR)/server ./cmd/server
	@echo "==> Building client..."
	go build -o $(BIN_DIR)/client ./cmd/client
	@echo "==> Done."

## ──────────────────────────────────────────────
## Run
## ──────────────────────────────────────────────

# Run the gRPC server.
run-server: header
	go run ./cmd/server

# Run the gRPC client demo.
run-client: header
	go run ./cmd/client

## ──────────────────────────────────────────────
## Quality
## ──────────────────────────────────────────────

# Run all tests (excludes helper-only packages like testutil).
test: header
	@echo "==> Running tests..."
	go test $$(go list ./cmd/... ./internal/... | grep -v -E '/testutil|cmd/(client|server)$$') -race -v -count=1
	@echo "==> Done."

# Run tests and evaluate coverage.
# Note: To view the detailed coverage report in your browser, run:
#   go tool cover -html=coverage.out
test-cover: header
	@echo "==> Running tests with coverage..."
	go test $$(go list ./cmd/... ./internal/... | grep -v -E '/testutil|cmd/(client|server)$$') -coverprofile=coverage.out
	go tool cover -func=coverage.out
	@echo "==> Done. (To view in browser: go tool cover -html=coverage.out)"

# Run go vet.
vet: header
	@echo "==> Running go vet..."
	go vet ./cmd/... ./internal/...
	@echo "==> Done."

# Run linter (requires golangci-lint).
lint: header
	@echo "==> Running golangci-lint..."
	golangci-lint run ./cmd/... ./internal/...
	@echo "==> Done."

## ──────────────────────────────────────────────
## Cleanup
## ──────────────────────────────────────────────

# Remove build artifacts and generated files.
clean: header
	rm -rf $(BIN_DIR)
	rm -rf pkg/gen
	rm -rf pkg/gen-ts
	rm -rf pkg/gen-php

## ──────────────────────────────────────────────
## Dependencies
## ──────────────────────────────────────────────

# Install required tools.
deps: header
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
