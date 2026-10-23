---
id: orchestrator-job-controls
kind: feature
stage: review
tags: [infra, tests, docs]
parent: null
depends_on: []
release_binding: null
created: 2026-10-21
updated: 2026-10-23
---

# Job controls for orchestrators running many peers

Requested by Nathan through godot-41. Ready for owner review. Round 5 prepares
v0.11.3 metadata; installed-plugin changes, committed binary refresh, tag and
push remain with the publishing owner.

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


## Blocked stdin cancellation correction (2026-10-22)

Owner accepted the follow-up review's stdin blocker. A prompt exceeding pipe
capacity, an exited launcher and an output-closed worker holding stdin could
send the runner into `Cmd.Wait` while Go's hidden input copier was blocked.
That made later cancellation unable to reach owned-tree cleanup.

The observed runner now uses `StdinPipe` and owns prompt copying itself.
Input completion stays in the select loop, alongside exit and output draining;
all copies finish before the synchronous reap. Forced cleanup explicitly closes
the stdin writer on every platform. No background `Cmd.Wait` was introduced on
Linux/macOS, and the unreaped leader continues to reserve its group number.
Early target stdin closure preserves Go's broken-pipe handling on Unix,
Windows and Plan 9; unexpected copy errors remain visible.

`TestCancelBlockedStdinAfterLauncherExit` creates the exact Linux descriptor
state, checks actual pipe capacity against the prompt length, then pauses the
runner after it consumed both output EOFs and launcher exit. A small unexported
runner test seam supplies that deterministic decision boundary. It cancels
there, then requires cancellation to return, the leader to be reaped and the
worker to be stopped. Fixture cleanup has a scoped release file if assertions
fail, so it never needs to guess process ids or kill an unrelated group.
`TestOSRunnerNativeStdin` now covers observed and ordinary delivery, while
`TestOSRunnerObservedEarlyStdinClose` pins successful early stdin closure.

The initial unstaged regression passed against old code because cancellation
could arrive before the bad wait. The final staged test passed with the fix and failed with the previous
implementation (only its readiness seam was backported). The expected old-code
failure is “blocked stdin prevented cancellation from returning”; fixture cleanup
then releases its own worker. No expected outcome was weakened.

Focused checks passed: measured 9 top-level / 11 total test passes, covering
blocked-input cancellation, full prompt delivery, early stdin closure, group
ownership, waitable exit, observer failure and inherited output pipes. A focused
`go test -race` on the three stdin cases also passed: measured 3 top-level / 5
total test passes. The fake measured an 8388608-byte prompt and 65536-byte pipe
capacity. These are byte counts, not performance estimates. Native build/vet
and diff checks passed before the fix commit.

### Blocked-input frozen-tree qualification

Tested code: `9ac7614220d2585c01006cc719f10cb6b44f3111`, using
`go version go1.26.4 linux/amd64`. The following evidence commit only updates
this record, with no Go changes after the frozen run.

- `GOFLAGS='-v -p=6' ./scripts/validate.sh`: passed offline in a proper detached
  disposable worktree. Its initial direct `go test ./...` was the **single full
  suite run on frozen code**. Measured from verbose test events: **222 top-level
  passes / 326 including subtests, 12 tested packages, zero failed or skipped
  tests**. The testsupport package has no tests. The extracted full-suite log
  is a copy of that validator step, not a second execution.
- Validator packaging, plugin checks, Linux/macOS amd64/arm64 archive builds,
  documentation/metadata and shim smoke checks passed. Existing missing-icon
  warning remains nonblocking. No publish flag or installed-plugin write.
- `GOOS=<os> GOARCH=<arch> CGO_ENABLED=0 go vet ./...` and
  `go test -c -o <scratch>/stdin-<os>-<arch>-tests/ ./...` passed for
  Windows amd64 and macOS amd64/arm64. No native execution is claimed there;
  the descriptor-state regression ran on Linux. The portable prompt-delivery
  and early-stdin-close cases compile on those platforms.
- Focused command:
  `go test -json ./internal/executil -run 'TestCancelBlockedStdin|TestOSRunnerNativeStdin|TestOSRunnerObservedEarlyStdinClose|TestOwnedGroup|TestExitObservation|TestOSRunnerStops|TestOSRunnerCompletes|TestLauncherExit' -count=1`.
- Race command:
  `go test -race -json ./internal/executil -run 'TestCancelBlockedStdin|TestOSRunnerNativeStdin|TestOSRunnerObservedEarlyStdinClose' -count=1`.
- `go build ./...`, `go vet ./...`, and `git diff --check` passed before
  each commit. All Go steps were direct light work, without `heavy` or a PATH
  wrapper. No timing or performance benchmark was run.

Raw evidence retained for owner review under
`/storage/scratch/peeragent/job-controls-2/`: `stdin-focused.jsonl` / `.exit`
(0), `stdin-race.jsonl` / `.exit` (0), `stdin-counterfactual.jsonl` / `.exit`
(1, the expected pre-fix cancellation failure), `stdin-frozen-environment.log`,
`stdin-frozen-full-test.log`, `stdin-frozen-validate.log` / `.exit` (0),
`stdin-frozen.exit` (0), and `stdin-<os>-<arch>-{vet,compile}.{log,exit}`
(all exit 0). The temporary old-code reproduction and validation worktrees,
compiled platform test binaries, scripts, PID markers and probe state were
removed. The unit's cache and raw logs remain for this open owner review, with
cleanup on acceptance/landing or abandonment as recorded above.

No new flags, output-shape or exit-code changes. No remaining blocker is known.
Native macOS/Windows runtime qualification, the existing Windows assignment
race and private-session-log-only completion remain the earlier follow-ups.


## Round 3: capacity diagnostics, live progress and safe resumes

Implementation requested by godot-41 from the October 22 friction sweep;
ready for owner review, with opt-in retry defaults unchanged.

- The direct-capacity incident's `error` followed by `turn.failed` is already
  classified by base `ae0c15a`; its job now records `capacity_exhausted`.
  Added real-shape parser and fake-CLI regression coverage rather than claiming
  a new parser fix for a failure this checkout cannot reproduce. The second
  incident's remote-compaction prefix prevented the diagnostic-prefix match;
  unwrap that known prefix before classifying, preserving terminal precedence.
  Both shapes now have stop/result/status and opt-in retry coverage. Summaries
  say the model is at capacity and include the resume command ahead of the
  prior agent message.
- Chose live compact JSONL in `agent.log`, preserving full messages separately
  and raw per-attempt logs after exit. Status reads only its last 8 KiB, including
  a timestamp and single-line message preview retained across tool events and
  retries. No full-log/report read per status or wait poll.
- Every CLI launch now uses the tracked worker. Blocking calls wait for its
  result with their existing success/failure exits; async calls return immediately.
  JSON and text carry the job id. Blocking text also prints it to stderr before
  waiting, preserving the existing stdout result prefix.
- A short per-agent/session launch lock serializes the check/start window.
  The check scans the requested worktree and worktrees known to the index,
  ignoring terminal jobs and recorded processes that have exited. A conflict
  exits 2 before launch and names the job, wait/cancel commands and
  `--allow-concurrent-session` override. The threat is interleaved/lost work in
  one session; the refusal prevents the second writer. Clearing the index loses
  knowledge of other worktrees; use the original worktree with `--cwd` then.
  The override also bypasses a stale launch lock. No process is signalled by
  the check. Tests cover concurrent launches, agent namespaces, cross-worktree
  conflicts, terminal/gone/missing PIDs, override and CLI exit 2.
- Retry errors and help now explicitly say to add `--async`, including with
  `--resume`, and require Codex. Defaults, budgets and backoff remain unchanged.

No installed plugin, committed binary, release/tag, main checkout or other
session's job folder was modified. Native Windows/macOS execution and real
model calls remain outside this Linux unit; final qualification follows below.


### Round 3 verification and handback

Frozen code tested: `41b973438a161beb700d93b8246bdbf7da69ba19`, with
`go version go1.26.4 linux/amd64`. The final evidence commit changes this
record only; the tested source remains unchanged. All commands passed:

- `go test -json ./...`: **235 top-level test passes / 355 including subtests**,
  12 tested packages, zero failures/skips; testsupport has no tests.
- `./scripts/validate.sh`: passed offline in a disposable Git worktree at the
  same SHA. Includes its own required full-suite step, source build, curated
  package validation, committed-binary/shim smoke, metadata/docs checks and
  Linux/macOS amd64/arm64 archives. Existing missing-icon warning is nonblocking.
  No publish flag, installed-plugin write or committed binary refresh.
- `GOOS=<os> GOARCH=<arch> CGO_ENABLED=0 go vet ./...` and
  `go test -c -o <scratch>/<os>-<arch>-tests/ ./...`: passed for Windows amd64
  and macOS amd64/arm64. These compile tests without native execution.
- `go build ./...`, `go vet ./...`, `git diff --check`: passed before handback,
  as build/vet did before each implementation commit.

Commands used `GOCACHE=/storage/build/peeragent/job-controls-3-go-cache` and
`GOFLAGS=-p=6`. The final run followed a compatibility correction: blocking
text's early job-id notice goes to stderr, keeping stdout's established
`Alt Subagent:` prefix. The new launch test asserts that prefix for new,
resumed, blocking and async calls. The full suite/validator were repeated
because that source changed; earlier successful runs were not treated as
qualification of the corrected tree.

| Requested outcome | Cause and regression coverage |
| --- | --- |
| Capacity stops and retries | Direct incident already classified at base; compaction prefix did not match. `TestReportedCapacityStopStreams` pins both real terminal shapes and later terminal precedence; `TestIncidentCapacityStopsReportAndRetry` proves stop/result/status and opt-in retry for each. |
| Live progress | Worker stdout stayed empty and raw logs were written after exit. `TestStatusShowsLiveProgressInSingleAndMultipleOutput`, `TestReadProgressUsesBoundedTail` and `TestProgressPreviewSurvivesToolActivityAndRetry` cover live log filling, bounded reads, partial writes, tool/retry continuity and status without a readable full report. |
| Job ids in text and JSON | The common text formatter omitted job ids; blocking calls had no tracked job. `TestEveryLaunchPrintsJobID` covers all four new/resumed and blocking/async combinations, preserved stdout prefix and JSON terminal identity. |
| One writer per session | There was no live-session launch check. `TestResumeGuardRunningTerminalGoneAndOverride`, `TestResumeGuardSerializesConcurrentLaunches`, `TestResumeGuardCLIExitAndOverride` and `TestResumeGuardStorageFailureStaysInfrastructureError` cover conflicts, atomic launch, exited/terminal jobs, override, namespaces, cross-worktree discovery, usage exit 2 and distinct infrastructure errors. |
| Retry prerequisites | The catch-all message did not explain how to fix a blocking resume. `TestCapacityRetryHelpAndResumeUsage` pins help and the instruction to add `--async`, including with `--resume`. |

Result log: `/storage/scratch/peeragent/job-controls-3/verification.log`.
Raw full-suite events: `full-test.jsonl`; validator detail: `validate.log`;
aggregate completion: `verify.exit` (0), all in that same scratch directory.
Logs and the warm Go cache remain for owner review/correction rounds; the owner
removes them after this unit lands or is abandoned. Disposable validation
worktree, generated package/archive outputs, platform test binaries and
one-off editing/verification scripts were removed before handback.

Native macOS/Windows runtime behavior and real model calls were not checked.
Existing Windows cancellation limitations remain. The guard checks numeric PID
liveness; a stale PID reused by another process can falsely refuse a resume
until the explicit override is used. No rate was measured, and the check never
signals that PID. Resume checks scan indexed worktrees once per launch; status
reads only 8 KiB per job. Blocking calls now pay for a worker process and tracked
state, making one shared lifecycle/protection path possible. No performance or
runtime timing measurements were made. No additional in-scope work is deferred;
owner review, merge and release remain with godot-41.


## Round 3 accepted correction: collect blocking worker exits

Godot-41 accepted the review finding “An exited but uncollected Unix worker
still blocks a resume” for immediate correction. The blocking launcher released
its worker handle and polled only for a terminal result. An abnormal worker exit
could therefore hang the caller indefinitely and leave an uncollected Unix
child which `kill(pid, 0)` mistook for an active session writer.

- Blocking launches now retain a worker handle and collect its exit through
  `exec.Cmd.Wait`, including the native handle on Windows. The caller watches
  both persisted completion and worker exit. An exit without a terminal result
  produces a durable failed job with the worker's exact status and `agent.log`
  pointer, removes its PID, and returns CLI exit 1. No wall-clock deadline was
  introduced. A zero-status worker without a result is also a failed job.
- Existing terminal/cancellation results take precedence and are not rewritten
  on normal worker exit. Failure repair uses the existing guarded finish path,
  preserving cancellation if it commits concurrently. Async launches keep their
  existing released-handle behavior.
- Unix liveness no longer treats an exited, uncollected process as a writer:
  Linux checks its process state, and macOS uses a zero-time native exit event.
  Failed observation degrades to the existing process probe. No probe collects
  another owner's child or signals any process. The concurrency override remains.
- `TestExitedUnreapedWorkerDoesNotBlockResume` reproduces the review's Linux
  counterexample using `waitid(WEXITED | WNOWAIT)`. It failed before correction
  with usage exit 2, then passed after the fix. `TestBlockingWorkerExitWithoutResult`
  covers retained ownership and missing-result exits 0/23, stored failure/logs/PID
  cleanup and resumption. `TestWorkerExitPreservesTerminalWinner` covers success,
  failure, capacity and cancellation. `TestBlockingCLIWorkerExitReturnsFailureAndAllowsResume`
  verifies the actual blocking caller returns exit 1 with worker status 23,
  exposes its logs, persists the failed state and permits the next resume.

CHANGELOG Unreleased and the lifecycle docs describe the correction. The first
counterexample output is retained in `fix-1-counterexample.log` in this unit's
scratch. Native macOS/Windows runtime checks remain unavailable in this Linux
unit; qualification is recorded below after the frozen run.


### Accepted worker-exit correction qualification

Tested frozen source: `148f87d3bd8e7324be3d518d94f5c1d6fdb2b447`, with
`go version go1.26.4 linux/amd64`. The final evidence commit changes this record
only. Commands used the same warm
`GOCACHE=/storage/build/peeragent/job-controls-3-go-cache` and `GOFLAGS=-p=6`.

- Focused fake-agent regressions passed: 14 top-level / 30 including subtests,
  across app/jobs/CLI, using `go test -json ./internal/app ./internal/jobs
  ./cmd/peeragent -run 'TestBlockingWorker|TestWorkerExit|TestExitedUnreaped|TestBlockingCLIWorker|TestEveryLaunchPrintsJobID|TestResumeGuard|TestProcessLauncher|TestProcessExists'
  -count=1`. The additional `go test ./internal/app -run TestWorkerExit -count=1`
  passed after adding `TestWorkerExitRepairsUnfinishedTerminalWrite`, which
  preserves result.json and repairs job.json when exit interrupts that update.
- `go test -json ./...`: **241 top-level / 367 total passes**, 12 tested
  packages, zero failures/skips. Testsupport has no tests.
- `./scripts/validate.sh`: passed offline in a disposable checkout of the same
  SHA, including its required suite replay, package/shim checks and Linux/macOS
  archive builds. Existing missing-icon warning remains nonblocking.
- `GOOS=<os> GOARCH=<arch> CGO_ENABLED=0 go vet ./...` and
  `go test -c -o <scratch>/fix-1-<os>-<arch>-tests/ ./...`: passed for Windows
  amd64 and macOS amd64/arm64. Native execution remains unverified there.
- `go build ./...`, `go vet ./...` and `git diff --check`: passed, as build/vet
  did before each code commit. No timing measurement or real model call.

Result log: `/storage/scratch/peeragent/job-controls-3/fix-1-verification.log`;
raw suite: `fix-1-full-test.jsonl`; focused batch: `fix-1-focused.jsonl`;
validator detail: `fix-1-validate.log`; aggregate completion:
`fix-1-verify.exit` (0), all under that same scratch directory. The original
failing counterexample is preserved in `fix-1-counterexample.log`. Logs and
the existing warm cache remain for godot-41's correction review. The temporary
validation worktree, generated archives/binaries, platform test binaries and
one-off scripts created for this correction were removed before handback.

The accepted finding is fixed rather than deferred. All launch/wait changes
stay local to this branch; no installed plugin/binary, release metadata/tag,
main checkout, other sessions' jobs or review files were modified. Godot-41
owns review, landing and release.


## Round 4: delivery receipts, job discovery and output-limit recovery

Requested for v0.11.2 on 2026-10-23. Implemented directly on `round-4`:

- `tell` reports `delivered to job <id> (<agent>, <status>); unread N` after
  persisting the reply. The count and lifecycle come from the delivery lock.
  New jobs record `messaging_enabled`; missing markers warn that replies reach
  the job on `--resume`. Both status formats expose the warning, and delivery
  still succeeds. Explicit `--no-preamble` gets its own accurate warning.
- `peeragent --jobs [--cwd DIR] [--all] [--json | --text]` lists every lifecycle
  in the selected directory, newest first, including unindexed local jobs.
  `--all` adds indexed directories, skipping removed worktrees/absent metadata
  but reporting corrupt or unreadable state. Rows reuse the multi-job contract,
  adding agent and version alongside model, status, creation (`started_at`),
  last message time and session. Empty JSON lists emit no rows.
- `agent_session` is persisted directly in `job.json` on live progress and
  completion; old `spec.resume` remains readable and synchronized.
- Codex's exact incident diagnostic, `Incomplete response returned, reason:
  max_output_tokens`, followed by `turn.failed` with the same message, now
  produces terminal `output_limit`. The partial handback, session and resume
  command remain available. Unrelated mentions/configuration errors do not
  trigger recovery; later authoritative outcomes override diagnostic history.
- Async Codex's existing `--retry-capacity N` budget permits one immediate
  output-limit continuation per job, sharing N with capacity retries. Capacity
  retains its configured delay. A repeated limit, missing session or spent
  budget remains terminal. `metadata.capacity_retries` keeps its compatible
  name and counts all additional attempts. Continuations preserve settings
  and the same session/job, and do not replay the original task.
- `peeragent --version` prints `0.11.2`; single status uses `metadata.version`,
  and multi-job/list rows use `version`, identifying the currently running
  binary even for older jobs. Release bump/validation scripts keep that
  runtime version synchronized with all six manifests. CHANGELOG, README,
  contract/messaging/spec docs and both copies of the peer skill are current.

### Round 4 verification

`go version go1.26.4 linux/amd64`. All requested checks passed:

- `go build ./...` and `go vet ./...`.
- `go test -json ./...`: **283 top-level / 468 including subtests**, 13 tested
  packages, zero failures/skips. Two helper packages have no tests. This adds
  ten top-level tests and 22 total cases over the starting branch, covering
  receipts, legacy/no-preamble warnings, listings/scope/order/stale indexes,
  live session capture, version synchronization, exact output-limit events,
  false positives, shared budgets, repeated limits, missing sessions and
  preservation of terminal output-limit results during worker cleanup.
- `./scripts/validate.sh`: passed, including its required suite replay,
  build/package/shim/documentation checks, runtime version check and Linux/macOS
  amd64/arm64 release archive builds. The existing optional missing-icon
  advisory remains; plugin validation passed.
- `GOOS=windows go vet ./...` and `git diff --check`.

Tests use fake target agents; there were no real model calls or timing runs.
Only repository source/shims/binaries were used for controls and validation;
no installed binary/plugin or main checkout was touched. Native macOS and
Windows execution remain unverified on this Linux host; the publisher owns
Windows CI. No push or tag was made. Generated archives/binaries and this
round's temporary logs are removed before handback. No in-scope work is
backlogged or deferred; owner review and publishing remain with godot-41.


## Round 5: accurate messaging labels, unread age and Windows test capture

Requested for v0.11.3 on 2026-10-23, based on `origin/main` at
`b587e8ec144643b2a65c7614f5514a0101a090a5`, on `round-5` in the existing worktree.
Implemented directly, with no delegated work.

### Messaging evidence

Absent `messaging_enabled` is now distinct from an explicit false flag. Controls
infer missing flags from the saved messaging preamble (including CRLF line
endings), or an outbox `say` entry, including pending/refused attempts. They
report true with `messaging_source: inferred from prompt` or `inferred from
outbox`. Explicit recorded flags remain authoritative. Only missing flags with
neither evidence source get the legacy warning. A prompt read failure remains
an infrastructure error when no alternate outbox evidence is available; missing
prompt files are ordinary absent evidence.

Inference shares the existing mailbox snapshot and does not acquire another
lock. Lifecycle saves preserve absent legacy markers, so an unrelated progress
update cannot convert an unknown legacy flag into a permanently recorded false.

Coverage: `TestMessagingSupportUsesRecordedOrObservedEvidence` (recorded flags,
legacy prompt/outbox/neither, CRLF, missing files and preamble mentions),
`TestLegacyMessagingMarkerStaysAbsentAcrossLifecycleSaves`,
`TestMessagingEvidenceReportsUnreadablePrompt`, and
`TestLegacyMessagingEvidenceAcrossControls` (single/multiple status, jobs, text
and tell, including source labels and warning behavior).

### Oldest unread reply

`--status`, `--jobs` and delivery confirmations show the oldest unread age.
JSON adds `oldest_unread_at` and `oldest_unread_age`; the single-job shape keeps
these in metadata. Empty inboxes omit both; unknown timestamps show `unknown`,
future timestamps clamp to `0s`, seconds are truncated below one minute and
longer ages use whole minutes. Read replies and replies reserved for a running
prompt are excluded. Releasing a reservation makes the original timestamp
visible again. No additional mailbox read or periodic reminder was introduced.

Enabled jobs get the context “the job reads its inbox between long steps”.
Jobs needing resume keep their specific warning, without contradictory context.
`tell` calculates its post-delivery count/age under the delivery lock, using its
existing snapshot and the new reply timestamp; it does not mark a reply read.

Coverage: `TestOldestUnreadAgeUsesOnlyUnreadEntries`,
`TestUnreadAgeFormatting` (seconds/minutes/hours/days, future/unknown timestamps
and empty inboxes), `TestUnreadAgeFollowsReservationAndAcknowledgement`, and
`TestUnreadAgeAcrossStatusJobsAndTell` (exact timestamp, text/JSON age and receipt,
then clearing the age on acknowledgement). The existing delivery receipt test
accepts elapsed age without depending on two commands finishing within a second.

### Windows hang: cause and correction

The shared test helper `captureStdout` replaced stdout with an anonymous pipe,
called the command synchronously, then drained the pipe only after the command
returned. A write larger than the pipe's buffer blocks the writer, so neither
the return nor the drain can happen. The v0.11.2 help output is **4,105 bytes**;
the in-process `--help` subtest in `TestCommandsSnapshotEachMailboxOnce` can
therefore cross the Windows pipe capacity while fitting the Linux/macOS pipes.
Go's Windows pipe implementation calls `CreatePipe` with the default buffer
size. This path does not wait for a child process or hold a job lock while
writing help. The lifecycle lock's own wait is separately bounded.

The helper only discards output, so it now uses the cross-platform `os.DevNull`
sink, restoring stdout and closing its own handle on every return. This removes
both pipe capacity and EOF/handle-inheritance dependencies instead of increasing
a timeout. Production process, lock and messaging transport code is unchanged
by this test fix.

`TestCaptureStdoutHandlesLargeOutput` writes 1 MiB through that helper in an
owned subprocess, with a 30-second safety timeout and a diagnostic naming pipe
capacity. It also checks restoration of stdout and propagation of the command's
error. With the old helper temporarily restored, it failed as expected with
“stdout capture blocked on output exceeding pipe capacity: context deadline
exceeded”; with the corrected helper it passes. The original
`TestCommandsSnapshotEachMailboxOnce` also passes. This reproduces the structural
fault on Linux; native Windows confirmation remains with the publishing owner.

### Round 5 qualification and release preparation

All requested checks passed with `go version go1.26.4 linux/amd64`:

- `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...`.
- `go test -json ./...`: **292 top-level / 495 including subtests**, 13 tested
  packages, zero failures/skips. Nine top-level tests and 27 total cases were
  added over v0.11.2. The two test-support packages have no tests.
- `./scripts/validate.sh`: package/plugin validation, its required suite replay,
  runtime version, metadata/docs/shim checks and Linux/macOS amd64/arm64 release
  builds. The existing optional missing-icon advisory remains nonblocking.
- `git diff --check`; all six manifests, runtime version and skill copies agree.

Release edits/staging follow `scripts/bump.sh`, including
`internal/version/version.go`, the four source manifests/README and regenerated
`plugin/` copies. The script's tag/push stages are not invoked. CHANGELOG and
messaging/contract/spec/skill docs describe the final behavior. No installed
binary/plugin or main checkout was touched, and no tag or push was made.

Tests use fake agents and owned subprocesses; there were no model calls or
performance measurements. Native macOS/Windows execution is unavailable here;
macOS cross-builds and Windows vet passed, and godot-41 owns Windows CI and
publication. Generated release/build files and this round's scratch are removed
before handback; the pre-existing `dist` directory itself is retained. No
in-scope work is backlogged or deferred beyond native Windows confirmation.


### Round 5 pre-release addition: questions-only messaging preamble

Nathan requested the standard preamble's exact three instructions: `say` only
for a question the job cannot settle, never status/progress; the final handback
carries final status; after asking, watch the inbox every few minutes between
steps until answered while doing independent work. The generated prompt,
README, messaging/contract/spec docs and both peer skill copies now agree.
The v0.11.3 CHANGELOG records the change; the release version stays v0.11.3.

Messaging-support inference recognizes the stable original introduction rather
than requiring the whole evolving preamble. A literal original v0.11 preamble
fixture pins compatibility so this wording change cannot undo legacy labels.
`TestMessagingPreambleQuestionsOnly` checks the exact instructions for all four
targets, retained launcher-transport guidance and `--no-preamble` behavior.

All requested checks passed: `go build ./...`, `go vet ./...`,
`go test -json ./...` (**293 top-level / 501 including subtests**, 13 tested
packages, zero failures/skips), `./scripts/validate.sh`,
`GOOS=windows go vet ./...`, and `git diff --check`. The validator includes its
required suite replay and Linux/macOS archive builds; the existing optional
missing-icon advisory remains nonblocking. Native Windows/macOS execution is
still unavailable here. Generated files and this addition's scratch are removed
before handback; the pre-existing dist directory is retained. No blocker,
in-scope deferral, installed-plugin change, push or tag.
