---
id: epic-result-contract-execution-details
kind: feature
stage: implementing
tags: [infra]
parent: epic-result-contract
depends_on: [epic-result-contract-formatters]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Execution Detail Mapping

## Brief

This feature maps Codex execution outcomes into the result model. It captures success/failure status, exit code, stdout/stderr details, concise failure summaries, and empty changed-file/verification arrays until richer extraction exists.

The feature exists so Claude gets a predictable result even when Codex fails, is missing, or exits non-zero.

## Epic Context

- Parent epic: `epic-result-contract`
- Position in epic: consumes formatters and completes the result contract for blocking execution.

## Foundation References

- `docs/CONTRACT.md` — exit codes and failure reporting.
- `docs/SPEC.md` — output requirements.

## Architectural Choice

Create a small mapping function in `cmd/codex-implement` for now. It translates the already-local execution result and error into the shared result schema. If async later needs the same mapping, it can be extracted into an internal package then.

Alternative considered: create a new mapper package immediately. Rejected because the mapping currently depends on CLI request metadata and would add abstraction before reuse exists.

## Implementation Units

### Unit 1: Execution Result Mapping

**File**: `cmd/codex-implement/main.go`

```go
func resultFromExecution(req input.Request, execResult codex.Result, execErr error) result.Result
```

**Implementation Notes**:
- Success when no error and exit code is 0.
- Failed when executor returns an error or Codex exits non-zero.
- Details include stdout/stderr excerpts currently available.
- Changed files and verification are initialized as empty arrays.

**Acceptance Criteria**:
- [ ] Success, non-zero exit, and executor error paths map correctly.
- [ ] Main uses the mapper.

## Implementation Order

1. Add mapper function.
2. Add focused main package tests for mapper behavior.
3. Run tests.

## Testing

### Unit Tests

Test success, non-zero exit, executor error, and access metadata mapping.

## Risks

Changed-file and verification extraction are still placeholders. That is acceptable until Codex final output parsing is based on real examples.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
