---
id: gate-security-job-file-permissions
kind: story
tags: [security]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: security
created: 2026-07-12
updated: 2026-07-12
---

# Restrict async job file permissions

## Severity
Low

## Domain
Data protection

## Location
`internal/jobs/store.go:67`, `internal/app/jobs.go:154`, `cmd/peeragent/main.go:178`

## Evidence
```go
os.MkdirAll(dir, 0o755)
AtomicWriteFile(path, content, 0o644)
```

Prompts, results, logs, and job metadata are readable by other local users who can traverse the checkout.

## Remediation direction
Default job directories to 0700 and sensitive files to 0600, with an explicit opt-in if shared runs are needed.
