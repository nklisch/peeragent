---
id: orchestrator-job-controls
kind: feature
stage: review
tags: [infra, tests, docs]
parent: null
depends_on: []
release_binding: null
created: 2026-10-21
updated: 2026-10-22
---

# Job controls for orchestrators running many peers

Requested by Nathan through godot-41. Ready for owner review, without release,
installed-plugin changes, committed binary refresh, tag or push.

## Implementation

- `--wait-any <job-id>...` resolves and validates the entire set before waiting.
  Unknown ids are usage exit 2; already terminal jobs return immediately.
  Ties use argument order. Terminal outcomes retain the existing `--wait`
  wrapper exit 0, with target status/exit metadata preserved.
- Multi-id `--status` emits one compact JSON object per line or a `--text`
  table: status, model, effort, session, launch time, capacity-backoff flag,
  current HEAD when git is available, and pending-completion flag. Single-id
  status keeps its established result shape. Legacy ignored task text remains
  accepted after single-id status.
- Both controls support `job-id@directory`; that directory wins over shared
  `--cwd`, then lookup falls back to the index and current directory. Explicit
  paths support spaces and resolve relative to the caller. Status reads each
  job once and HEAD once per distinct worktree. Waiters cache resolved locations
  and avoid reading potentially large running messages on each poll.
- Codex session ids are persisted as soon as stdout reports them. Completed
  messages are stored without truncation in `message.txt`; running and terminal
  results include them or expose `metadata.message_path`, including after
  cancellation, failure and capacity retry. Per-attempt raw logs remain intact.
- Explicit Codex completion/failure events and messages marked with a final
  phase start a five-second grace period. Status reports “final message received,
  process still alive”. On expiry, the runner ends only its owned child/group
  (Windows Job Object), reaps it and uses the terminal-event outcome. Ordinary
  unphased messages can be progress and never initiate shutdown. Natural target
  exit codes retain existing semantics.
- Help, README, foundation docs and both skill copies put opt-in
  `--retry-capacity N` next to async guidance. Retry defaults are unchanged.

## Review and decisions

A separate harness reviewer inspected implementation, lifecycle contracts,
process ownership and compatibility without rebuilding. Accepted and fixed the
inherited-output-pipe finding: `Cmd.WaitDelay` could beat the completion timer,
return a false failure and leave a descendant alive. The wait-error path now
ends the owned tree and preserves terminal-event completion for that case. A
fake descendant deliberately retains output after the direct child exits.

Self-review removed a second parse of Codex stdout after streaming and avoided
reading completed-message files during each running wait poll. Message pointers
are visible in both JSON and text results. No inbox or messaging protocol was
added: a cheap honest alternative is to finish/cancel and resume the saved
session with the next instruction.

## Known limits and follow-ups

- Native macOS/Windows runtime checks and real target-model calls are not
  available in this Linux unit. Platform build/static qualification is recorded
  below. Existing Windows `--cancel` process-group signaling remains unsupported;
  this unit's completion watchdog owns a Windows Job Object separately.
- Windows assigns its launched process to a Job Object immediately after Start.
  A shim could theoretically spawn a descendant before assignment, allowing it
  to escape later watchdog cleanup. No occurrence or rate was measured. Defer a
  suspended-launch improvement until native Windows evidence warrants it,
  following the rare-condition policy; do not claim that startup race is closed.
- This fix consumes stdout events. The historical incidents showed completion
  in Codex's private session log; their stdout was not captured. A target that
  emits no terminal stdout event and no explicitly final message still requires
  ordinary exit or cancellation. No private-session-log tailer was added.
- JSONL records without a trailing newline are processed when the process exits.
  Intermediate unphased messages cannot reliably distinguish final output.
- Multi-status fails before printing records when any job cannot be loaded;
  single-job missing-id exit 4 remains unchanged. No tracker reorganization beyond
  convention-required closure moves for the fixed/shipped items.

## Verification log

Focused fake-agent checks passed during development for multi-job lookup,
selection and prevalidation; early sessions; cancellation/failure; long messages;
capacity retry; explicit completion with a sleeping target; inherited pipes;
observer failure; and ownership isolation from an unrelated process. Build and
vet passed before the first implementation commit `ce6b3a5`.

Frozen code tested: `ac7e5f826f807aedddca64db5a7e6f12bae2b93a`, using
`go version go1.26.4 linux/amd64`. All commands below passed offline. Counts
are measured from the JSON test events, not estimates:

- `go test -json ./...`: exit 0; 209 top-level test passes, 305 including
  subtests, 12 tested packages, zero failed or skipped tests. The testsupport
  package has no tests. Reused this earlier full-suite pass after the interrupted
  verification because the code SHA was unchanged, as godot-41 directed.
- `./scripts/validate.sh`: exit 0 in a proper detached disposable Git worktree
  at the same SHA. Its required suite replay, build, plugin package, strict
  Claude plugin validation, committed-binary/shim smoke, metadata/docs checks
  and archive builds all passed. Archive targets: Linux amd64/arm64 and macOS
  amd64/arm64. No publish flag, tag, push, installed-plugin write or committed
  binary refresh. Existing missing-plugin-icon warning is nonblocking.
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./...` and
  `GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go vet ./...`: exit 0 each.
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -o <scratch>/windows-tests/ ./...`
  and the equivalent Darwin command: exit 0 each. These compile tests without
  executing them; no native Windows or macOS runtime coverage is claimed.
- `go build ./...`, `go vet ./...`, and `git diff --check`: passed before the
  final evidence-only commit, as before each implementation commit.

Feature-to-regression mapping (all included in the full-suite pass):

| Requested outcome | Regression evidence |
| --- | --- |
| Wait for any job, resolve across worktrees, reject unknown ids first | `TestWaitAnyReturnsFirstTerminalAcrossDirectories`, `TestWaitAnyWaitsForLaterCompletion`, `TestWaitAnyValidatesAllIDsBeforeReturningTerminal`, `TestMultipleJobReferences` |
| Compact multi-status and compatible single-status | `TestMultipleStatusJSONAndText`, existing `TestParseStatus`, cancelled-backoff regression in `session_fixes_test.go` |
| Live session ids survive cancellation | `TestLiveSessionAndUntruncatedMessageSurviveCancellation`, `TestProgressPreservesTerminalWinner` |
| Preserve full reports on running, cancelled, failed and retried jobs | The live-session test, `TestFailedCodexRetainsFinalMessage`, `TestCapacityRetryPreservesMessagePointer`, `TestLineObserverPreservesLargeFragmentedLines`; fake report is 240000 bytes (measured UTF-8 length) |
| Complete a target that prints its terminal event then sleeps | `TestTaskCompleteStopsOwnedHangingCodex`, `TestTerminalStreamEnvelopes`, `TestProgressMessageDoesNotEndTurn` |
| End only owned processes, including inherited output pipes | `TestOSRunnerCompletesWithInheritedOutputPipes`, `TestCompletionDoesNotStopUnrelatedProcess`, `TestOSRunnerStopsOwnedProcessOnObserverFailure` |
| Avoid repeated large-message reads while waiting | `TestWaitPollDoesNotReadRunningMessage` |
| Help/README/both skill copies expose opt-in capacity recovery | Validation documentation and skill checks; existing async capacity/default-no-retry regression tests |

Verification logs retained while this feature awaits godot-41's review:
`/storage/scratch/peeragent/job-controls-2/full-test.jsonl`,
`full-test.exit`, `environment.log`, `direct-environment.log`,
`validate-direct.log`, `validate-direct.exit`, and the Windows/Darwin
`*-vet.log`, `*-compile.log` and matching exit files. The final aggregate
`verify-direct.exit` is 0. The owner can remove these on acceptance; the tracked
record above is the permanent verification log. The unit's Go cache remains at
`/storage/build/peeragent/job-controls-2-go-cache` for any correction round and
must be removed when the unit lands or is abandoned.

The previous queued validation was stopped by godot-41; that interruption was
not a product failure. Removed the task-local Go queue wrapper and reran the
validator directly, with no `heavy` and no PATH override. The earlier archive
copy lacked Git metadata; it was deleted and replaced by the disposable Git
worktree. Disposable checkout, test binaries and probes were removed after
verification. No performance or runtime timing measurements were made.
