---
id: epic-async-jobs-launch
kind: feature
stage: done
tags: [infra]
parent: epic-async-jobs
depends_on: [epic-async-jobs-store]
release_binding: 0.5.0
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Async Launch

## Brief

This feature implements `--async`. The wrapper creates a job record, starts a detached child process for the blocking implementation path, and immediately returns a `running` result with the job id.

The feature exists so Claude can intentionally start long-running implementation work without blocking the current tool call.

## Epic Context

- Parent epic: `epic-async-jobs`
- Position in epic: consumes the job store and produces running jobs.

## Foundation References

- `docs/SPEC.md` — async invocation mode.
- `docs/CONTRACT.md` — async launch behavior.

## Architectural Choice

Use a hidden `--job-run <id>` mode for the child process. Parent `--async` creates a job, starts the current executable with `--job-run <id>` plus the original task args, and immediately returns a `running` result. The child executes the normal blocking path and writes the final JSON result to the job's result path.

Alternative considered: shell out through `nohup`. Rejected because starting the current executable directly is more portable and easier to test.

## Implementation Units

### Unit 1: Async Flags

**File**: `internal/input/input.go`

Add `Async bool` and `JobRunID string`.

### Unit 2: Async Launch Flow

**File**: `cmd/codex-implement/main.go`

If `req.Async`, create a job, start a child process with `--job-run <id>`, store pid, and return `running` result.

### Unit 3: Child Job Run

**File**: `cmd/codex-implement/main.go`

If `req.JobRunID` is set, run blocking execution and write the final JSON result to the job result path.

## Implementation Order

1. Parse async flags.
2. Add launch helper.
3. Add job-run helper.
4. Test argument parsing and job result writing where practical.

## Testing

### Unit Tests

Cover async flag parsing. Process spawning is smoke-tested manually or by later integration tests.

## Risks

Detached child behavior varies by OS. The implementation should stay modest and file-backed rather than becoming a daemon.

## Implementation Notes

- Added `--async` and hidden `--job-run <id>` parsing.
- `--async` creates a job record, launches the current executable detached, and returns a `running` result with job id.
- `--job-run` executes the blocking path and writes the final JSON result to the job result path.
- Child stdout/stderr are redirected to the job log.

## Review

Approved. Async launch creates durable job state, starts a detached child process, returns a running result with job id, and keeps local job artifacts ignored by Git. Smoke testing verified a child job wrote a final result through the worktree error path.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
