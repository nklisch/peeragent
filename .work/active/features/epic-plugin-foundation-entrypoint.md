---
id: epic-plugin-foundation-entrypoint
kind: feature
stage: drafting
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

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

