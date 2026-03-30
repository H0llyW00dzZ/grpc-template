---
description: Synchronize all documentation sources
---

# sync-docs

## Context for AI

This command is used by opencode to keep documentation consistent after code changes in the gRPC template. When triggered, proactively analyze changes (using git diff or tools) and update all relevant docs while preserving accuracy, Go conventions, and consistency with AGENTS.md.

## When to Run

Invoke after changes involving:
- New server options or interceptors
- Modified gRPC service APIs or behavior
- Updates to logging, auth, rate limiting, or recovery
- Proto changes (requires regeneration)
- Public API or functional option changes
- Test or build updates

## Documentation Sources to Sync

### 1. Primary Documentation
- `README.md` (source of truth for features, usage, project structure, make targets)
- `AGENTS.md` (build/lint/test commands, code style for agents)

### 2. Package-Level GoDoc
- `internal/server/doc.go`
- `internal/server/interceptor/doc.go`
- `internal/logging/logging.go` (and any new doc.go)
- Other `**/doc.go` files

### 3. Proto Documentation
- Update comments in `proto/*/v1/*.proto` files if new RPCs/services added
- Ensure README.md "Proto Collection" table stays current

### 4. Code Examples & Demos
- Update `cmd/client/main.go` if client demo changes
- Ensure README showcase/gifs and examples in doc.go remain accurate
- `internal/testutil/grpctest.go` if test helpers change

### 5. Other
- Makefile targets/comments if new commands added
- .github/workflows/test.yaml if CI changes

When adding new services/interceptors, update relevant doc.go, README "Adding a New Service" and "Customization" sections.

## Your Sync Workflow (follow exactly)

### Step 0: Analyze Changes
Use tools (grep, glob, git via bash if needed) to examine recent changes, git diff, impacted functions/options/protos.

### Step 1: Update Primary Docs
Revise `README.md` (features, structure, make targets, customization table) and `AGENTS.md` (commands, style guidelines, new patterns).

### Step 2: Update GoDoc
Update all `doc.go` files with accurate godoc, examples, option lists matching current code (see server/doc.go:9, interceptor/doc.go:6). Follow existing style: detailed # sections, code blocks, links.

### Step 3: Update Protos & Generated
If protos changed, run `make proto`. Update proto collection table in README.

### Step 4: Verify & Test
- Run `make proto` (if needed), `make lint`, `make vet`, `make test`
- Ensure examples in doc.go are valid and match code
- Check all public APIs are documented

### Step 5: Final Review
Output diffs/changes for review. Ensure no outdated references.

## Requirements
- Keep docs in sync with code (e.g., server options must match option.go and doc.go)
- Follow Go doc conventions (complete sentences, examples)
- NEVER edit pkg/gen/ files
- Maintain professional tone matching existing README/AGENTS.md
- Update AGENTS.md "Code Style Guidelines" if new patterns introduced
- Use existing patterns from neighboring doc.go files

## Commit Message
```
docs: sync documentation after [brief change desc]

- Update README.md and AGENTS.md
- Revise package docs in doc.go files
- Refresh proto collection table if applicable
- Verify with make lint vet test
```

> [!NOTE]
> Use question tool if needed to confirm before committing.

## Verification Checklist
- [ ] README.md updated
- [ ] AGENTS.md updated
- [ ] doc.go files updated with accurate godoc
- [ ] Proto table/README examples current
- [ ] `make lint`, `make vet`, `make test` pass
- [ ] No references to outdated features
