.PHONY: all proto proto-path lint-proto build run-server run-client example-lb test test-cover bench vet lint gocyclo clean deps deps-cpp header init

# Binary output directory.
BIN_DIR := bin

# Original module path in this template repository.
TEMPLATE_MODULE := github.com/H0llyW00dzZ/grpc-template

# Test packages (excludes test helpers and cmd mains, matching CI).
TEST_PKGS := $(shell go list ./cmd/... ./internal/... | grep -v -E '/testutil|cmd/(client|server)$$')

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
## Project Initialisation
## ──────────────────────────────────────────────

# Initialise a new project from this template.
#
# MODULE is derived automatically from the git remote host + git config user.name + project dir name:
#   <git-host>/<git-username>/<project-name>
#
# The git host is detected from `git remote get-url origin` — works with
# GitHub, GitLab, Gitea, Bitbucket, or any self-hosted git server.
# Falls back to github.com if no remote is configured.
#
# The project name is sanitised for a valid Go module path:
#   - lowercase everything
#   - spaces → hyphens
#   - keep only alphanumeric, hyphen, underscore, dot
#   - collapse multiple hyphens into one
#   - remove leading and trailing hyphens
#   - this prevents broken module paths like "My Awesome Project"
#
# Usage (auto-derive from git remote + git config):
#   make init DIR=../my-grpc-project
#
# Usage (explicit override — bypasses all auto-detection):
#   make init MODULE=github.com/yourorg/yourproject DIR=../my-grpc-project
#   make init MODULE=gitlab.com/yourorg/yourproject   # in-place
#
# What it does:
#   1. Detects git host from origin remote URL (HTTPS or SSH format).
#   2. Derives MODULE from <host>/<git-username>/<project-name> (if MODULE not set).
#   3. Copies the entire template to DIR (if given) or works in-place.
#   4. Replaces every occurrence of the template module path with MODULE
#      across all .go, .proto, and .yaml files (including buf.gen.yaml).
#   5. Rewrites the template project name (grpc-template) with the actual
#      project name across Kubernetes manifests, deploy docs, and Dockerfile.
#   6. Updates go.mod via `go mod edit -module`.
#   7. Re-initialises git: removes .git, runs git init, and creates
#      an initial commit so the developer starts with a clean history.
#   8. Runs `go mod tidy` to sync the dependency graph.
#
# Signed commits:
#   Set SIGNED=1 to GPG/SSH sign the initial commit (requires git signing to be configured).
#   make init SIGNED=1 DIR=../my-grpc-project
MODULE ?=
DIR    ?= .
SIGNED ?=
init: header
	@set -e; \
	UNAME_S=$$(uname -s); \
	GIT_USER=$$(git config user.name 2>/dev/null || true); \
	if [ -z "$$GIT_USER" ]; then \
		echo "ERROR: git config user.name is not set. Run: git config --global user.name 'yourname'"; \
		exit 1; \
	fi; \
	GIT_USER=$$(echo "$$GIT_USER" | tr ' ' '-'); \
	REMOTE_URL=$$(git remote get-url origin 2>/dev/null || true); \
	if [ -n "$$REMOTE_URL" ]; then \
		GIT_HOST=$$(echo "$$REMOTE_URL" | sed -E 's|https?://([^/:]+)/.*|\1|; s|git@([^:]+):.*|\1|'); \
	else \
		GIT_HOST="github.com"; \
	fi; \
	if [ "$(DIR)" = "." ]; then \
		PROJECT=$$(basename "$$(pwd)"); \
	else \
		PROJECT=$$(basename "$(DIR)"); \
		if [ -d "$(DIR)" ] && [ "$$(ls -A '$(DIR)' 2>/dev/null)" ]; then \
			echo "ERROR: $(DIR) already exists and is not empty. Choose a different path."; \
			exit 1; \
		fi; \
	fi; \
	PROJECT=$$(echo "$$PROJECT" \
		| tr '[:upper:]' '[:lower:]' \
		| tr ' ' '-' \
		| tr -c '[:alnum:]-._' '-' \
		| tr -s '-' \
		| sed 's/^-//; s/-$$//'); \
	if [ -z "$$PROJECT" ]; then \
		echo "ERROR: Project name became empty after sanitization. Please choose a different directory name."; \
		exit 1; \
	fi; \
	if [ -z "$(MODULE)" ]; then \
		RESOLVED_MODULE="$$GIT_HOST/$$GIT_USER/$$PROJECT"; \
	else \
		RESOLVED_MODULE="$(MODULE)"; \
	fi; \
	echo "==> Initialising project: $$RESOLVED_MODULE"; \
	ORIG_PWD="$$(pwd)"; \
	if [ "$(DIR)" != "." ]; then \
		trap 'RET=$$?; if [ $$RET -ne 0 ]; then echo ""; echo "==> Failed or interrupted — cleaning up $(DIR)..."; cd "$$ORIG_PWD" && rm -rf "$(DIR)"; fi; exit $$RET' EXIT INT TERM; \
		echo "==> Copying template to $(DIR) (excluding .git)..."; \
		mkdir -p "$(DIR)"; \
		rsync -a --exclude='.git' . "$(DIR)/"; \
		cd "$(DIR)"; \
	fi; \
	echo "==> Rewriting module path in source files..."; \
	TEMPLATE_PROJECT="grpc-template"; \
	if [ "$$UNAME_S" = "Darwin" ]; then \
		find . -type f \( -name '*.go' -o -name '*.proto' -o -name '*.yaml' -o -name '*.yml' -o -name 'Makefile' \) \
			-not -path './.git/*' \
			-exec sed -i '' "s|$(TEMPLATE_MODULE)|$$RESOLVED_MODULE|g" {} +; \
		find . -type f \( -name '*.yaml' -o -name '*.yml' -o -name '*.md' -o -name 'Dockerfile' -o -name 'Makefile' \) \
			-not -path './.git/*' \
			-exec sed -i '' "s|$$TEMPLATE_PROJECT|$$PROJECT|g" {} +; \
	else \
		find . -type f \( -name '*.go' -o -name '*.proto' -o -name '*.yaml' -o -name '*.yml' -o -name 'Makefile' \) \
			-not -path './.git/*' \
			-exec sed -i "s|$(TEMPLATE_MODULE)|$$RESOLVED_MODULE|g" {} +; \
		find . -type f \( -name '*.yaml' -o -name '*.yml' -o -name '*.md' -o -name 'Dockerfile' -o -name 'Makefile' \) \
			-not -path './.git/*' \
			-exec sed -i "s|$$TEMPLATE_PROJECT|$$PROJECT|g" {} +; \
	fi; \
	echo "==> Updating go.mod..."; \
	go mod edit -module "$$RESOLVED_MODULE"; \
	echo "==> Running go mod tidy..."; \
	go mod tidy; \
	echo "==> Re-initialising git..."; \
	rm -rf .git; \
	git init; \
	git add -A; \
	COMMIT_FLAGS=""; \
	if [ -n "$(SIGNED)" ]; then \
		SIGN_KEY=$$(git config user.signingkey 2>/dev/null || true); \
		GPG_FORMAT=$$(git config gpg.format 2>/dev/null || true); \
		if [ -z "$$SIGN_KEY" ] && [ -z "$$GPG_FORMAT" ]; then \
			echo "ERROR: SIGNED=1 requires a signing key. Configure one with:"; \
			echo "  gpg  : git config --global user.signingkey <key-id>"; \
			echo "  ssh  : git config --global user.signingkey ~/.ssh/id_ed25519.pub"; \
			echo "         git config --global gpg.format ssh"; \
			exit 1; \
		fi; \
		COMMIT_FLAGS="-S"; \
		echo "==> Signed commit enabled (format: $${GPG_FORMAT:-gpg})"; \
	fi; \
	git commit $$COMMIT_FLAGS -m "chore: initialise project from grpc-template"; \
	if [ "$(DIR)" != "." ]; then trap - EXIT INT TERM; fi; \
	echo ""; \
	echo "==> Done! Your project is ready at: $(DIR)"; \
	echo "    Module : $$RESOLVED_MODULE"; \
	echo "    Signed : $${COMMIT_FLAGS:+yes}$${COMMIT_FLAGS:-no}"; \
	if [ "$(DIR)" = "." ]; then \
		echo "    Next   : make deps && make proto && make run-server"; \
	else \
		echo "    Next   : cd $(DIR) && make deps && make proto && make run-server"; \
	fi


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
	@if [ -z "$(PROTO_PATH)" ]; then \
		echo "ERROR: PROTO_PATH is required. Usage: make proto-path PROTO_PATH=proto/storage/v1"; \
		exit 1; \
	fi
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
	@mkdir -p $(BIN_DIR)
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

# Run the load balancing demo (starts 3 servers, round-robin client).
example-lb: header
	@echo "==> Running load balancing demo..."
	go run ./examples/loadbalancing
	@echo "==> Done."

## ──────────────────────────────────────────────
## Quality
## ──────────────────────────────────────────────

# Run all tests (excludes helper-only packages like testutil).
test: header
	@echo "==> Running tests..."
	go test $(TEST_PKGS) -race -v -count=1
	@echo "==> Done."

# Run tests and evaluate coverage.
# Note: To view the detailed coverage report in your browser, run:
#   go tool cover -html=coverage.txt
test-cover: header
	@echo "==> Running tests with coverage..."
	go test $(TEST_PKGS) -race -v -coverprofile=coverage.txt -covermode=atomic -count=1
	go tool cover -func=coverage.txt
	@echo "==> Done. (To view in browser: go tool cover -html=coverage.txt)"

# Run benchmarks with memory allocation reporting.
# Usage:
#   make bench                          # run all benchmarks
#   make bench BENCH_FILTER=GetConfig   # run only matching benchmarks
BENCH_FILTER ?= .
bench: header
	@echo "==> Running benchmarks..."
	go test $(TEST_PKGS) -bench=$(BENCH_FILTER) -benchmem -benchtime=1s -run='^$$' -count=1 -timeout 300s
	@echo "==> Done."

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

# Run cyclomatic complexity analysis (requires gocyclo).
# Scans cmd/ and internal/ only — pkg/ is excluded because it contains
# generated protobuf code.
# Usage:
#   make gocyclo                    # report functions with complexity > 10
#   make gocyclo CYCLO_THRESHOLD=15 # custom threshold
CYCLO_THRESHOLD ?= 14
gocyclo: header
	@echo "==> Running gocyclo (threshold=$(CYCLO_THRESHOLD))..."
	gocyclo -over $(CYCLO_THRESHOLD) cmd/ internal/
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
	rm -rf pkg/gen-cpp

## ──────────────────────────────────────────────
## Dependencies
## ──────────────────────────────────────────────

# Install required tools.
deps: header
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest

# Install C++ protobuf and gRPC code-generation tools (system packages).
# These are needed to compile or locally invoke grpc_cpp_plugin.
# Usage: sudo make deps-cpp
deps-cpp: header
	@echo "==> Installing C++ protobuf and gRPC tools..."
	sudo apt-get install -y \
		protobuf-compiler \
		protobuf-compiler-grpc \
		libprotobuf-dev \
		libgrpc++-dev
	@echo "==> Done."
