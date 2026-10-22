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

## Verification

Frozen source tested: `7cc3c221b6831dc017752b0a76acc178706b20dc`, with
`go version go1.26.4 linux/amd64`. The final handback commit only updates this
record. All checks passed:

- `go build ./...`, `go vet ./...`, `git diff --check` before the source commit.
- `go test -json ./...`: **260 top-level / 418 including subtests**, 12 tested
  packages, zero failed or skipped tests; two helper packages have no tests.
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -o
  /storage/scratch/peeragent/messaging/windows-tests/ ./...` and Windows
  `go vet ./...` with the same platform environment.
- `GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build ./...` and the corresponding
  `GOARCH=arm64` build.
- `./scripts/validate.sh` from a disposable worktree at the frozen source:
  full tests, build, plugin validation/metadata, release archives, documentation
  and shim smokes all passed. Generated products were stored under `/storage`
  and removed afterward. All six manifests and curated skill copies match.

Go commands reused `GOCACHE=/storage/build/peeragent/job-controls-3-go-cache`
and `GOFLAGS=-p=6`; final runs used
`GOTMPDIR=/storage/scratch/peeragent/messaging/go-tmp`. The validator explicitly
requested by the brief includes its own full-suite invocation.

The thirteen new top-level tests cover labels and Unicode bounds, header/body
sanitization, shared rate limits including concurrent senders and uncertain
Herdr failures, no-channel/transport refusals, tell/inbox and concurrent writers,
failed stdout delivery, unread notices (including help/errors), raw-task opt-out,
resume startup/snapshot acknowledgement, capacity retry carryover, private state,
all-target job identity with terminal isolation, and status/result conversation.
Herdr was inspected only with help commands; all delivery tests use a fake binary
and synthetic addresses. No real session was contacted.

Raw evidence for review is retained under
`/storage/scratch/peeragent/messaging/` (`verification.log`, `full-test.jsonl`,
`test-counts.json`, platform logs and `validate.log`). Disposable validation
worktree, release products, cross-test binaries and one-off verification script
were removed. The existing unit Go cache remains for correction/review reuse;
the owner removes retained evidence/cache after landing.

No push, tag, installed binary update or installed plugin change was made.

No in-scope work is deferred. Native macOS/Windows runtime qualification is
unavailable on this Linux workstation; cross-compilation will be recorded.

## Accepted review correction

Godot-41 accepted all five findings from the review of `e999cc6`: preserve
replies across refused/unstarted turns; deduplicate alias-discovered job IDs;
flatten the named Unicode visual tricks; share one mailbox snapshot per job per
command; and document the same-user, well-meaning-agent trust boundary honestly.
The correction brief is
`/storage/scratch/workstations/briefs/peeragent-messaging-fix-1.md`.

Implementation and focused regressions are complete. Final frozen-source full
suite, validator, Windows and macOS compile evidence is pending below.
