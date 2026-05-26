---
id: epic-wrapper-cli-blocking-exec
kind: feature
stage: drafting
tags: [infra]
parent: epic-wrapper-cli
depends_on: [epic-wrapper-cli-prompt]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Blocking Codex Exec

## Brief

This feature implements the blocking delegation path. The wrapper discovers the local `codex` executable, runs `codex exec --cd <cwd>` with the constructed prompt, waits for completion, and returns a minimal JSON result.

The feature exists to make Codex Implement actually delegate work in the same checkout. It does not implement the final result schema, async jobs, or full safety-mode matrix.

## Epic Context

- Parent epic: `epic-wrapper-cli`
- Position in epic: consumes prompt construction and completes the core blocking path.

## Foundation References

- `docs/SPEC.md` — first implementation path uses `codex exec`.
- `docs/ARCHITECTURE.md` — blocking flow.
- `docs/CONTRACT.md` — default invocation.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

