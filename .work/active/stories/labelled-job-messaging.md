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
  snapshot after successful startup. Share session discovery with the existing
  concurrent-resume check. Capacity continuations use their own mailbox.
- Expose counts/time in both status forms and ordered conversation after agent
  details in results. Windows supports mailbox commands and records the explicit
  no-launcher-channel refusal for `say`.
- Update foundation docs, both bundled skill copies, CHANGELOG and all six
  versioned manifests to v0.11.0.

## Verification

Pending frozen-source build, vet, full Go suite, Windows compile and offline
packaging validation. Focused fake-agent/fake-Herdr tests cover the requested
contracts. No installed target CLI or real Herdr prompt command is used.

No in-scope work is deferred. Native macOS/Windows runtime qualification is
unavailable on this Linux workstation; cross-compilation will be recorded.
