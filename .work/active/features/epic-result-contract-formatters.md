---
id: epic-result-contract-formatters
kind: feature
stage: drafting
tags: [infra]
parent: epic-result-contract
depends_on: [epic-result-contract-model]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Result Formatters

## Brief

This feature renders the shared result model as JSON by default and human-readable text when requested with `--text`. It keeps rendering separate from execution.

The feature exists so Claude receives structured output while humans still have an explicit readable mode.

## Epic Context

- Parent epic: `epic-result-contract`
- Position in epic: consumes result model.

## Foundation References

- `docs/CONTRACT.md` — human-readable and JSON output shapes.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

