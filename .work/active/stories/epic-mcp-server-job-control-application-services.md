---
id: epic-mcp-server-job-control-application-services
kind: story
stage: done
tags: [infra]
parent: epic-mcp-server-job-control
depends_on: []
release_binding: 0.5.0
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# Extract async job-control services

## Scope

Move status, result retrieval, and cancellation from the CLI composition root into shared application services. Preserve repository-local lookup, exit-code-4 missing-job results, result-file races, terminal locking, TERM/KILL escalation, and idempotent cancellation behind injectable process-control seams.

## Acceptance criteria

- [x] CLI status, result, and cancel behavior remains compatible.
- [x] Shared methods return `result.Result` values without writing stdout or exiting.
- [x] Running and every terminal state are covered.
- [x] Cancellation race winners and TERM/KILL cleanup are deterministic under fake process control.
- [x] After cancellation state commits, process termination and PID cleanup finish even if the caller context is cancelled.
- [x] No package under `internal/` calls `os.Exit` or writes `os.Stdout`; validation enforces this adapter boundary.
- [x] Missing jobs remain structured failures; corrupt storage remains an infrastructure error.

## Implementation notes
- Execution capability: highest, selected by the autopilot caller because cancellation crosses filesystem terminal races, detached process groups, and MCP context lifetime boundaries.
- Review weight: standard (autopilot default).
- Files changed: `internal/app/jobs.go`, `internal/app/cancel.go`, `internal/app/jobs_test.go`, `internal/app/cancel_test.go`, `internal/app/service.go`, `cmd/peeragent/main.go`.
- Tests added: application lifecycle tables for running/terminal/missing/corrupt state; cancellation race, idempotency, fake process-control, PID cleanup, and post-commit caller cancellation coverage.
- Design decisions: added a service-level working-directory resolver so CLI and MCP job requests share repository-local lookup normalization; added a context-free `ProcessController.TerminateAndWait` port so cleanup continues after cancellation state commits.
- Discrepancies from design: the existing repository centralizes child completion in `cmd/peeragent`; it now delegates the terminal transition to `app.FinishJob` rather than introducing an `internal/app/async.go` file.
- Verification: `go test ./internal/app ./cmd/peeragent` passed (47 tests); no package under `internal/` references `os.Exit` or `os.Stdout`.
- Adjacent issues parked: none.

## Review notes

- Effective review weight: standard; fresh-context deep review selected for cancellation and persisted terminal-state risk.
- Reviewer: GLM 5.2. It manually traced cancel-wins, completion-wins by status, completion-wins by result, and stale-cancel repair; all invariants and dedicated tests passed.
- Verdict: approve with comments. Removed extraction-residue wrappers from `cmd/peeragent/main.go` and moved tests to the authoritative `internal/app` functions before approval. Remaining boolean-pointer style comment was non-blocking.
