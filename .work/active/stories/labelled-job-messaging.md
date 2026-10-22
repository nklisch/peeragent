---
id: labelled-job-messaging
kind: story
stage: review
tags: [infra, tests, docs, security]
parent: null
depends_on: []
release_binding: v0.11.0
created: 2026-10-22
updated: 2026-10-22
---

# Labelled two-way messaging between jobs and launchers

Nathan requested a labelled channel from delegated jobs to their launching
sessions and a reply mailbox. Godot-41 owns publication; this unit starts at
published v0.10.1 (`bc76979`) and prepares v0.11.0 without tag, push, installed
binary changes or real session contact. The brief is
`/storage/scratch/workstations/briefs/peeragent-messaging.md`.

## Implementation

- Capture a private launcher channel before worker startup. Targets retain the
  terminal isolation policy and receive job id/directory through execution
  context, including resumes and capacity retries.
- Implement `say` through Herdr's documented prompt argv, with the mandatory
  origin/reply header, neutralized body brackets, character/rate limits and an
  auditable timestamped outbox. Herdr calls run outside the job lock; pending
  reservations enforce the rate bound across concurrent senders.
- Implement `tell` and `inbox` with the same lock for appends, printing and read
  acknowledgement. Failed output leaves replies unread. Commands announce
  unread counts on stderr. Prompts explain messaging unless `--no-preamble`.
- Reserve carried replies by attempt before adding them to resumed prompts.
  Reserved entries are hidden from inbox output/notices/counts; later replies
  remain independently readable. Target output and a non-capacity outcome mark
  reservations read; non-start/refusal releases them. Share session discovery
  with the existing concurrent-resume check. Capacity continuations use their
  own mailbox, and crashed attempts recover on the next read.
- Expose counts/time in both status forms and ordered conversation after agent
  details in results. Windows supports mailbox commands and records the explicit
  no-launcher-channel refusal for `say`.
- Update foundation docs, both bundled skill copies, CHANGELOG and all six
  versioned manifests to v0.11.0.

## Accepted review correction

Godot-41 accepted all five findings from the review of `e999cc6`. The correction
brief is `/storage/scratch/workstations/briefs/peeragent-messaging-fix-1.md`.
All are fixed, with no in-scope work deferred:

1. **Replies survive refused/unstarted turns.** Resume jobs persist mailbox
   references; each target attempt snapshots unread entries and includes them
   afresh. Only target output followed by a non-capacity execution outcome can
   acknowledge the included entries. Worker startup alone, target startup
   failure, no target event, cancellation and capacity refusal preserve them.
   Acknowledgement checks only the cursor and cannot advance it twice. Covered
   by `TestRetryStartupFailurePreservesUnreadReply`,
   `TestCapacityRepliesReachSuccessfulTurn`,
   `TestResumeAcknowledgesOnlyAcceptedTargetTurn`,
   `TestCancelledTurnLeavesCarriedRepliesUnread`, and target-adapter event/
   print-envelope acceptance tests. The first two reproduce the review probes:
   failure after capacity followed by recovery, and two refusals then success.
2. **Directory aliases carry each reply once.** Session discovery deduplicates
   loaded jobs by persisted job ID, with no filesystem identity probing. Inbox
   sources are also deduplicated by that ID. Covered by
   `TestResumeDirectoryAliasIncludesEachReplyOnce` using real/alias paths.
3. **Named Unicode visual tricks are flattened.** Bodies map vertical space,
   including U+2028/U+2029, to spaces, remove U+200E/U+200F, U+202A–U+202E and
   U+2066–U+2069, and map the supported and named look-alike bracket pairs to
   ordinary brackets. Labels remove direction formatting too. Covered by
   `TestMessageSanitizerFlattensUnicodeVisualTricks` and the fake-Herdr payload
   checks in `TestSayLabelBoundsAndConversation`. Docs claim only these specific
   normalizations, never general Unicode spoof prevention.
4. **One mailbox snapshot per job per command.** A command-scoped snapshot is
   shared by unread notices and output. Inbox/say take their first snapshot in
   the mutation lock. Outbound completion appends an outcome with the reserved
   message ID, and readers fold the journal into one entry; completion does not
   rescan history. No incremental count state was added. Covered by
   `TestCommandsSnapshotEachMailboxOnce` (inbox, say, status/result, waits,
   repeated status IDs, help/errors and another wait-any winner), plus the
   existing concurrent send/receive and rate-limit tests.
5. **Trust boundary stated honestly.** `docs/MESSAGING.md` and both synchronized
   skill copies explain accidental-input protection for well-meaning agents,
   same-user access to launcher state/direct Herdr by a deliberately misbehaving
   job, and the supported workflow rather than an adversarial boundary. Static
   copy equality and the full validator passed.

Changed production areas: CLI command context/worker retry loop, application
launch/mailbox services, job references/session discovery/mailbox journal,
shared execution result, and all four target adapters. Tests, CHANGELOG,
README, foundation docs and both skill copies were updated. The six manifests
remain at v0.11.0. No unrelated production bug was discovered or deferred.

## Attempt reservation correction

Godot-41 confirmed fixes 2–5 and the original reply-loss cases, then requested
three delivery states in
`/storage/scratch/workstations/briefs/peeragent-messaging-fix-2.md`: unread,
reserved by an attempt, and read. This remaining interaction is fixed:

- Preparing a prompt reserves its selected entries under the inbox lock and
  records the attempt ID, worker PID and private outcome-file path with them.
  Inbox output, unread notices and status counts exclude reservations. Sparse
  state records later read entries separately, so reading a new arrival cannot
  acknowledge an earlier reserved reply.
- Accepted attempts durably publish acceptance before settling source mailboxes.
  Non-start, cancellation, no target event and capacity refusal publish release
  and make the entries unread again. Repeated settlement does not advance or
  rewrite already-settled entries. A preparation failure also releases any
  reservations already created.
- Each mailbox read recovers stale reservations. An accepted terminal record
  commits read state even if the worker exited mid-settlement; a dead recorded
  worker with no terminal outcome releases the reservation. The terminal record
  is checked again after observing death, so acceptance published immediately
  before exit cannot be mistaken for abandonment. Recovery never signals a
  process, and uses no filesystem identity machinery.
- `inbox-state.json` stores read/reserved entry state; private
  `inbox-attempts/<attempt-id>.json` stores the attempt's running/accepted/released
  outcome. Older cursor-only mailboxes remain readable. Conversation history is
  unchanged. No incremental message-count state was added; the single command
  snapshot still serves notices, status and result output.

Regression evidence:

- `TestCarriedReplyIsNotDeliveredAgainByRunningInbox` reproduces the reviewer's
  running-target case. The carried instruction occurs once in the retry prompt,
  then inbox output and its notice are empty. A later reply appears once with
  count one, remains independently readable, and final acceptance commits both
  messages without skipping either.
- `TestInboxAttemptCrashRecovery` uses an owned, waited-on test subprocess to
  leave reservations behind. Abandoned and missing terminal records recover to
  unread; an accepted record recovers to read. A second read is stable.
- `TestInboxReservationsPreserveIndependentDeliveryOrder` accepts a later reply
  before releasing an earlier reservation, then delivers only the released one.
- `TestInboxAttemptAcceptanceWinsProcessExitRace` deterministically publishes
  acceptance between the initial metadata read and the observed process exit.
- Earlier startup-failure, two-capacity-refusals-then-success, cancellation,
  alias, concurrent inbox/send, rate-bound and one-snapshot tests remain green.
  The acceptance test now creates a fresh attempt for each refused turn and
  additionally asserts release restores its unread count.

Changed source: `internal/jobs/inbox_attempt.go`, `internal/jobs/messages.go`,
`internal/app/messaging.go`, CLI unread notices and their tests. Messaging docs,
README, CHANGELOG, both skill copies and this handback record are synchronized.
The six manifests remain at v0.11.0. No in-scope work is deferred.

## Verification and handback

Frozen source tested: `7bc10dbb7325d0371188ff0896fd93269c3f5b25`, with
`go version go1.26.4 linux/amd64`. The final handback commit changes only this
record. All checks passed:

- Focused correction run: **18 top-level / 35 including subtests**, three
  packages, zero failed or skipped tests. `verification.log` records the exact
  command/filter alongside `focused-test.jsonl`.
- `go build ./...`, `go vet ./...`, `git diff --check` before the source commit.
- `go test -json ./...`: **273 top-level / 446 including subtests**, 12 tested
  packages, zero failed or skipped tests; two helper packages have no tests.
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -o
  /storage/scratch/peeragent/messaging-fix-2/windows-tests/ ./...`, plus Windows
  `go vet ./...` in that same platform environment.
- `GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go test -c -o
  /storage/scratch/peeragent/messaging-fix-2/darwin-amd64-tests/ ./...` and the
  corresponding `GOARCH=arm64`/`darwin-arm64-tests` command. Both compile tests
  as well as production code.
- `./scripts/validate.sh` from a disposable worktree at the frozen source:
  full tests, build, plugin validation/metadata, release archives, documentation
  and shim smokes all passed. The brief explicitly requests this validator,
  which includes its own full-suite invocation. Version/skill copy checks passed.

Go commands reused `GOCACHE=/storage/build/peeragent/job-controls-3-go-cache`
and `GOFLAGS=-p=6`; final runs used
`GOTMPDIR=/storage/scratch/peeragent/messaging-fix-2/go-tmp`.

Raw evidence remains under `/storage/scratch/peeragent/messaging-fix-2/`:
`verification.log`, focused/full JSON and counts, platform compile/vet logs and
`validate.log`. Disposable validation worktree, release products, cross-test
binaries and one-off verification script were removed. The existing Go cache
remains for review/corrections; the owner removes evidence/cache after landing.

Native Windows/macOS execution was unavailable on this Linux host; their test
compiles passed. Tests used fake targets/Herdr and owned test subprocesses only.
No session was contacted. No peeragent delegation, installed binary/plugin
update, push or tag was performed. No blockers remain.
