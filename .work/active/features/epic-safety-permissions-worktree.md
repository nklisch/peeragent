---
id: epic-safety-permissions-worktree
kind: feature
stage: drafting
tags: [security, infra]
parent: epic-safety-permissions
depends_on: [epic-safety-permissions-defaults]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Worktree Opt-In

## Brief

This feature defines the `--worktree` opt-in surface. The default product does not isolate work, but callers can request worktree behavior. If the first implementation does not create worktrees yet, it should fail clearly instead of silently ignoring the flag.

The feature exists to keep isolation as a deliberate escape hatch without complicating the default path.

## Epic Context

- Parent epic: `epic-safety-permissions`
- Position in epic: explicit alternate execution posture beside full access.

## Foundation References

- `docs/VISION.md` — no worktree unless explicitly requested.
- `docs/SPEC.md` — same checkout default.
- `docs/CONTRACT.md` — `--worktree`.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

