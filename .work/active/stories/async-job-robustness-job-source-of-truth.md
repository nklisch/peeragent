---
id: async-job-robustness-job-source-of-truth
kind: story
stage: implementing
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

- [ ] `Job.TaskText` and `Job.PID` removed.
- [ ] `ExecSpec` populated and round-trips through `job.json`.
- [ ] `Store.Create(cwd, spec, prompt)` writes `job.json` + `prompt.txt`
      atomically.
- [ ] `Store.WritePrompt` / `ReadPrompt` round-trip preserves raw bytes.
- [ ] `SaveGuarded` refuses to overwrite `cancelled`, `failed`, or
      `complete` with a non-matching status.
- [ ] `launchAsync` spawns child with argv exactly
      `--job-run <id> --cwd <cwd>` — no other flags.
- [ ] `runAsyncJob` loads spec+prompt from the job directory; child's
      `os.Stdin` is never read.
- [ ] Worktree-mode async job writes its `failed` result through
      `finishAsyncJob` (not bypassing the terminal-state path).
- [ ] Atomic write proof: no `.tmp` files remain after a successful
      sequence; partial writes do not leave a half-written `job.json`.
- [ ] Async run with original prompt-via-stdin now works end-to-end:
      `echo "task" | bin/peeragent --async --agent codex` runs codex
      against the piped prompt.

## Out of Scope

- PID sidecar (Story C+D, which removes the `Job.PID` field with this
  story; the sidecar lands in C+D).
- Setsid / process-group cancel (Story C+D).
- Foundation-doc roll-forward (Story C+D — lands with the visible
  behavior change in cancel + child detachment).
