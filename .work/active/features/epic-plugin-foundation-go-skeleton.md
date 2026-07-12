---
id: epic-plugin-foundation-go-skeleton
kind: feature
stage: done
tags: [infra]
parent: epic-plugin-foundation
depends_on: []
release_binding: 0.5.0
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Go Skeleton

## Brief

This feature creates the Go project skeleton for the `codex-implement` wrapper. It establishes `go.mod`, `cmd/codex-implement/`, and internal package boundaries that later wrapper features build on.

The feature exists to make the wrapper a durable compiled CLI with minimal runtime assumptions. It does not implement Codex invocation, result formatting, or permission modes.

## Epic Context

- Parent epic: `epic-plugin-foundation`
- Position in epic: foundation feature for all CLI implementation work.

## Foundation References

- `docs/SPEC.md` — Go wrapper binary supplied by the plugin.
- `docs/ARCHITECTURE.md` — `cmd/codex-implement/` and `internal/` layout.

## Design Decisions

- **Wrapper runtime**: Go wrapper CLI, not Node or compiled Bun.

## Architectural Choice

Create a small Go module rooted at the plugin repository. The `cmd/codex-implement` package owns process startup, while `internal/` packages hold reusable wrapper logic as it appears in later features.

Alternative considered: place all code in `main.go`. Rejected because upcoming CLI parsing, Codex invocation, result formatting, and async jobs need separate package boundaries to stay testable.

Alternative considered: create the full package tree now. Rejected because empty abstraction folders add noise. Start with `cmd/codex-implement` and introduce `internal/` packages as behavior lands.

## Implementation Units

### Unit 1: Go Module

**File**: `go.mod`

```go
module github.com/nklisch/codex-implement

go 1.22
```

**Implementation Notes**:
- Use a stable module path matching the intended distributable plugin identity.
- Avoid dependencies until a feature proves one is needed.

**Acceptance Criteria**:
- [ ] `go.mod` exists.
- [ ] `go test ./...` can discover the module.

### Unit 2: Initial Command Package

**File**: `cmd/codex-implement/main.go`

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	_, _ = fmt.Fprintln(os.Stdout, `{"status":"blocked","summary":"codex-implement wrapper is not implemented yet"}`)
	return nil
}
```

**Implementation Notes**:
- The initial command should compile and return JSON-shaped output because JSON is the default contract.
- It should not pretend Codex invocation exists yet.

**Acceptance Criteria**:
- [ ] `go test ./...` passes.
- [ ] `go run ./cmd/codex-implement` exits successfully.
- [ ] Initial output is valid JSON.

## Implementation Order

1. Create `go.mod`.
2. Create `cmd/codex-implement/main.go`.
3. Run `gofmt`.
4. Run `go test ./...`.
5. Run the command and validate JSON output.

## Testing

### Module Validation

Run `go test ./...` to verify the module and packages compile.

### Command Smoke Test

Run `go run ./cmd/codex-implement` and parse stdout as JSON.

## Risks

The final module path may change if the repository name or owner changes. That is packaging metadata, not an architectural concern, and can be rolled forward when known.

## Implementation Notes

- Created `go.mod` for `github.com/nklisch/codex-implement`.
- Created `cmd/codex-implement/main.go` with a compiling JSON-shaped placeholder command.
- Verified the module with `go test ./...` and a command smoke test.

## Review

Approved. The Go skeleton compiles, the placeholder command emits valid JSON-shaped output, and the build introduces no unnecessary dependencies. This gives later wrapper work a clean compiled CLI base.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
