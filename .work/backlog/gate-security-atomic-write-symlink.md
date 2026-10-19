---
id: gate-security-atomic-write-symlink
kind: story
tags: [security]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: security
created: 2026-07-12
updated: 2026-10-19
---

# Harden atomic job writes against symlink redirection

## Severity
Low

## Domain
Data protection / filesystem race

## Location
`internal/jobs/store.go:199`

## Evidence
```go
tmp := path + ".tmp"
if err := os.WriteFile(tmp, content, perm); err != nil {
```

The temporary write follows a pre-planted symlink in a writable job directory.

Session-control review on 2026-10-19 confirmed that the older job writer still
uses this predictable temporary path. The new job/session locator index uses
`os.CreateTemp` and rename, so it avoids this particular temporary-file issue;
it does not harden the job writer. Directory components and reads in both the
store and index still follow symlinks. The index is not a general protection
for a shared or pre-existing state root. Exploitation depends on another actor
being able to alter those directories; no production exploit was observed.

## Remediation direction
Create temporary files without following symlinks, validate final paths, and use platform-safe rename semantics.
