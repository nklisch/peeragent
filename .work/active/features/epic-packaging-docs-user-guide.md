---
id: epic-packaging-docs-user-guide
kind: feature
stage: drafting
tags: [docs]
parent: epic-packaging-docs
depends_on: [epic-packaging-docs-build-artifacts]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# User Guide

## Brief

This feature adds the practical user documentation for installing and using Codex Implement as a Claude Code plugin. It should cover prerequisites, installation shape, blocking delegation, async jobs, full-access escalation, effort selection, JSON/text output, and troubleshooting for Codex CLI availability or authentication issues.

The capability delivered here is operational clarity for a developer trying the plugin from the repository. It should align the public README and the bundled skill instructions so Claude and humans see the same defaults.

This feature does not introduce new CLI modes or promise unsupported behavior such as default worktree isolation.

## Epic Context

- Parent epic: `epic-packaging-docs`
- Position in epic: consumes build artifact decisions so setup instructions point at the real distributable path.

## Foundation References

- `docs/VISION.md` — autonomous implementor expectations.
- `docs/SPEC.md` — runtime assumptions and modes.
- `docs/CONTRACT.md` — CLI synopsis and result behavior.
- `skills/codex-implement/SKILL.md` — Claude-facing invocation guidance.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
