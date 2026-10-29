---
id: gate-security-stale-job-lock
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

# Recover stale async job locks safely

## Severity
Low

## Domain
Availability / process lifecycle

## Location
`internal/jobs/store.go:153`

## Evidence
```go
file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
...
return fmt.Errorf("acquire job lock %s: timed out", id)
```

A crashed lock holder leaves a file that permanently wedges future terminal updates for the job.

## Remediation direction
Verify holder liveness and safely recover stale locks after timeout, with diagnostics and race coverage.

## Resolution (2026-10-29)

Kernel-backed guard locks serialize writers and release on process exit. Owner records include boot/PID/start identity; proven-dead owner files are recovered under the guard, including concurrent recoverers. `TestStaleLockRecoverySerializesContenders` covers stale identity and serialized updates.

Implemented in the friction batch; release and installed binaries remain with the owner after review.
