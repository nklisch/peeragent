---
id: epic-mcp-server-job-control-application-services
kind: story
stage: implementing
tags: [infra]
parent: epic-mcp-server-job-control
depends_on: []
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# Extract async job-control services

## Scope

Move status, result retrieval, and cancellation from the CLI composition root into shared application services. Preserve repository-local lookup, exit-code-4 missing-job results, result-file races, terminal locking, TERM/KILL escalation, and idempotent cancellation behind injectable process-control seams.

## Acceptance criteria

- [ ] CLI status, result, and cancel behavior remains compatible.
- [ ] Shared methods return `result.Result` values without writing stdout or exiting.
- [ ] Running and every terminal state are covered.
- [ ] Cancellation race winners and TERM/KILL cleanup are deterministic under fake process control.
- [ ] After cancellation state commits, process termination and PID cleanup finish even if the caller context is cancelled.
- [ ] No package under `internal/` calls `os.Exit` or writes `os.Stdout`; validation enforces this adapter boundary.
- [ ] Missing jobs remain structured failures; corrupt storage remains an infrastructure error.
