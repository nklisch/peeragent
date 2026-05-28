---
id: async-job-robustness-stdin-gate
kind: story
stage: done
tags: [infra]
parent: async-job-robustness
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-27
updated: 2026-05-27
---

# stdin gate + --job-run allow-list

## Scope

Story A of `async-job-robustness`. Smallest unblocker — ships alone.

Stop `input.Parse` from blanket-draining `os.Stdin`. The current
unconditional `io.ReadAll(stdin)` blocks the parent indefinitely when
the invoking host holds stdin open (TTY, or any pipe the parent never
closes). It looks like "codex is stuck" but the parent never even
reached codex.

Also pre-stage Story B by adding `--job-run` to the no-task-text
allow-list. Story B removes the child's argv prompt source; without
this allow-list entry, the child would error `"no task text supplied"`.

See parent feature `async-job-robustness` Unit 1 for the full design.

## Files

- `internal/input/input.go` — replace unconditional `io.ReadAll` block
  with a TTY-aware gate; add `--job-run` to the no-task-text allow list.
- `internal/input/input_test.go` — extend with the three new test cases.

## Acceptance Criteria

- [x] `Parse(["--job-run","id"], nil, getwd)` returns no error and
      empty `TaskText`.
- [x] `Parse(["task"], strings.NewReader("ctx"), getwd)` still merges:
      `TaskText == "task\n\nctx"` (existing
      `TestParseCombinesInputs`-shape).
- [x] `Parse(nil, strings.NewReader("from stdin"), getwd)` still reads
      the reader (existing `TestParseStdin`-shape).
- [x] New test: `*os.File` from a non-TTY `os.Pipe()` with data is read.
- [x] New test: a TTY-shaped `*os.File` is skipped — best-effort,
      `t.Skip` if `/dev/ptmx` (or platform equivalent) is unavailable.
- [x] Repro: `( sleep 5 | bin/peeragent --async --agent codex "task" )`
      returns a `job_id` within 1s instead of hanging.
      Covered by `TestParseSkipsNonTTYStdinFileWhenTaskTextExists`,
      which verifies real `*os.File` pipe stdin is skipped when task text
      already exists.

## Out of Scope

- Job-storage schema changes (Story B).
- Sidecar files (Story C+D).
- Cancellation lifecycle (Story C+D).

## Implementation notes

- Files changed: `internal/input/input.go`, `internal/input/input_test.go`.
- Tests added: `TestParseCombinesArgsAndStdin`,
  `TestParseReadsNonTTYStdinFile`, `TestParseSkipsTTYStdinFile`,
  `TestParseSkipsNonTTYStdinFileWhenTaskTextExists`,
  `TestParseJobRunAllowsEmptyTaskText`.
- Review correction: the first implementation still read held-open
  non-TTY `*os.File` pipes when positional task text existed. The review
  pass tightened stdin reads for real files to the no-existing-task and
  no-job-control cases while preserving non-`*os.File` test-reader merge
  behavior.
- Adjacent issues parked: none.
- Verification: `env GOCACHE=/tmp/peeragent-go-build go test
  ./internal/input` passed; `env GOCACHE=/tmp/peeragent-go-build go test
  ./...` passed.

## Review (2026-05-28)

**Verdict**: Approve

**Blockers**: none
**Important**: none
**Nits**: none

**Notes**: Review found and fixed one local correctness issue before
approval: held-open non-TTY file stdin is now skipped when task text,
prompt-file, or job-control intent already exists. Verification passed
with `env GOCACHE=/tmp/peeragent-go-build go test ./internal/input` and
`env GOCACHE=/tmp/peeragent-go-build go test ./...`.
