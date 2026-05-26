---
id: epic-async-jobs-cancel
kind: feature
stage: drafting
tags: [infra]
parent: epic-async-jobs
depends_on: [epic-async-jobs-status-result]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Async Cancel

## Brief

This feature implements `--cancel <job-id>` as a best-effort local process stop. It marks the job cancelled and returns a `cancelled` result when cancellation succeeds or when the process is already gone.

The feature exists to give Claude an explicit way to stop async work without treating the job store as a daemon.

## Epic Context

- Parent epic: `epic-async-jobs`
- Position in epic: final lifecycle operation after launch/status/result.

## Foundation References

- `docs/CONTRACT.md` — `--cancel`.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

