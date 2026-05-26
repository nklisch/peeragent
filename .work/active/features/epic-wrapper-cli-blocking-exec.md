---
id: epic-wrapper-cli-blocking-exec
kind: feature
stage: done
tags: [infra]
parent: epic-wrapper-cli
depends_on: [epic-wrapper-cli-prompt]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Blocking Codex Exec

## Brief

This feature implements the blocking delegation path. The wrapper discovers the local `codex` executable, runs `codex exec --cd <cwd>` with the constructed prompt, waits for completion, and returns a minimal JSON result.

The feature exists to make Codex Implement actually delegate work in the same checkout. It does not implement the final result schema, async jobs, or full safety-mode matrix.

## Epic Context

- Parent epic: `epic-wrapper-cli`
- Position in epic: consumes prompt construction and completes the core blocking path.

## Foundation References

- `docs/SPEC.md` — first implementation path uses `codex exec`.
- `docs/ARCHITECTURE.md` — blocking flow.
- `docs/CONTRACT.md` — default invocation.

## Architectural Choice

Create an `internal/codex` package that owns process execution. The package accepts a cwd and prompt, locates `codex` from `PATH`, runs `codex exec --cd <cwd> <prompt>`, captures stdout/stderr, and returns a structured execution result.

Alternative considered: run Codex directly from `main.go`. Rejected because safety permissions, result shaping, and async mode all need a reusable execution boundary.

The first implementation returns a minimal JSON result from `main.go`: success if Codex exits 0, failed otherwise, with stdout/stderr included. The richer output schema remains owned by `epic-result-contract`.

## Implementation Units

### Unit 1: Codex Executor

**File**: `internal/codex/exec.go`

```go
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

func Exec(ctx context.Context, cwd string, prompt string) (Result, error)
```

**Implementation Notes**:
- Use `exec.LookPath("codex")` so missing Codex produces a clear error.
- Use `exec.CommandContext` for future cancellation compatibility.
- Pass `exec --cd <cwd> <prompt>` as argv entries, not through a shell.
- Capture stdout and stderr separately.

**Acceptance Criteria**:
- [ ] Missing Codex can be represented as an error.
- [ ] Command arguments are built without shell interpolation.
- [ ] Exit code is captured for non-zero exits.

### Unit 2: Main Blocking Path

**File**: `cmd/codex-implement/main.go`

Use `codex.Exec(context.Background(), req.CWD, prompt.Build(req.TaskText))` and return JSON containing status, summary, stdout, stderr, cwd, and exit code.

**Implementation Notes**:
- Keep JSON manually small or use `encoding/json`.
- Do not add final result-contract fields beyond the minimal execution status.

**Acceptance Criteria**:
- [ ] `go test ./...` passes.
- [ ] `go run ./cmd/codex-implement <task>` attempts Codex execution.
- [ ] Missing/failed Codex exits non-zero and emits JSON.

## Implementation Order

1. Add `internal/codex`.
2. Add unit tests around command result handling where possible.
3. Wire `main.go`.
4. Run Go tests.
5. Smoke test with a harmless prompt if Codex is available.

## Testing

### Unit Tests

Use a small injectable command runner seam if direct process tests become brittle. The important behavior is argument construction, exit status capture, and missing-command reporting.

### Smoke Test

Run the wrapper with a harmless prompt. If live Codex execution is not practical, verify missing-command/error behavior and leave the live execution path to integration testing.

## Risks

Running live Codex during tests can be expensive and environment-dependent. Unit tests should avoid invoking real Codex; smoke tests can be manual or guarded.

## Implementation Notes

- Added `internal/codex` executor around `codex exec --cd <cwd> <prompt>`.
- Captures stdout, stderr, and exit code without shell interpolation.
- Wired `cmd/codex-implement` to call the executor and emit minimal JSON.
- Added unit coverage for argv construction.

## Review

Approved. The blocking execution path has a reusable process boundary, does not route through a shell, captures process output, and keeps the richer result contract for the downstream result epic. Live Codex execution was not run during review to avoid spending model/network resources.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
