---
id: blocking-poll-reconciles-owned-exit
kind: story
stage: review
tags: [infra, tests]
created: 2026-10-30
updated: 2026-10-30
---

# Keep the blocking caller's worker-exit result

The friction sweep's full package run failed
`TestBlockingCLIWorkerExitReturnsFailureAndAllowsResume`: the fake worker exited
with code 23 without a result, but the blocking call returned `interrupted`
instead of `failed` with that code and its diagnostic log. Five focused repeats
failed the same way after adding Git-directory discovery.

`waitForBlockingWorker` polls through the external observer's `PollJobResult`.
That read reconciles a dead worker to `interrupted` before the blocking caller
can collect its owned handle and run `WorkerExited`. The terminal result then
wins over the owned exit handler, losing the exit code and log reference.

The blocking caller now reads persisted completion through `PollOwnedJobResult`
without reconciling process loss; its owned handle supplies the exit code and
log. External observers keep recovering dead workers. Natural completion and
cancellation retain terminal precedence. `TestOwnedWorkerPollDefersLossToExitHandler`
checks the distinction without racing a timer.

The old-store overlay passed all five worker-exit repetitions; the new lookup
before the ownership fix failed all five. After the fix, the focused ownership,
external recovery and terminal-precedence run passed **4 top-level tests / 17
including subtests**, across two packages:

```sh
go test ./internal/app ./cmd/peeragent -run 'Test(OwnedWorkerPollDefersLossToExitHandler|BlockingCLIWorkerExitReturnsFailureAndAllowsResume|WorkerExitPreservesTerminalWinner|DeadWorkerBecomesInterruptedAcrossReadersAndResume)$' -count=1 -json
```


## Friction batch verification

The storage and diagnostic regressions passed **12 top-level tests / 76 including
subtests**, across four packages:

```sh
go test ./internal/codex ./internal/jobs ./internal/app ./cmd/peeragent -run 'Test(FailureDiagnosticsSurviveBareTerminal|UsageLimitMentionsAreNotProviderStops|FrictionRetryOutcomes|GitJobStorageControls|GitJobStoreIsPerWorktree|GitJobStoreDoesNotHideCorruptPrimary|JobStatusMapsPersistedStates|JobStatusMappingsUsePersistedLifecycleConstants|WorkerExitPreservesTerminalWinner|TerminalResultsSurviveRemovedJobDirectory|ResumeAcknowledgesOnlyAcceptedTargetTurn|ResumeGuardRunningTerminalGoneAndOverride)$' -count=1 -json
```

The reported red-sandbox job's first attempt ended for capacity; its
`target-retry-1.log` instead ended for usage limit. `usage_limited` preserves the
exact provider message and retry time and is never automatically retried.
Separately, preserving an error through a bare `turn.failed` lets repeated
capacity stops consume the complete configured retry budget.

The final source passed `GOFLAGS='-json -count=1' make test`: **364
top-level tests / 698 including subtests**, across 14 tested packages, with no
failures or skips. Two helper packages have no tests. `go vet ./...`,
`git diff --check` and matching bundled skill copies also passed. The initial
full-run worker-exit failure is fixed; no test was relaxed. Verification used
fake targets and local Git fixtures, with no live provider calls. Native
Windows/macOS runtime checks and release packaging are left to the release gate.
