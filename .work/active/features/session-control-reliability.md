---
id: session-control-reliability
kind: feature
stage: review
tags: [infra, tests, docs]
parent: null
depends_on: []
release_binding: null
created: 2026-10-19
updated: 2026-10-19
---

# Reliable peer job controls and capacity recovery

Nathan requested fixes from the Voxlar sessions' 2026-10-19 evidence: unknown
flags started real work, job controls depended on the caller's directory, and
Codex model-capacity stops needed a resumable outcome and opt-in recovery.

## Scope and decisions

- Reject unknown options, empty/missing option values, and values swallowed from
  the next option before reading task input or starting a job. Usage exit is 2;
  `--` preserves dash-leading literal task text. The source-checkout shim
  preserves the compiled CLI's exits too; its child story records the
  `go run` mismatch found during verification and the fix.
- Keep repository-local job state as truth. Add a private per-user cache index
  of job ids and agent/session pairs to absolute working directories. Separate
  atomic entries support concurrent launches; explicit `--cwd` wins. Index
  writes warn rather than preventing legitimate work. Older unindexed jobs
  remain accessible from their original directory or with `--cwd`.
- Expose `capacity_exhausted` as a terminal result and persisted job status.
  Preserve session ids on failure. Only explicit Codex diagnostics qualify;
  successful completion after a transient diagnostic is not a capacity stop.
- Async Codex callers may opt into N additional attempts, defaulting to ten
  minutes between attempts. Retry the same session with a continuation prompt
  and original execution settings, under one job id. No session means no retry;
  ordinary failures are not retried. Keep separate raw logs per attempt.
- Keep Codex profile selection on the parent `exec` command when resuming;
  the resume subcommand does not accept that option itself (checked against
  the installed CLI's help without starting a session).
- Filter orphan-tool diagnostic lines only from the visible Codex stderr summary;
  preserve raw output in logs.

## Acceptance and handoff

Tests cover strict options before any fake-agent launch or state creation,
all valued-option missing/empty cases, dash-leading task text, cross-directory
wait/status/result/cancel and resume, explicit directory precedence, capacity
classification and one-job retries with a fake executable and shortened delay,
retry exhaustion, missing sessions, default no-retry, and ordinary failures.
Index tests cover concurrent writes, agent namespaces and Unix permissions;
service tests verify index-write failures do not prevent delegation. Application,
CLI and input test harnesses use a private index inherited by their
helper processes, so ordinary test runs do not pollute the user cache.

README, help, foundation docs, and both bundled skill copies describe the
new behavior. Final verification passed on the frozen implementation at `0198014`.
No merge, push, release, or committed binary refresh is authorized in this unit.

## Known limits

- Target-process hangs after a final message remain tracked separately in
  `async-codex-job-never-exits-after-final-message`; capacity handling still
  waits for target process exit.
- Clearing or losing the cache index requires `--cwd` again. Moving a checkout
  also leaves persisted absolute job paths stale, so `--cwd` alone does not
  recover result/wait behavior; tracked in `moved-job-result-path`. The index
  is a locator only.
- Real model calls and native macOS/Windows runtime behavior are not exercised
  by fake-agent checks on Linux.

## Final verification

On Linux, the full `make test` suite and `go vet ./...` passed on `0198014`.
Formatting, `git diff --check`, both shims' `sh -n`, curated-copy comparisons,
and the metadata/documentation checks extracted from `scripts/validate.sh`
passed. Both scratch-binary and actual source-shim fake-agent async smoke
checks passed, including retries, cross-directory controls/resume, usage exits,
and raw attempt logs. No real model was started.

Full `make validate` was excluded because it builds release archives and runs
committed binaries. ShellCheck is unavailable; native macOS/Windows behavior
and cross-compilation were not checked. An unnecessarily queued cold-cross-build
plan was stopped before it started; warm harness checks ran as light work.
No verification process remains active. Results are in the unit's requested
report and `warm-verification.log`; the worktree/cache remain for review.
