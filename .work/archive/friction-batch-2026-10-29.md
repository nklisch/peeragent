---
id: friction-batch-2026-10-29
kind: story
stage: done
tags: [infra, tests, docs, security]
parent: null
depends_on: []
release_binding: null
created: 2026-10-29
updated: 2026-10-29
---

# Recover interrupted jobs and remove recurring CLI friction

## Delivered parts

1. Running jobs record worker boot ID where available, PID and process start
   time. Status, result, wait, wait-any and resume reconcile proven worker loss
   to interrupted, retaining the session and last message. Terminal publication
   wins under the existing writer boundary. Compact index snapshots also retain
   identity for removed-worktree reads. Linux uses procfs, macOS uses the kernel
   process start timestamp with microseconds, and Windows uses process creation
   time. Kernel-backed short locks release on death and serialize recovery of
   stale owner files. Tests: `TestDeadWorkerBecomesInterruptedAcrossReadersAndResume`,
   `TestLostWorkerControlsTerminateAndResume`, `TestIdentityDetectsReboot`,
   `TestIdentityMatchesLiveWorkerAndRejectsReusedPID`,
   `TestLiveWorkerAndTerminalWinnerArePreserved`,
   `TestStaleLockRecoverySerializesContenders`,
   `TestCancelNeverSignalsMismatchedOrLegacyPID`.
2. Missing unindexed jobs return not_found and exit 4, naming the searched folder.
   Status/result/history and multi-job reads use the indexed location after a
   miss, naming both folders in lookup_warning. Corrupt/unreadable local state
   keeps its error. Test: `TestWrongDirectoryUsesIndexOrReportsNotFound`.
3. Command-like task input followed by a job ID, and a bare job ID, exit 2 before
   job creation. The diagnostic lists commands and the explicit `--` escape.
   Prompt files and explicit task input retain normal launch behavior; stdin
   receives the same typo check. Tests: `TestCommandLikeTasksRequireExplicitInput`,
   `TestUnknownSubcommandDoesNotLaunch`.
4. Existing retained metadata already implements the requested session, options,
   summary and timestamp retention: 30 days, at most 10,000 jobs, summaries capped
   at 600 Unicode characters. Its local-file precedence and removed-worktree
   resumption remain in place. Worker identity extends its recovery behavior.
   Tests: `TestRemovedWorktreeJobsAndResumeCLI`,
   `TestIndexedMetadataLegacyAndRetentionBounds`,
   `TestRemovedWorktreeRetainsWorkerRecovery`.
5. `messages <job-id> [--cwd DIR] [--json | --text]` prints both directions,
   timestamped and oldest first, without consuming inbox replies. JSON is JSON
   Lines; outbound outcome updates collapse to their latest delivery state.
   Test: `TestMessagesHistoryBothDirectionsWithoutAcknowledgement`.
6. Help exposes say's 2,000-character cap. Rate refusal gives the exact UTC time
   the next native send is allowed, preserving three attempts per five minutes.
   Tests: `TestSayRateRefusalNamesExactNextSend`,
   `TestHelpNamesLimitsJSONLinesAndTextFlag`.
7. Text launches print a standalone Job line and a wait command; help says text
   is a format flag with no value. Failed results keep the stop cause in summary
   and the last agent update in details, leaving final_message null, including
   reads of older failed results. Listing help explicitly names JSON Lines.
   Tests: `TestTextAsyncLaunchNamesJobAndWait`,
   `TestFailedCodexJobFinalMessage`,
   `TestProviderFilterResultExplainsStopAndPreservesDiagnostic`,
   `TestHelpNamesLimitsJSONLinesAndTextFlag`, plus existing recovery and worker-exit tests.

## Closed reports

The archived items are gate-security-stale-job-lock,
 gate-security-cancel-pid-reuse, say-no-such-job-for-a-wrong-cwd,
 make-a-text-mode-launch-show-its-job-id, name-the-stop-cause-in-a-failed-result,
 and say-jobs-json-is-json-lines. The retained metadata report was already closed
 in retain-cross-directory-job-metadata; its behavior was confirmed rather than
 reimplemented.

Completion notifications or a detached wait exit-file mode are deferred in
`completion-notification-or-detached-wait-exit-file`, carrying the voxlar-8e and
godot-34 sweep reports. They require a completion-delivery contract outside this batch.

## Verification and limits

`make build`, `make test`, `scripts/validate.sh`, `go vet ./...`, Go formatting,
and `git diff --check` passed. Full validation covers 14 tested Go packages,
source/shim smoke checks, plugin metadata and examples, and release archive
builds for Linux amd64/arm64 and macOS amd64/arm64. Claude plugin validation
passes with the existing nonblocking missing-icon warning. Final macOS builds
and a Windows amd64 build also compile the native identity adapters.
The kernel timeval decoder is tested for subsecond identity preservation.
Native macOS and Windows execution was not available on this Linux host.
Focused worker recovery, terminal precedence, cancellation and stale-lock
contention tests pass with the Go race detector. The contention fixture uses
atomic overlap counters because the race detector does not infer Go memory
ordering from kernel file locks.

A legacy job with neither original files nor retained metadata/archive cannot
be recovered. Without a saved start time, cancellation records cancelled state
but does not signal an unverified PID. Identity verification and signalling
remain separate kernel operations: a narrow exit/reuse race remains a known
limit. Unknown probes alone do not prove interruption. Unknown legacy lock
owners retain a bounded wait and manual recovery or concurrent-session override.

No version bump, committed platform-binary regeneration, installed-plugin change,
publication, push or release is part of this unit. The owner reviews, selects the
next version, regenerates the committed binaries and release archives, then
ships. Temporary verification builds and logs are removed after their outcomes
are recorded here; the supplied worktree and branch remain for review.
