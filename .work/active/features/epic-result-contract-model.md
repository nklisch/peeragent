---
id: epic-result-contract-model
kind: feature
stage: drafting
tags: [infra, docs]
parent: epic-result-contract
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Result Model

## Brief

This feature creates the shared result model used by the CLI. The model defines status values, summary, changed files, verification entries, detail text, and metadata including cwd, access, profile, exit code, and job/session placeholders.

The feature exists so output formatting and execution mapping are not scattered through `main.go`.

## Epic Context

- Parent epic: `epic-result-contract`
- Position in epic: foundation result schema.

## Foundation References

- `docs/CONTRACT.md` — result shape and statuses.
- `docs/SPEC.md` — output requirements.

## Design Decisions

- **Default output**: JSON by default.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

