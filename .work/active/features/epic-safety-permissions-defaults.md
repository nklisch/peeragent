---
id: epic-safety-permissions-defaults
kind: feature
stage: review
tags: [security, infra]
parent: epic-safety-permissions
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Default Permission Flags

## Brief

This feature makes the wrapper pass explicit classifier-compatible Codex defaults for normal blocking execution. The default invocation uses `--sandbox workspace-write`, `--ask-for-approval on-request`, and `-c approvals_reviewer=auto_review`.

The feature exists so behavior is stable even when the user's Codex config differs. It does not implement full-access or worktree modes.

## Epic Context

- Parent epic: `epic-safety-permissions`
- Position in epic: foundation safety behavior for all normal execution.

## Foundation References

- `docs/SPEC.md` — classifier-compatible Codex execution defaults.
- `docs/ARCHITECTURE.md` — permission model.
- `docs/CONTRACT.md` — default Codex invocation.

## Design Decisions

- **Default permissions**: Pass explicit defaults rather than relying on user config.

## Architectural Choice

Move Codex argv construction into a small options-driven function in `internal/codex`. The default options append `--sandbox workspace-write`, `--ask-for-approval on-request`, and `-c approvals_reviewer=auto_review` to every normal `codex exec` invocation.

Alternative considered: keep permission flags in `main.go`. Rejected because full-access, worktree, and profile modes all need to modify the same argv shape.

## Implementation Units

### Unit 1: Execution Options

**File**: `internal/codex/exec.go`

```go
type Options struct {
	CWD    string
	Prompt string
}

func Exec(ctx context.Context, opts Options) (Result, error)
```

**Implementation Notes**:
- Build args as `exec --cd <cwd> --sandbox workspace-write --ask-for-approval on-request -c approvals_reviewer=auto_review <prompt>`.
- Keep prompt as a single argv entry.
- Preserve test seam for argv assertions.

**Acceptance Criteria**:
- [ ] Unit test verifies default permission args are present.
- [ ] `go test ./...` passes.

### Unit 2: Main Integration

**File**: `cmd/codex-implement/main.go`

Call the new options-based executor.

**Acceptance Criteria**:
- [ ] CLI still compiles.
- [ ] Existing wrapper behavior is preserved except for safer Codex argv defaults.

## Implementation Order

1. Add `codex.Options`.
2. Update argv construction.
3. Update tests.
4. Wire `main.go`.
5. Run tests.

## Testing

### Unit Tests

Assert the executor builds the expected permission flags without invoking a shell.

## Risks

Codex CLI flag names can change over time. Keeping the defaults centralized makes future updates localized.

## Implementation Notes

- Added `codex.Options`.
- Centralized Codex argv construction in `internal/codex`.
- Default args now include `--sandbox workspace-write`, `--ask-for-approval on-request`, and `-c approvals_reviewer=auto_review`.
- Updated tests to assert the explicit default permission flags.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
