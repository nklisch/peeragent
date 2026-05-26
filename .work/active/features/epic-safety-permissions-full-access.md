---
id: epic-safety-permissions-full-access
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

# Full Access Opt-In

## Brief

This feature adds explicit `--full-access` behavior. When selected, the wrapper uses Codex's full-access/bypass mode instead of the classifier-compatible defaults and reports that access posture in output.

The feature exists for trusted contexts where the caller intentionally wants maximum autonomy. It does not make full access implicit.

## Epic Context

- Parent epic: `epic-safety-permissions`
- Position in epic: explicit alternate to default permissions.

## Foundation References

- `docs/VISION.md` — explicit full-access mode.
- `docs/SPEC.md` — full access is not default.
- `docs/CONTRACT.md` — `--full-access`.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

