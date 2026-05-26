---
id: epic-async-jobs
kind: epic
stage: done
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

## Design Decisions

- **Should async mode ship immediately or be designed now and implemented after blocking works?** Design async now. Implementation follows after the blocking path and result contract exist.

## Decomposition

Split by lifecycle. Job storage defines the local state shape, launch creates tracked jobs, status/result reads those jobs through the result contract, and cancellation stops tracked processes when possible.

### Child features

- `epic-async-jobs-store` — local job directory, metadata, and result file shape — depends on: `[]`
- `epic-async-jobs-launch` — `--async` starts a detached wrapper job and returns running result — depends on: `[epic-async-jobs-store]`
- `epic-async-jobs-status-result` — `--status` and `--result` read job state/results — depends on: `[epic-async-jobs-launch]`
- `epic-async-jobs-cancel` — `--cancel <job-id>` best-effort process cancellation — depends on: `[epic-async-jobs-status-result]`

### Decomposition risks

Detached process handling differs across platforms. Keep the first implementation local and simple: metadata, pid, log/result files, and best-effort cancellation without promising robust daemon semantics.

## Review

Done. The async lifecycle now has local job metadata, detached launch, status lookup, stored result retrieval, and best-effort cancellation. Verification for child implementations included `go test ./...` plus smoke checks for async launch, status, result, and cancel behavior.

<!-- The design pass on each child feature will fill in real specifics. -->
