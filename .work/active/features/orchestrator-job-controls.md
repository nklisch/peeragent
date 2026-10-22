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
  phase start a five-second grace period. A provisional `turn.completed` is
  revoked by continuing agent/tool activity; later completion restarts grace. Status reports “final message received,
  process still alive”. On expiry, the runner ends only its owned child/group
  (Windows Job Object), reaps it and uses the terminal-event outcome. Ordinary
  unphased messages can be progress and never initiate shutdown. Natural target
  exit codes retain existing semantics.
- Help, README, foundation docs and both skill copies put opt-in
  `--retry-capacity N` next to async guidance. Retry defaults are unchanged.

## Review and decisions

A separate harness reviewer inspected implementation, lifecycle contracts,
process ownership and compatibility without rebuilding. Accepted and fixed the
inherited-output-pipe finding during the initial implementation. The subsequent
owner-accepted review found the unconditional pipe deadline could interrupt an
active worker, and group signals could outlive the leader's ownership. Both are
fixed in the correction round below.

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
- JSONL records without a trailing newline are processed at stdout EOF.
  Intermediate unphased messages cannot reliably distinguish final output.
- Multi-status fails before printing records when any job cannot be loaded;
  single-job missing-id exit 4 remains unchanged. No tracker reorganization beyond
  convention-required closure moves for the fixed/shipped items.

## Initial implementation verification log

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


## Accepted review corrections (2026-10-22)

- **Keep ownership through the last Unix group signal.** The runner observes
  leader exit without reaping (`waitid`/`WNOWAIT` on Linux, `EVFILT_PROC`/
  `NOTE_EXIT` on macOS). A shared ownership lock revokes every group signal
  before the explicit reap can release the group number. Windows retains its
  Job Object and process handle. `TestOwnedGroupNeverSignalsReusedNumber`
  simulates reuse after leader exit/reap, including cleanup during reap; no real
  unrelated process is touched. `TestExitObservationLeavesLeaderWaitable`
  checks that actual exit observation leaves the child's status collectible.
- **Keep ordinary pipe draining unbounded.** Removed the unconditional
  `Cmd.WaitDelay`. Stdout/stderr draining and exit observation are independent;
  forced reader closure occurs only for terminal grace, cancellation or a
  concrete I/O/observation failure requiring cleanup. The inherited-pipe test
  still covers terminal cleanup, while
  `TestLauncherExitDoesNotTimeOutActiveWorkerPipes` lets a finite worker outlive
  its launcher and the former deadline, then checks its entire output and
  successful outcome.
- **Allow Codex continuation after turn completion.** Completion observations
  can start, cancel or restart grace; a generation check rejects stale timer
  wake-ups. `turn.completed` is provisional and following turn/item/agent/tool
  events cancel it, including persisted `final_received`. `task_complete` and
  explicitly final messages remain session completion. Parser tests cover each
  activity kind and prevent later turn markers from downgrading explicit
  completion. `TestContinuedCodexTurnGetsFreshCompletionGrace` runs a fake
  continued turn past the original grace, then emits `task_complete` and hangs
  until the fresh watchdog cleans it up.

Platform references: [Apple kqueue documentation](https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man2/kqueue.2.html)
defines process exit notification separately from reap. [Go's exit-wait implementation](https://go.dev/src/os/wait_waitid.go)
uses Linux `WNOWAIT` and records the stopped-process problem with Darwin
`waitid`, which is why this implementation uses kqueue there.

### Correction-round verification

Tested code SHA: `12524e570bb7c72567f6a294fe61897656965f1a`.
Toolchain: `go version go1.26.4 linux/amd64`. The final evidence commit changes
only this record; no Go source changed after the frozen run.

- Focused `go test -json ./internal/executil ./internal/codex -run
  'TestOwnedGroup|TestLauncherExit|TestOSRunner|TestLineObserver|TestTurnCompletion|TestTerminalStream|TestProgressMessage'
  -count=1`: passed, measured 11 top-level / 17 total test passes.
- After adding the real unreaped-exit check and explicit-completion precedence,
  focused `-run 'TestExitObservation|TestTurnCompletion|TestExplicitSession'`
  passed: measured 3 top-level / 9 total test passes. The activity parser's
  added event prefixes and observer-failure preservation were then included in
  the frozen full suite.
- Focused CLI checks for existing live output, task completion, failures and
  capacity retries passed. The first new continuation test mistakenly requested
  message text from `--status`; changed that observation to `--result`, without
  changing its expected continuation or final-message assertions. The focused
  `go test -json ./cmd/peeragent -run TestContinuedCodexTurn -count=1` then
  passed. The corrected test is also in the frozen full-suite count.
- `GOFLAGS='-v -p=6' ./scripts/validate.sh`: exit 0, offline, in a proper
  detached disposable worktree at the tested SHA. Its initial `go test ./...`
  was the single full-suite execution on the frozen code. Measured from verbose
  pass records: **217 top-level tests, 319 including subtests, 12 tested
  packages, zero failed or skipped tests**; testsupport has no tests.
  `review-frozen-full-test.log` extracts that suite's output from the validator
  log; it is not a repeated run. Packaging, strict plugin validation, Linux and
  macOS amd64/arm64 archive builds, metadata/docs and shim checks all passed.
  The existing missing-plugin-icon warning remains nonblocking.
- For `windows/amd64`, `darwin/amd64` and `darwin/arm64`,
  `GOOS=<os> GOARCH=<arch> CGO_ENABLED=0 go vet ./...` and
  `go test -c -o <scratch>/review-<os>-<arch>-tests/ ./...` passed (exit 0
  each). These are static/compile checks, not native execution. Native macOS
  exit notifications and Windows Job Object behavior remain unverified here;
  the existing Windows assignment race remains the documented limit above.
- Direct `go build ./...`, `go vet ./...` and `git diff --check` passed
  before each correction-round commit. All Go work was light work, without
  `heavy` or a PATH wrapper. No performance measurements were made.

Logs retained for owner review under
`/storage/scratch/peeragent/job-controls-2/`: `review-frozen-environment.log`,
`review-frozen-full-test.log`, `review-frozen-validate.log`, the corresponding
`.exit` files (all frozen-run exits 0), `review-focused-*.jsonl`, and platform
`review-<os>-<arch>-{vet,compile}.{log,exit}` files. The initial `review-focused.exit` is 1 for the test observation mistake
described above; corrected `review-focused-round2.exit` is 0. These supersede the earlier
code-SHA qualification for this correction round. The temporary worktree,
platform test binaries, scripts, process-id markers and probe state were removed.
The unit's existing cache and review logs remain owned by this unit and are to
be removed on acceptance/landing or abandonment, as recorded above.

No release, tag, push, committed binary refresh, installed-plugin change or
other session's job-folder access. No new CLI flag or exit-code changes in this
round. Follow-ups remain native platform execution, the existing Windows
startup assignment race, and completion visible only in private session logs.
