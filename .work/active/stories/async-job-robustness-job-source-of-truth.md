---
id: async-job-robustness-job-source-of-truth
kind: story
stage: review
tags: [infra]
parent: async-job-robustness
depends_on: [async-job-robustness-stdin-gate]
release_binding: null
gate_origin: null
created: 2026-05-27
updated: 2026-05-27
---

# Job is the source of truth for the async child

## Scope

Story B of `async-job-robustness`. Depends on Story A so the
`--job-run` no-task-text allow-list is in place before the child is
spawned without prompt argv.

Clean-break the `job.json` schema. Drop `TaskText` from the `Job`
struct, drop `PID` (will move to a sidecar in Story C+D), add a typed
`ExecSpec`. Persist the resolved prompt to `<jobdir>/prompt.txt`.
Rewrite `launchAsync` to write spec+prompt and spawn the child as
`peeragent --job-run <id> --cwd <cwd>` only. Rewrite `runAsyncJob` to
load spec+prompt from the job directory instead of reparsing argv.

See parent feature `async-job-robustness` Units 2 and 3 for the full
design.

## Files

- `internal/jobs/store.go` — `Job` struct change, `ExecSpec` type,
  `Create(cwd, spec, prompt)` signature change, prompt sidecar
  helpers, atomic write, `SaveGuarded`.
- `internal/jobs/store_test.go` — extend with create-with-spec
  round-trip, prompt sidecar round-trip, `SaveGuarded` cancellation
  refusal, atomic-write proof.
- `cmd/peeragent/main.go` — `launchAsync` and `runAsyncJob` rewrites
  per Unit 3; add `requestFromJob` helper.
- `cmd/peeragent/main_test.go` — extend with `requestFromJob`
  reconstruction test.

## Acceptance Criteria

- [x] `Job.TaskText` and `Job.PID` removed.
- [x] `ExecSpec` populated and round-trips through `job.json`.
- [x] `Store.Create(cwd, spec, prompt)` writes `job.json` + `prompt.txt`
      atomically.
- [x] `Store.WritePrompt` / `ReadPrompt` round-trip preserves raw bytes.
- [x] `SaveGuarded` refuses to overwrite `cancelled`, `failed`, or
      `complete` with a non-matching status.
- [x] `launchAsync` spawns child with argv exactly
      `--job-run <id> --cwd <cwd>` — no other flags.
- [x] `runAsyncJob` loads spec+prompt from the job directory; child's
      `os.Stdin` is never read.
- [x] Worktree-mode async job writes its `failed` result through
      `finishAsyncJob` (not bypassing the terminal-state path).
- [x] Atomic write proof: no `.tmp` files remain after a successful
      sequence; partial writes do not leave a half-written `job.json`.
- [x] Async run with original prompt-via-stdin now works end-to-end:
      `echo "task" | bin/peeragent --async --agent codex` runs codex
      against the piped prompt.

## Out of Scope

- PID sidecar (Story C+D, which removes the `Job.PID` field with this
  story; the sidecar lands in C+D).
- Setsid / process-group cancel (Story C+D).
- Foundation-doc roll-forward (Story C+D — lands with the visible
  behavior change in cancel + child detachment).

## Implementation Notes

- Replaced the async job schema with `Job{ID, Status, CWD, Spec, CreatedAt,
  UpdatedAt, LogPath, ResultPath, PromptPath}` and a typed `ExecSpec`. Prompt
  text is persisted only in `<jobdir>/prompt.txt`; `job.json` no longer carries
  prompt text or PID.
- `Store.Create(cwd, spec, prompt)` creates the job directory, atomically writes
  `job.json`, then atomically writes `prompt.txt` using `<path>.tmp` plus
  `os.Rename`. `Save` and `WritePrompt` share the same atomic writer.
- Added `SaveGuarded`, which reloads the prior job and returns the prior status
  without overwriting when an existing terminal `cancelled`, `failed`, or
  `complete` status would be replaced by a different status.
- `launchAsync` now persists `ExecSpec` and the resolved prompt, then spawns the
  child with exactly `--job-run <id> --cwd <cwd>`. The old argv carry-forward
  path was removed.
- `runAsyncJob` loads the job and prompt sidecar, reconstructs
  `input.Request` with `requestFromJob`, and routes worktree-mode failure
  through `finishAsyncJob`.
- Cancellation no longer attempts PID signalling in this story. It still writes
  a cancelled result and guarded cancelled status; PID sidecar and signalling
  remain in Story C+D.

## Verification

- `gofmt -w internal/jobs/store.go internal/jobs/store_test.go cmd/peeragent/main.go cmd/peeragent/main_test.go`
- `env GOCACHE=/tmp/peeragent-go-build go test ./internal/jobs ./cmd/peeragent` — passed.
- `env GOCACHE=/tmp/peeragent-go-build go test ./...` — passed.

Prompt-via-stdin end-to-end behavior is covered through the unit-test contract
rather than by spawning a real target CLI: input parsing already resolves piped
stdin into `req.TaskText`, `launchAsync` writes that prompt into `prompt.txt`,
the child argv helper permits only `--job-run <id> --cwd <cwd>`, and
`runAsyncJob` reconstructs the target request from persisted spec plus prompt.
No real Codex/Claude/Gemini CLI was invoked during verification.
