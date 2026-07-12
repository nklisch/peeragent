---
id: gate-security-gemini-prompt-argv
kind: story
tags: [security]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: security
created: 2026-07-12
updated: 2026-07-12
---

# Avoid exposing Gemini prompts through process argv

## Severity
Low

## Domain
Data protection

## Location
`internal/gemini/exec.go:98`

## Evidence
```go
return append(args, opts.Prompt)
```

The delegated prompt is visible through local process inspection on multi-user systems.

## Remediation direction
Use stdin if Antigravity supports it; otherwise document the local argv exposure and its trust assumptions.
