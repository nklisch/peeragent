---
id: story-fix-test-helper-main-inheritance
kind: story
stage: done
tags: [bug, tests]
parent: epic-mcp-server-delegation
depends_on: []
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# Prevent inherited test-helper state from launching peeragent during go test

## Symptom

Running `go test -run '^$' ./...` from a process that inherited
`PEERAGENT_TEST_HELPER_MAIN=1` makes `cmd/peeragent` treat `-test.run ^$` as a
delegation task and launch Codex instead of running the Go test harness.

## Root cause

`TestMain` selects CLI-helper mode from the environment marker alone. Child
processes inherit that marker, including later `go test` processes whose test
binary arguments are unrelated to an intentional peeragent subprocess.

## Fix approach

Require both the helper marker and the absence of Go test-runner arguments
before dispatching to `main()`. Keep the predicate testable and confined to the
test harness.

## Regression test

`cmd/peeragent/main_async_test.go` verifies that helper mode is enabled for an
intentional CLI subprocess but rejected when inherited by a Go test invocation.

## Implementation notes

Added a testable `testHelperMainRequested` predicate that requires the explicit helper marker and rejects inherited `-test.*` arguments before calling the CLI `main`. This keeps intentional peeragent subprocess tests working while allowing nested `go test` verification to remain in the Go test harness.

## Verification

- `go test ./cmd/peeragent`
- `go test ./...`

Both passed as part of the MCP stdio integration verification.

## Review notes

- Effective review weight: standard; fast administrative acceptance was supplemented by the parent feature's fresh-context review because this regression sits on the subprocess protocol test path.
- Verdict: approve. The regression predicate and all three marker/test-argument branches were verified; full and race test suites passed.
