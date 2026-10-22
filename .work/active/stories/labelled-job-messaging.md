---
id: labelled-job-messaging
kind: story
stage: implementing
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
- Carry unread messages into resumed prompts; acknowledge only the included
  snapshot after target output and a non-capacity outcome. Share session discovery with the existing
  concurrent-resume check. Capacity continuations use their own mailbox.
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

## Verification and handback

Frozen source tested: `8835eb84fc965decd58656c60e4957f51fe08f0e`, using
`go version go1.26.4 linux/amd64`. The final handback commit changes only this
record. All checks passed:

- Focused correction run: **24 top-level / 44 including subtests**, seven
  packages, zero failed or skipped tests. The exact filter is recorded in
  `verification.log` alongside the focused JSON results.
- `go build ./...`, `go vet ./...`, `git diff --check` before the source commit.
- `go test -json ./...`: **269 top-level / 439 including subtests**, 12 tested
  packages, zero failed or skipped tests; two helper packages have no tests.
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -o
  /storage/scratch/peeragent/messaging-fix-1/windows-tests/ ./...`, plus Windows
  `go vet ./...` in that same platform environment.
- `GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go test -c -o
  /storage/scratch/peeragent/messaging-fix-1/darwin-amd64-tests/ ./...` and the
  corresponding `GOARCH=arm64`/`darwin-arm64-tests` command. Both compile tests
  as well as production code.
- `./scripts/validate.sh` from a disposable worktree at the frozen source:
  full tests, build, plugin validation/metadata, release archives, documentation
  and shim smokes all passed. The brief explicitly requests this validator,
  which contains its own full-suite invocation.

Go commands reused `GOCACHE=/storage/build/peeragent/job-controls-3-go-cache`
and `GOFLAGS=-p=6`; final runs used
`GOTMPDIR=/storage/scratch/peeragent/messaging-fix-1/go-tmp`.

Raw review evidence remains under
`/storage/scratch/peeragent/messaging-fix-1/`: `verification.log`, focused/full
JSON results and counts, platform compile/vet logs, and `validate.log`.
Disposable validation worktree, release products, cross-test binaries and the
one-off verification script were removed. The existing unit Go cache remains
for correction/review reuse; the owner removes evidence/cache after landing.

Native macOS/Windows runtime qualification was unavailable on this Linux host;
only their compiles were run. Delivery tests used fake Herdr and synthetic
addresses throughout. No session was contacted, and no peeragent delegation,
installed binary/plugin update, push or tag was performed. No blockers remain.

## Attempt reservation correction

The second correction brief,
`/storage/scratch/workstations/briefs/peeragent-messaging-fix-2.md`, accepts the
three-state delivery model: unread, reserved by an attempt, and read. Carried
replies are reserved under the inbox lock and hidden from inbox output, notices
and unread counts. Sparse read state lets later arrivals be consumed without
acknowledging an earlier reserved reply. A terminal attempt outcome precedes
mailbox settlement, and a read recovers abandoned reservations by the recorded
worker PID without signalling anything.

Focused regressions pass for the review's duplicate case, earlier loss cases,
independent reservations, later arrivals and crash recovery. Final full suite,
validator and cross-platform compile evidence is pending.
