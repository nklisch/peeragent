---
id: epic-safety-permissions
kind: epic
stage: drafting
tags: [security, infra]
parent: null
depends_on: [epic-wrapper-cli]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Safety And Permissions

## Brief

This epic defines and implements the permission behavior that makes same-checkout delegation practical without silently bypassing safety boundaries. The default mode preserves classifier-compatible Codex behavior while allowing Codex to edit the current working tree.

The capability delivered here includes explicit full-access mode, explicit worktree mode, support for Codex profiles/config overrides, and clear reporting of the access posture used for each invocation. It keeps same-repo execution distinct from no-sandbox execution.

This epic does not try to make classifier review a security guarantee or replace Claude Code permissions. It makes high-risk modes visible and deliberate.

## Foundation References

- `docs/VISION.md` — classifier-compatible defaults and explicit full access.
- `docs/SPEC.md` — Codex execution defaults and safety boundaries.
- `docs/ARCHITECTURE.md` — permission model and extension points.
- `docs/CONTRACT.md` — `--full-access`, `--worktree`, `--profile`, and access metadata.

## Anticipated Child Features

- Default Codex permission configuration.
- Full-access opt-in flag.
- Worktree opt-in mode.
- Codex profile/config pass-through.
- Access-mode reporting in wrapper output.

## Design Decisions

- **Should default Codex invocation pass explicit permission defaults or rely on user config?** Pass explicit defaults. The wrapper should invoke Codex with stable classifier-compatible defaults, conceptually `--sandbox workspace-write --ask-for-approval on-request -c approvals_reviewer=auto_review`, unless the caller explicitly selects another mode.

<!-- The design pass on each child feature will fill in real specifics. -->
