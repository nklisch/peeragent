---
id: async-job-robustness
kind: feature
stage: drafting
tags: [infra]
parent: null
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-27
updated: 2026-05-27
---

# Async Job Robustness

## Brief

The first async-jobs implementation (epic-async-jobs, done) ships with five
latent bugs uncovered by peer-reviewed investigation: a blanket stdin drain
that hangs the parent when the invoking host holds stdin open, a child
process that reparses argv and ignores the persisted prompt source, an
initial PID-save race against fast-failing children, a wrapper that shares
the parent's session and has no process-group cancel, and a finish path
that can overwrite cancelled status.

This feature tightens the async path end-to-end so it survives realistic
invocation contexts (held-open stdin, host disconnects, cancellation
mid-run) and so the on-disk job becomes the single source of truth for
what the child should execute. Scope is remediation only — no new
capabilities, no new flags.

## Foundation References

- `docs/CONTRACT.md` — async state layout under `.peeragent/jobs/<id>/`
  and `--cancel` semantics; rolls forward when sidecar files land.
- `docs/ARCHITECTURE.md` — async flow and process-group handling; rolls
  forward when Setsid + group-kill ship.

## Strategic Decisions

- **Feature shape**: One feature with three child stories (stdin gate;
  job-as-source-of-truth; pid-sidecar + Setsid + group-kill +
  terminal-status guards). Coheres as one remediation arc.
- **Claude 90s timeout (surfaced during peer review)**: Skip entirely
  for now — not tracked. User will revisit if it bites again.
- **Job storage migration**: Clean break. Bump `job.json` schema: drop
  `TaskText` from the embedded record, persist prompt to a sidecar
  `prompt.txt`, move PID to a sidecar `pid` file, add a `spec` block
  carrying agent/profile/effort/model/resume/full-access/worktree/json.
  Pre-1.0 (v0.2.x); in-flight jobs at upgrade time are acceptable
  collateral.

## Peer-Reviewed Fix Surface

Three workstreams, ordered for ship sequence. Stories spawn during
feature-design.

### Story A — stop blanket-draining stdin (smallest unblocker)

- `internal/input/input.go`: only read `os.Stdin` when no positional
  task text, no `--prompt-file`, and no job-control flag was supplied;
  and only when `stdin` is an `*os.File` whose `Stat()` mode lacks
  `os.ModeCharDevice` (skip TTYs). Preserve `io.ReadAll` for non-`*os.File`
  readers passed by tests.
- Add `jobRunID` to the no-task-text allow list alongside
  `statusJobID`, `resultJobID`, `cancelJobID`. Required because Story B
  removes the child's argv prompt source.
- Preserve merge semantics for non-TTY stdin alongside positional text
  (existing `echo context | peeragent "task"` pattern).

### Story B — job is the source of truth for the async child

- `internal/jobs/store.go`: drop `TaskText` from `Job`; add an
  `ExecSpec` (agent, profile, effort, model, resume, full-access,
  worktree, json).
- `launchAsync` writes the resolved prompt to `<jobdir>/prompt.txt` and
  the resolved spec into `job.json` once, then spawns the child as
  `peeragent --job-run <id> --cwd <cwd>` and nothing else.
- `runAsyncJob` loads spec from `job.json` and prompt from `prompt.txt`;
  no `input.Parse` reparse of execution flags in the child.

### Story C+D — pid sidecar, detachment, group cancel, terminal guards

- `<jobdir>/pid` sidecar written by parent after `cmd.Start()`; `job.json`
  becomes child-owned for status/result transitions, so the parent never
  writes to it after initial create.
- `launchAsync` sets `SysProcAttr.Setsid: true` on unix; split into
  `launch_unix.go` / `launch_windows.go` with a stub for the latter so
  cross-builds still compile.
- `cancelJob` reads the pid sidecar, sends `SIGTERM` to `-pid` (process
  group), waits a short grace period, then `SIGKILL` to `-pid`.
- `finishAsyncJob` reloads job state before writing and refuses to
  overwrite terminal states (`cancelled`, `failed`, `complete`) — guard
  applies to both `job.json` AND `result.json` writes, since the child
  writes `result.json` first.

## Foundation-Doc Roll-Forward (during implementation)

- `docs/CONTRACT.md`: update the `Async State` section to list
  `prompt.txt` and `pid` in `<jobdir>/`; tighten `--cancel` description
  to mention process-group kill.
- `docs/ARCHITECTURE.md`: update the async flow to describe the child
  reading spec + prompt from the job dir, and Setsid-based detachment.

Per the rolling-foundation principle, edit in place — no "previously"
notes.

## Tests (minimum lock-in set)

- `input.Parse(["--job-run","id"], nil, ...)` succeeds with empty task.
- `input.Parse(["task"], strings.NewReader("ctx"), ...)` still merges
  non-TTY stdin into task text.
- TTY-shaped `*os.File` stdin is skipped; pipe/file stdin is read.
- `launchAsync` writes `job.json` (with `ExecSpec`), `prompt.txt`, and
  `pid`; child argv is exactly `--job-run <id> --cwd <cwd>`.
- `runAsyncJob` loads prompt/spec from the job dir, not from argv parse.
- `finishAsyncJob` does not overwrite `cancelled` status or
  `result.json`.
- `cancelJob` reads pid sidecar and targets the process group on Unix
  (use a sleep-loop child as the test target to confirm group reach).

## Verification

`go test ./...` plus a reproduction script that exercises `--async`
with stdin held open (`sleep N | peeragent --async`) to confirm the
parent returns `job_id` promptly and the child completes.

## Out of Scope

- Claude 90s timeout surfaced during peer review (skipped per strategic
  decision).
- Worktree mode (`--worktree`) implementation — still returns
  not-implemented error.
- Bidirectional streaming / MCP / A2A transport — discussed during peer
  review, ruled overkill for current scope.

<!-- Stories spawn during /agile-workflow:feature-design. -->
