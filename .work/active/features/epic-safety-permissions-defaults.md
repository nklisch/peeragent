---
id: epic-safety-permissions-defaults
kind: feature
stage: drafting
tags: [security, infra]
parent: epic-safety-permissions
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Default Permission Flags

## Brief

This feature makes the wrapper pass explicit classifier-compatible Codex defaults for normal blocking execution. The default invocation uses `--sandbox workspace-write`, `--ask-for-approval on-request`, and `-c approvals_reviewer=auto_review`.

The feature exists so behavior is stable even when the user's Codex config differs. It does not implement full-access or worktree modes.

## Epic Context

- Parent epic: `epic-safety-permissions`
- Position in epic: foundation safety behavior for all normal execution.

## Foundation References

- `docs/SPEC.md` — classifier-compatible Codex execution defaults.
- `docs/ARCHITECTURE.md` — permission model.
- `docs/CONTRACT.md` — default Codex invocation.

## Design Decisions

- **Default permissions**: Pass explicit defaults rather than relying on user config.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

