---
id: epic-async-jobs-store
kind: feature
stage: drafting
tags: [infra]
parent: epic-async-jobs
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Async Job Store

## Brief

This feature creates a local job store for async Codex Implement jobs. It defines where job metadata, logs, and final results live, and provides helpers to create and read job records.

The feature exists so async launch/status/result/cancel commands share one durable state model.

## Epic Context

- Parent epic: `epic-async-jobs`
- Position in epic: foundation for all async behavior.

## Foundation References

- `docs/ARCHITECTURE.md` — async flow.
- `docs/CONTRACT.md` — async job contract.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

