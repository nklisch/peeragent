---
id: source-shim-exit-codes
kind: story
stage: done
tags: [infra, tests]
parent: session-control-reliability
depends_on: []
release_binding: 0.9.0
created: 2026-10-19
updated: 2026-10-21
---

# Preserve wrapper exit codes in the source-checkout shim

The session-control fixes make the compiled CLI return usage exit 2. The
source-checkout shim still executes `go run`, which converts that nonzero exit
to 1. Reproduced with the branch's `bin/peeragent --list`: JSON correctly
reported `metadata.exit_code: 2`, but the process exited 1 and stderr printed
`exit status 2`. No agent was started. Missing-job exit 4 has the same issue.

The source entry point must preserve the compiled program's exit codes, while
still using current source and supporting detached jobs. Keep the existing
explicit-binary and developer-build precedence. Cover exit 2 and 4 with a fake
Go compiler and CLI helper; check temporary-binary cleanup and an actual
source-shim async job with the fake Codex agent.

## Implementation and focused verification

The source fallback builds a private temporary executable, from the source
module directory, then returns its exact exit status and cleans up. Explicit
binary and developer-build precedence stay intact. The packaged shim matches.

- Regression test failed on the original source shim with OS exit 1 and JSON
  exit 2, then passed after the fix; it also covers missing-job exit 4 and
  temporary-directory cleanup, using a fake Go compiler and real CLI helper.
- `sh -n bin/peeragent` and `sh -n plugin/bin/peeragent` passed. ShellCheck is
  not installed on the verification host.
- Actual source-shim fake-agent smoke passed from throwaway directories:
  same-job capacity retry, cross-directory controls/resume, explicit directory
  precedence, strict options, and retained attempt logs. No model was started.
- The combined final `make test` suite, `go vet`, formatting and static
  metadata/documentation checks passed on `0198014`. Native macOS/Windows
  behavior and cross-builds were not checked. The optional cold cross-build
  queue entry was stopped before execution; no verification process remains.

## Shipped closure

Closed after checking Git history: `1f40116` (terminal capacity outcomes) and
`0198014` (source-shim exits), with the parent implementation `0b88729`, are
ancestors of the `v0.9.0` tag. The item was left in review after shipping. No
release operation was performed during this bookkeeping correction.
