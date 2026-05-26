---
id: epic-result-contract-model
kind: feature
stage: implementing
tags: [infra, docs]
parent: epic-result-contract
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Result Model

## Brief

This feature creates the shared result model used by the CLI. The model defines status values, summary, changed files, verification entries, detail text, and metadata including cwd, access, profile, exit code, and job/session placeholders.

The feature exists so output formatting and execution mapping are not scattered through `main.go`.

## Epic Context

- Parent epic: `epic-result-contract`
- Position in epic: foundation result schema.

## Foundation References

- `docs/CONTRACT.md` — result shape and statuses.
- `docs/SPEC.md` — output requirements.

## Design Decisions

- **Default output**: JSON by default.

## Architectural Choice

Create `internal/result` as the single source of truth for result status, metadata, verification entries, and changed-file lists. Keep it data-only in this feature; formatting and execution mapping come next.

Alternative considered: keep the result struct in `main.go`. Rejected because downstream async jobs and formatters also need the same schema.

## Implementation Units

### Unit 1: Result Types

**File**: `internal/result/result.go`

```go
type Status string
const (
	StatusSuccess Status = "success"
	StatusFailed Status = "failed"
	StatusBlocked Status = "blocked"
	StatusCancelled Status = "cancelled"
	StatusRunning Status = "running"
)

type Result struct { ... }
```

**Implementation Notes**:
- Include `changed_files: []string`.
- Include `verification: []Verification`.
- Include metadata fields for cwd, access, profile, exit code, codex session, and job id.
- Keep JSON field names aligned with `docs/CONTRACT.md`.

**Acceptance Criteria**:
- [ ] Result types compile.
- [ ] JSON field names are stable.

## Implementation Order

1. Create `internal/result`.
2. Add a JSON marshal test for expected field names.
3. Run tests.

## Testing

### Unit Tests

Marshal a representative result and assert key JSON fields are present.

## Risks

The schema should be stable enough for Claude but conservative enough to evolve. Avoid fields that require inference the wrapper cannot reliably provide yet.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
