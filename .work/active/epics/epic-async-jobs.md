---
id: epic-async-jobs
kind: epic
stage: drafting
tags: [infra]
parent: null
depends_on: [epic-result-contract]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Async Jobs

## Brief

This epic adds explicit long-running job support without changing the default blocking experience. Claude can start a Codex implementation job in async mode, receive a job id, and later inspect status, retrieve results, or cancel the job.

The capability delivered here includes local job metadata, logs, process tracking, status/result/cancel commands, and result reuse through the same contract as blocking mode. Async output stays concise by default and log-backed for diagnostics.

This epic does not turn Codex Implement into a dashboard or general job-control system. Async remains an opt-in mode for long-running work.

## Foundation References

- `docs/VISION.md` — optional async mode.
- `docs/SPEC.md` — async invocation mode.
- `docs/ARCHITECTURE.md` — async flow and extension points.
- `docs/CONTRACT.md` — async job contract and status/result/cancel commands.

## Anticipated Child Features

- `--async` launch behavior.
- Job id and metadata storage.
- Status and result lookup commands.
- Cancellation support.
- Async log management.

<!-- The design pass on each child feature will fill in real specifics. -->

