---
id: async-job-robustness-process-lifecycle
kind: story
stage: review
tags: [infra]
parent: async-job-robustness
depends_on: [async-job-robustness-job-source-of-truth]
release_binding: null
gate_origin: null
created: 2026-05-27
updated: 2026-05-27
---

# PID sidecar, Setsid, group cancel, terminal guards

## Scope

Story C+D of `async-job-robustness`. Depends on Story B because PID is
no longer on the `Job` struct and the lifecycle owns sidecars.

Three coupled concerns ship together:

1. PID sidecar at `<jobdir>/pid` — parent writes it after `cmd.Start()`,
   child clears it on exit. `job.json` becomes child-owned for
   status/result transitions after launch.
2. True process detachment — `SysProcAttr.Setsid: true` on unix builds,
   split into platform files so Windows compiles. Cancel sends SIGTERM
   to the process group (`-pid`), waits 5s, then SIGKILL.
3. Terminal-status guards — `finishAsyncJob` reload-checks before
   writing and refuses to overwrite cancelled state (in both `job.json`
   and `result.json`). `cancelJob` writes cancelled state BEFORE
   signalling so a late natural-finish can't undo cancel.

Foundation-doc roll-forward (CONTRACT.md + ARCHITECTURE.md) ships in
this story because the visible behavior change lands here.

See parent feature `async-job-robustness` Units 4, 5, and 6 for the
full design.

## Files

- `internal/jobs/store.go` — `WritePID` / `ReadPID` / `RemovePID`
  helpers; export `AtomicWriteFile`.
- `internal/jobs/launch_unix.go` (new, build tag `//go:build unix`) —
  `ApplyDetachAttrs` setting `Setsid`, `SignalProcessGroup` using
  `syscall.Kill(-pid, sig)`.
- `internal/jobs/launch_windows.go` (new, build tag
  `//go:build windows`) — stub `ApplyDetachAttrs` (no-op),
  `SignalProcessGroup` returning a clear "not implemented" error.
- `cmd/peeragent/main.go` —
  - `launchAsync` calls `jobs.ApplyDetachAttrs(cmd)` and
    `store.WritePID` after start.
  - `finishAsyncJob` reload-and-guards both file writes.
  - `cancelJob` writes cancelled state first, signals the process
    group, polls for exit, escalates to SIGKILL after 5s, clears the
    pid sidecar.
- `cmd/peeragent/main_test.go` — extend with finish-vs-cancelled
  guard test, cancel-without-pid-sidecar path.
- `cmd/peeragent/main_async_test.go` (new) — integration test that
  spawns the real binary and confirms group-kill semantics.
- `docs/CONTRACT.md` — Async State section reflects new layout
  (`prompt.txt`, `pid`); `--cancel` description mentions process-group
  kill and 5s grace.
- `docs/ARCHITECTURE.md` — async flow describes the new contract.

## Acceptance Criteria

- [x] `Store.WritePID` / `ReadPID` / `RemovePID` round-trip with a
      newline-terminated decimal integer.
- [x] On unix builds, async children run with `Setsid` (verified by
      reading `/proc/<pid>/status` — `NSpid` / `PGid` differ from parent).
- [x] `cancelJob` writes `job.json:cancelled` and `result.json:cancelled`
      atomically BEFORE signalling.
- [x] Cancel reaches both the wrapper child AND its descendants. Test
      target: a `bash -c 'trap "" TERM; sleep 100'` subtree under the
      peeragent binary; cancel must terminate the whole tree within
      `5s + 500ms` (SIGTERM grace + SIGKILL latency).
- [x] If pid sidecar is missing during cancel (race or upgrade
      collateral), terminal status still lands; signalling is skipped
      with no error.
- [x] `finishAsyncJob` does not overwrite `cancelled` status in
      `job.json`.
- [x] `finishAsyncJob` does not overwrite a `cancelled` result.json
      written by `cancelJob`.
- [x] After natural finish, pid sidecar is removed.
- [x] After cancel, pid sidecar is removed.
- [x] Windows build compiles: `GOOS=windows go build ./...` succeeds.
- [x] CONTRACT.md Async State section lists `prompt.txt` and `pid`;
      `--cancel` description tightened.
- [x] ARCHITECTURE.md async flow describes spec-via-job + Setsid +
      group-kill.
- [x] No "previously" / "in v0.2.x" / migration prose in foundation
      docs.

## Verification

- `env GOCACHE=/tmp/peeragent-go-build go test ./internal/jobs ./cmd/peeragent`
  passes.
- `env GOCACHE=/tmp/peeragent-go-build go test ./...` passes.
- `env GOCACHE=/tmp/peeragent-go-build GOOS=windows go build ./...`
  passes. Go emitted a non-fatal stat-cache warning for the read-only
  module cache path, but the command exited 0.
- Manual repro:
  ```
  ( sleep 30 | bin/peeragent --async --agent codex "long task" )
  # returns job_id within 1s
  bin/peeragent --cancel <job-id>
  # full subtree (peeragent child + codex + any codex children) gone
  # within ~5s + epsilon
  bin/peeragent --result <job-id>
  # status: cancelled
  ```

## Out of Scope

- Configurable cancel grace period (hardcoded 5s).
- Windows cancel semantics beyond "compiles."
- Claude 90s timeout.
- Bidirectional streaming / MCP / A2A transport.

## Implementation Notes

- Exported `jobs.AtomicWriteFile` and routed job, prompt, result, and pid
  sidecar writes that this story owns through atomic replacement.
- Added `pid` sidecar helpers on `jobs.Store`; the format is decimal PID plus
  newline, read with trimmed `strconv.Atoi`, and missing pid removal is
  idempotent.
- Added unix/windows launch helpers. Unix async children get
  `SysProcAttr.Setsid = true`, and cancellation signals the negative pid as
  the process group. Windows helpers compile and return a clear
  not-implemented error for process-group signalling; Windows cancellation
  behavior remains out of scope for this story.
- `launchAsync` writes the pid sidecar immediately after `cmd.Start()` and
  before releasing the child. If the sidecar write fails, it attempts
  process-group and direct child cleanup before returning the error to avoid a
  loose child.
- `cancelJob` writes the cancelled result and guarded cancelled job status
  before signalling. Missing pid sidecars are treated as successful terminal
  state writes with signalling skipped. Present pid sidecars receive SIGTERM,
  a 5-second group-exit wait, then SIGKILL and a short final wait before the
  sidecar is removed.
- `finishAsyncJob` reloads the job before writing. If the current job status
  is terminal and different from the target status, or if an existing
  terminal result differs from the result being written, finish removes the
  pid sidecar and leaves both terminal artifacts untouched.
- Rationale for the Setsid test: the integration test asserts
  `syscall.Getpgid(wrapperPID) == wrapperPID`, which directly proves the
  child is a process-group leader after Setsid. This replaces the original
  `/proc` parsing suggestion and avoids a Linux-specific file-format
  dependency while still running only on unix-like test hosts.
