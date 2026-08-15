# Target Executor Adapter

A per-agent outbound adapter wraps one external coding-agent CLI behind a uniform port so the application layer does not know target-specific flags or output envelopes.

## Rationale

Every target package exposes the same core shape: `Result`, `Options`, an overridable `lookPath`, `Exec`, `ExecWithRunner`, and target-specific argument/result normalization. This keeps target CLI knowledge at the infrastructure edge and makes executor tests independent of installed binaries.

## Examples

- `internal/codex/exec.go:31` — resolves `codex`, invokes an injected `executil.Runner`, and normalizes JSONL.
- `internal/claude/exec.go:30` — resolves `claude` through the same seam and normalizes Claude JSON.
- `internal/gemini/exec.go:29` — resolves `agy`, carries resume metadata, and normalizes Antigravity output.
- `internal/zai/exec.go:30` — resolves `pi` and builds the fixed Z.AI invocation.

## When to use

Use this shape when adding another local agent CLI target. Keep binary names, flags, and output parsing inside its adapter package.

## When not to use

Do not put async lifecycle, persistence, shared access policy, or host orchestration in a target adapter; those belong in application services or jobs.

## Common violations

- Switching on target names in multiple consumers rather than deriving behavior from one authoritative target registry.
- Leaking target-specific flags into `internal/app` or CLI job-control handlers.
- Calling a real CLI in unit tests instead of using `ExecWithRunner`.
