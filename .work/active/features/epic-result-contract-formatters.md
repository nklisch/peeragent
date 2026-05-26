---
id: epic-result-contract-formatters
kind: feature
stage: review
tags: [infra]
parent: epic-result-contract
depends_on: [epic-result-contract-model]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Result Formatters

## Brief

This feature renders the shared result model as JSON by default and human-readable text when requested with `--text`. It keeps rendering separate from execution.

The feature exists so Claude receives structured output while humans still have an explicit readable mode.

## Epic Context

- Parent epic: `epic-result-contract`
- Position in epic: consumes result model.

## Foundation References

- `docs/CONTRACT.md` — human-readable and JSON output shapes.

## Architectural Choice

Add formatting functions to `internal/result`: JSON is compact one-line output for Claude, and text is an explicit human-readable mode. The CLI chooses based on `Request.JSON`.

Alternative considered: keep JSON only for now. Rejected because `--text` already exists in input parsing and should have meaningful behavior.

## Implementation Units

### Unit 1: JSON Formatter

**File**: `internal/result/format.go`

```go
func FormatJSON(Result) ([]byte, error)
```

### Unit 2: Text Formatter

**File**: `internal/result/format.go`

```go
func FormatText(Result) string
```

**Acceptance Criteria**:
- [ ] JSON formatter emits valid JSON.
- [ ] Text formatter includes status, summary, metadata, verification, changed files, and details when present.
- [ ] CLI uses JSON by default and text when `--text` is supplied.

## Implementation Order

1. Add formatter functions and tests.
2. Replace `main.go` result struct/write function with `internal/result`.
3. Run tests.

## Testing

### Unit Tests

Test JSON validity and representative text sections.

## Risks

Text output is secondary. Keep it readable but do not over-invest in terminal formatting.

## Implementation Notes

- Added `result.FormatJSON`.
- Added `result.FormatText`.
- Replaced ad hoc `main.go` JSON struct with the shared result model and formatters.
- Added formatter tests.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
