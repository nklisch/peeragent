---
id: gate-security-environment-inheritance
kind: story
tags: [security]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: security
created: 2026-07-12
updated: 2026-07-12
---

# Restrict environment inherited by delegated agents

## Severity
Low

## Domain
Secrets / configuration

## Location
`internal/app/service.go:215`

## Evidence
```go
cmd := exec.Command(executable, "--job-run", job.ID, "--cwd", job.CWD)
cmd.Dir = job.CWD
```

The detached child and target CLI inherit the full host/MCP environment.

## Remediation direction
Define an explicit environment policy that preserves required target authentication while avoiding unintended secret propagation.
