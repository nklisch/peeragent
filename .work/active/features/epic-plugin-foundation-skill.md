---
id: epic-plugin-foundation-skill
kind: feature
stage: drafting
tags: [infra, docs]
parent: epic-plugin-foundation
depends_on: [epic-plugin-foundation-entrypoint]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Codex Implement Skill

## Brief

This feature creates `skills/codex-implement/SKILL.md`, the Claude-facing delegation instructions. The skill explains when Claude should delegate implementation to Codex, how to invoke the wrapper, how to pass arbitrary task text, and how to interpret the structured result.

The feature exists to make delegation feel natural instead of exposing a large Codex command surface. It does not implement wrapper internals or async behavior.

## Epic Context

- Parent epic: `epic-plugin-foundation`
- Position in epic: consumer of the command entrypoint; its instructions must match the real executable surface.

## Foundation References

- `docs/VISION.md` — Claude as primary collaborator and Codex as autonomous implementor.
- `docs/SPEC.md` — skill name and invocation shape.
- `docs/ARCHITECTURE.md` — skill role.
- `docs/CONTRACT.md` — skill contract and CLI synopsis.

## Design Decisions

- **Distribution posture**: Distributable Claude Code plugin from day one.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

