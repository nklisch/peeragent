---
id: gate-security-cancel-pid-reuse
kind: story
tags: [security]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: security
created: 2026-07-12
updated: 2026-07-12
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
