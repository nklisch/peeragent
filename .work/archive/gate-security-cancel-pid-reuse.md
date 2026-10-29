---
id: gate-security-cancel-pid-reuse
kind: story
stage: done
tags: [security]
parent: null
depends_on: []
release_binding: null
gate_origin: security
created: 2026-07-12
updated: 2026-10-29
---

# Guard cancellation against PID reuse

## Severity
Low

## Domain
Process lifecycle

## Location
`internal/app/cancel.go:127`, `internal/jobs/launch_unix.go:11`

## Evidence
```go
pid, err := store.ReadPID(job.ID)
controllerErr = s.processController.TerminateAndWait(pid, ...)
```

A tracked process may exit and its PID/process-group id may be reused before cancellation signals it.

## Remediation direction
Use kernel-owned process handles where available, or record and verify process start identity before signaling.

## Resolution (2026-10-29)

Worker boot/PID/start identity is recorded and checked before cancellation enters process control and again before escalation. Legacy identities without a start time are never signalled. `TestCancelNeverSignalsMismatchedOrLegacyPID` covers both cases; live-worker cancellation remains covered through the process-control port. Identity verification and signalling remain separate kernel operations, so a narrow exit/reuse race is a known limit; no persistent kernel process handle is held across CLI invocations.

Implemented in the friction batch; release and installed binaries remain with the owner after review.
