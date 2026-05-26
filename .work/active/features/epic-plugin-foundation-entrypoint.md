---
id: epic-plugin-foundation-entrypoint
kind: feature
stage: review
tags: [infra]
parent: epic-plugin-foundation
depends_on: [epic-plugin-foundation-go-skeleton]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Wrapper Entrypoint

## Brief

This feature creates the `bin/codex-implement` executable that Claude Code calls from the plugin. The entrypoint locates or invokes the Go wrapper in a predictable way and passes through all arguments and standard input.

The feature exists to separate Claude Code's plugin executable surface from the Go implementation internals. It does not implement Codex command behavior or result formatting.

## Epic Context

- Parent epic: `epic-plugin-foundation`
- Position in epic: depends on the Go skeleton so the shim has a concrete binary/package target.

## Foundation References

- `docs/SPEC.md` — `bin/codex-implement` as the executable Claude calls.
- `docs/ARCHITECTURE.md` — executable entrypoint and Go wrapper implementation.
- `docs/CONTRACT.md` — CLI invocation contract.

## Architectural Choice

Use a POSIX shell shim at `bin/codex-implement`. The shim locates the plugin root, prefers a packaged binary under `dist/`, and falls back to `go run ./cmd/codex-implement` when running from a development checkout.

Alternative considered: make `bin/codex-implement` the compiled binary itself. Rejected for now because plugin development benefits from a stable script path while packaging decides where platform-specific binaries land.

Alternative considered: require users to build before use. Rejected because the development path should work immediately when Go is installed.

## Implementation Units

### Unit 1: Executable Shim

**File**: `bin/codex-implement`

```sh
#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PLUGIN_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

if [ -x "$PLUGIN_ROOT/dist/codex-implement" ]; then
  exec "$PLUGIN_ROOT/dist/codex-implement" "$@"
fi

exec go run "$PLUGIN_ROOT/cmd/codex-implement" "$@"
```

**Implementation Notes**:
- Preserve all arguments exactly.
- Let stdin flow through naturally.
- Use `exec` so process exit status is the wrapper exit status.
- The fallback assumes Go is installed only in development contexts.

**Acceptance Criteria**:
- [ ] `bin/codex-implement` exists and is executable.
- [ ] Running `bin/codex-implement` invokes the Go wrapper fallback.
- [ ] Arguments are passed through.

## Implementation Order

1. Create `bin/codex-implement`.
2. Mark it executable.
3. Run the shim and verify JSON output.
4. Run the shim with arbitrary args and verify it still succeeds.

## Testing

### Shim Smoke Test

Run `bin/codex-implement` and verify it emits the wrapper's JSON placeholder.

### Argument Pass-Through Smoke Test

Run `bin/codex-implement hello world` and verify the wrapper still executes.

## Risks

The final packaged binary layout may differ by platform. Keeping the shim small and explicit makes that packaging decision easy to revise later.

## Implementation Notes

- Created `bin/codex-implement`.
- The shim locates the plugin root from its own path.
- The shim prefers `dist/codex-implement` and falls back to `go run ./cmd/codex-implement`.
- Verified direct and argument-bearing invocations.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
