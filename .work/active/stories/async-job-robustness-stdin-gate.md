---
id: async-job-robustness-stdin-gate
kind: story
stage: implementing
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

- [ ] `Parse(["--job-run","id"], nil, getwd)` returns no error and
      empty `TaskText`.
- [ ] `Parse(["task"], strings.NewReader("ctx"), getwd)` still merges:
      `TaskText == "task\n\nctx"` (existing
      `TestParseCombinesInputs`-shape).
- [ ] `Parse(nil, strings.NewReader("from stdin"), getwd)` still reads
      the reader (existing `TestParseStdin`-shape).
- [ ] New test: `*os.File` from a non-TTY `os.Pipe()` with data is read.
- [ ] New test: a TTY-shaped `*os.File` is skipped — best-effort,
      `t.Skip` if `/dev/ptmx` (or platform equivalent) is unavailable.
- [ ] Repro: `( sleep 5 | bin/peeragent --async --agent codex "task" )`
      returns a `job_id` within 1s instead of hanging.

## Out of Scope

- Job-storage schema changes (Story B).
- Sidecar files (Story C+D).
- Cancellation lifecycle (Story C+D).
