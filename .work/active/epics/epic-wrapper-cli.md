---
id: epic-wrapper-cli
kind: epic
stage: drafting
tags: [infra]
parent: null
depends_on: [epic-plugin-foundation]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Wrapper CLI

## Brief

This epic delivers the blocking `codex-implement` command that Claude calls when delegating implementation work. The wrapper accepts arbitrary task text, resolves the working directory, checks Codex CLI availability, constructs the Codex prompt, and invokes `codex exec` in the current checkout.

The capability delivered here is the core delegation path: Claude calls one command, Codex works in the same repository, and the wrapper returns enough information for Claude to continue. Blocking mode is the default and is treated as the primary product path.

This epic does not finalize the permission model, full result schema, or async job control. It provides the executable foundation those capabilities depend on.

## Foundation References

- `docs/VISION.md` — default execution mode and success criteria.
- `docs/SPEC.md` — Codex CLI strategy and invocation modes.
- `docs/ARCHITECTURE.md` — wrapper role, prompt construction, and blocking flow.
- `docs/CONTRACT.md` — CLI synopsis and working-directory contract.

## Anticipated Child Features

- CLI argument parsing for task text and prompt files.
- Working-directory resolution and validation.
- Codex CLI discovery and compatibility checks.
- Prompt construction for autonomous implementation.
- Blocking `codex exec` invocation.

<!-- The design pass on each child feature will fill in real specifics. -->

