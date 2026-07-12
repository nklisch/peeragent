# Per-Package Runner Test Double

Each target-adapter test drives `ExecWithRunner` through a recording `executil.Runner` and a cleanup-restored `lookPath` stub rather than executing a real CLI.

## Rationale

Target argument construction and output normalization need deterministic unit tests independent of PATH, authentication, network, or installed agent binaries. Keeping assertions near each adapter makes target-specific contracts visible.

## Examples

- `internal/codex/exec_test.go:230` — recording runner plus `stubLookPath` for Codex.
- `internal/claude/exec_test.go:121` — the same seam for Claude.
- `internal/gemini/exec_test.go:199` — the same seam for Antigravity.
- `internal/zai/exec_test.go:109` — the same seam for Pi/Z.AI.

## When to use

Use for every target adapter's unit tests; integration tests that intentionally execute a real binary belong elsewhere.

## When not to use

Do not mock application behavior through this runner or use it to claim end-to-end CLI availability.

## Common violations

- Forgetting to restore package-level `lookPath` in `t.Cleanup`.
- Sharing mutable runner state across parallel tests.
- Allowing byte-identical doubles to drift independently instead of using a focused shared test-support package when the duplication becomes costly.
