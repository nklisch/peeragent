---
id: epic-wrapper-cli-prompt
kind: feature
stage: drafting
tags: [infra]
parent: epic-wrapper-cli
depends_on: [epic-wrapper-cli-inputs]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Prompt Construction

## Brief

This feature wraps Claude's arbitrary task text in a stable Codex instruction envelope. The prompt tells Codex to work in the current repository, make direct code changes, run relevant verification, keep the final answer concise, and report blockers.

The feature exists so every delegated implementation pass has the same operating posture without forcing Claude to repeat boilerplate. It does not run Codex.

## Epic Context

- Parent epic: `epic-wrapper-cli`
- Position in epic: consumes collected task text and feeds blocking execution.

## Foundation References

- `docs/ARCHITECTURE.md` — prompt construction.
- `docs/VISION.md` — autonomous implementation delegation.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

