---
id: epic-async-jobs-launch
kind: feature
stage: drafting
tags: [infra]
parent: epic-async-jobs
depends_on: [epic-async-jobs-store]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Async Launch

## Brief

This feature implements `--async`. The wrapper creates a job record, starts a detached child process for the blocking implementation path, and immediately returns a `running` result with the job id.

The feature exists so Claude can intentionally start long-running implementation work without blocking the current tool call.

## Epic Context

- Parent epic: `epic-async-jobs`
- Position in epic: consumes the job store and produces running jobs.

## Foundation References

- `docs/SPEC.md` — async invocation mode.
- `docs/CONTRACT.md` — async launch behavior.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

