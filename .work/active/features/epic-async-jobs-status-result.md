---
id: epic-async-jobs-status-result
kind: feature
stage: drafting
tags: [infra]
parent: epic-async-jobs
depends_on: [epic-async-jobs-launch]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Async Status And Result

## Brief

This feature implements `--status` and `--result`. Status reports local job state; result returns the final result file using the same result contract as blocking mode.

The feature exists so Claude can reconnect to async work without parsing logs manually.

## Epic Context

- Parent epic: `epic-async-jobs`
- Position in epic: consumes launched job records.

## Foundation References

- `docs/CONTRACT.md` — `--status` and `--result`.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

