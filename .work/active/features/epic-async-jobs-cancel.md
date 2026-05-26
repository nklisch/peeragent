---
id: epic-async-jobs-cancel
kind: feature
stage: review
tags: [infra]
parent: epic-async-jobs
depends_on: [epic-async-jobs-status-result]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Async Cancel

## Brief

This feature implements `--cancel <job-id>` as a best-effort local process stop. It marks the job cancelled and returns a `cancelled` result when cancellation succeeds or when the process is already gone.

The feature exists to give Claude an explicit way to stop async work without treating the job store as a daemon.

## Epic Context

- Parent epic: `epic-async-jobs`
- Position in epic: final lifecycle operation after launch/status/result.

## Foundation References

- `docs/CONTRACT.md` — `--cancel`.

## Architectural Choice

Implement cancellation as a local best-effort operation over the existing job store. The wrapper loads `job.json`, attempts to kill the recorded PID only when the job is still marked `running`, writes a `cancelled` result file, clears the PID, saves the job as `cancelled`, and returns a `cancelled` result object.

Completed and failed jobs should not be rewritten as cancelled. For those states, `--cancel` should return the current status and summarize that the job is already terminal.

Alternative considered: process-group management with a daemon-like supervisor. Rejected for the first implementation because the architecture intentionally avoids a long-lived service and the contract only promises best-effort local cancellation.

## Implementation Units

### Unit 1: Cancel Flag

**File**: `internal/input/input.go`

Add `CancelJobID` and parse `--cancel <job-id>` without requiring task text.

### Unit 2: Cancel Handler

**File**: `cmd/codex-implement/main.go`

Load the job, preserve terminal jobs, best-effort kill the PID for running jobs, write a cancelled result, and save the updated job.

### Unit 3: Tests

**Files**: `internal/input/input_test.go`, `cmd/codex-implement/main_test.go`

Cover flag parsing and helper behavior for terminal status detection.

## Testing

- `go test ./...`
- Smoke test `--cancel` against an already-failed local async job.

## Implementation Notes

Implemented `--cancel <job-id>` parsing and the cancellation handler. Running jobs are marked cancelled after a best-effort PID kill, with a cancelled result written to the job result file. Terminal jobs return their existing terminal status and are not rewritten.

Verification:

- `go test ./...`
- `go run ./cmd/codex-implement --cancel 20260526T034837Z-2da71b11`

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
