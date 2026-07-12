---
id: epic-wrapper-cli-prompt
kind: feature
stage: done
tags: [infra]
parent: epic-wrapper-cli
depends_on: [epic-wrapper-cli-inputs]
release_binding: 0.5.0
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Prompt Construction

## Brief

This feature wraps Claude's arbitrary task text in a stable Codex instruction envelope. The prompt tells Codex to work in the current repository, make direct code changes, run relevant verification, keep the final answer concise, and report blockers.

The feature exists so every delegated implementation pass has the same operating posture without forcing Claude to repeat boilerplate. It does not run Codex.

## Epic Context

- Parent epic: `epic-wrapper-cli`
- Position in epic: consumes collected task text and feeds blocking execution.

## Foundation References

- `docs/ARCHITECTURE.md` — prompt construction.
- `docs/VISION.md` — autonomous implementation delegation.

## Architectural Choice

Create an `internal/prompt` package that builds the Codex implementation prompt from normalized task text. The prompt package is pure string construction so it is easy to test and reuse across blocking and async execution modes.

Alternative considered: inline prompt text in the execution package. Rejected because prompt policy is a product decision and should be visible/testable independently from process execution.

## Implementation Units

### Unit 1: Prompt Builder

**File**: `internal/prompt/prompt.go`

```go
func Build(taskText string) string
```

**Implementation Notes**:
- Preserve the original task text under a clearly marked section.
- Tell Codex to work in the current repository.
- Tell Codex to make direct edits and run relevant verification.
- Tell Codex to keep the final response concise.
- Tell Codex to report blockers instead of guessing.

**Acceptance Criteria**:
- [ ] Prompt contains the original task text.
- [ ] Prompt contains direct-edit instructions.
- [ ] Prompt contains verification instructions.
- [ ] Prompt contains blocker-reporting instructions.

### Unit 2: Main Integration

**File**: `cmd/codex-implement/main.go`

Use `prompt.Build(req.TaskText)` before returning placeholder output so execution can consume it next.

**Acceptance Criteria**:
- [ ] `go test ./...` passes.
- [ ] Command still emits valid JSON placeholder output.

## Implementation Order

1. Create prompt package.
2. Add prompt tests.
3. Wire `main.go` to build the prompt.
4. Run tests.

## Testing

### Unit Tests

Test that the built prompt includes task text and the durable behavioral instructions needed for autonomous implementation.

## Risks

Overly long prompt boilerplate would waste model context. Keep the envelope focused on operating posture, not project-specific recap.

## Implementation Notes

- Added `internal/prompt.Build`.
- Prompt preserves task text and adds direct-edit, verification, concise-result, and blocker-reporting instructions.
- Wired `main.go` to build the prompt before placeholder output.
- Added unit tests for task preservation and operating instructions.

## Review

Approved. The prompt builder is pure, tested, and focused on durable execution posture without overloading Codex with project recap. This unblocks blocking execution.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
