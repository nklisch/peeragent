---
id: epic-safety-permissions
kind: epic
stage: done
tags: [security, infra]
parent: null
depends_on: [epic-wrapper-cli]
release_binding: 0.5.0
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

## Design Decisions

- **Should default Codex invocation pass explicit permission defaults or rely on user config?** Pass explicit defaults. The wrapper should invoke Codex with stable classifier-compatible defaults, conceptually `--sandbox workspace-write --ask-for-approval on-request -c approvals_reviewer=auto_review`, unless the caller explicitly selects another mode.

## Decomposition

Split by safety capability. Default classifier-compatible invocation is the foundation. Full-access and worktree modes are explicit alternates. Profile/config pass-through and access reporting sit on top so callers and Claude can understand which posture was used.

### Child features

- `epic-safety-permissions-defaults` — explicit default `codex exec` permission flags — depends on: `[]`
- `epic-safety-permissions-full-access` — `--full-access` opt-in behavior — depends on: `[epic-safety-permissions-defaults]`
- `epic-safety-permissions-worktree` — `--worktree` opt-in flag and clear unsupported/error behavior if not implemented yet — depends on: `[epic-safety-permissions-defaults]`
- `epic-safety-permissions-profile-reporting` — profile pass-through and access-mode metadata — depends on: `[epic-safety-permissions-full-access, epic-safety-permissions-worktree]`

### Decomposition risks

The main risk is making "same checkout" and "full access" blur together. The implementation should keep the default same-worktree behavior while preserving Codex approval boundaries unless the caller explicitly chooses full access.

## Review

Approved. Default Codex execution now uses explicit classifier-compatible flags; full-access is an explicit bypass mode; worktree is recognized and fails clearly until implemented; and output reports access/profile metadata.

<!-- The design pass on each child feature will fill in real specifics. -->
