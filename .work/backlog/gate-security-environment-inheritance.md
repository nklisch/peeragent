---
id: gate-security-environment-inheritance
kind: story
tags: [security]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: security
created: 2026-07-12
updated: 2026-10-22
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


## Terminal-control incident and scoped correction

Godot-41's October 22 incident report showed a delegated Codex invoking Herdr
prompt input against the launching session. The rollout's command events were
inspected read-only; no terminal/session was contacted. This follow-up removes
inherited terminal-control addresses at the shared target runner boundary by
default, with a recorded `--keep-terminal-env` override. The canonical list and
scope are in `docs/TERMINAL_ENV.md`; implementation/qualification is tracked by
`terminal-environment-isolation` (Isolate delegated agents from terminal control).

This addresses terminal-control inheritance only. The broader question of
inheriting unrelated configuration/secrets remains open; authentication and
ordinary settings deliberately remain available in this patch.
