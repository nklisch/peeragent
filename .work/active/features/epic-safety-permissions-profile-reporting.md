---
id: epic-safety-permissions-profile-reporting
kind: feature
stage: drafting
tags: [security, infra]
parent: epic-safety-permissions
depends_on: [epic-safety-permissions-full-access, epic-safety-permissions-worktree]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Profile And Access Reporting

## Brief

This feature adds Codex profile pass-through and access-mode metadata. Claude can tell whether a run used default, full-access, or worktree posture and which Codex profile was requested.

The feature exists so safety decisions are visible in results and downstream logs. It does not redefine the result contract beyond adding minimal metadata fields.

## Epic Context

- Parent epic: `epic-safety-permissions`
- Position in epic: metadata and pass-through layer after permission modes exist.

## Foundation References

- `docs/CONTRACT.md` — `--profile` and access metadata.
- `docs/SPEC.md` — reporting what Codex did and preserving user control.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

