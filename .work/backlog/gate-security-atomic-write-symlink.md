---
id: gate-security-atomic-write-symlink
kind: story
tags: [security]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: security
created: 2026-07-12
updated: 2026-07-12
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

## Remediation direction
Create temporary files without following symlinks, validate final paths, and use platform-safe rename semantics.
