---
id: source-shim-exit-codes
kind: story
tags: [infra, tests]
parent: session-control-reliability
depends_on: []
created: 2026-10-19
updated: 2026-10-19
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
