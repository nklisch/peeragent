---
id: epic-async-jobs-status-result
kind: feature
stage: done
tags: [infra]
parent: epic-async-jobs
depends_on: [epic-async-jobs-launch]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Async Status And Result

## Brief

This feature implements `--status` and `--result`. Status reports local job state; result returns the final result file using the same result contract as blocking mode.

The feature exists so Claude can reconnect to async work without parsing logs manually.

## Epic Context

- Parent epic: `epic-async-jobs`
- Position in epic: consumes launched job records.

## Foundation References

- `docs/CONTRACT.md` — `--status` and `--result`.

## Architectural Choice

Implement explicit `--status <job-id>` and `--result <job-id>` commands. Status reads `job.json` and returns a normal result object; result reads `result.json` if present or reports the job as still running.

Alternative considered: allow omitted job ids and infer the latest job. Rejected for the first implementation because explicit ids avoid surprising cross-session behavior.

## Implementation Units

### Unit 1: Status/Result Flags

**File**: `internal/input/input.go`

Add `StatusJobID` and `ResultJobID`.

### Unit 2: Status/Result Handlers

**File**: `cmd/codex-implement/main.go`

Load from `jobs.NewStore(req.CWD)`. `--status` returns the metadata status. `--result` returns the saved result file when available.

## Implementation Order

1. Parse flags.
2. Add handlers.
3. Test parsing.
4. Smoke test against the existing async job store.

## Testing

### Unit Tests

Cover flag parsing. Use smoke tests for local file behavior.

## Risks

The contract allows optional job ids, but this implementation requires explicit ids until a latest-job policy is designed.

## Implementation Notes

Implemented `--status <job-id>` and `--result <job-id>` in the Go wrapper. Status loads `job.json` and maps local job lifecycle states into the result contract. Result loads and returns `result.json` unchanged for JSON output, or formats it through the text renderer when `--text` is requested. Missing result files report the job as still running.

Verification:

- `go test ./...`
- `go run ./cmd/codex-implement --status 20260526T034837Z-2da71b11`
- `go run ./cmd/codex-implement --result 20260526T034837Z-2da71b11`

## Review

Approved. The implementation keeps status/result lookup local to the job store, preserves the JSON result contract for stored results, and covers the new flag parsing plus lifecycle status mapping. Review verification passed with `go test ./...`.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
