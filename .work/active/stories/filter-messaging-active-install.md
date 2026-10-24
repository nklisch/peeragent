---
id: filter-messaging-active-install
kind: story
stage: review
tags: [infra, tests, docs]
parent: null
depends_on: []
release_binding: null
created: 2026-10-24
updated: 2026-10-24
---

# Explain provider filter stops, read message files, and follow the active install

The three reported Voxlar attempts ended with the same structured `turn.failed`
cybersecurity diagnostic. Current source already classifies it as
`failure: policy_refusal`; the stale installed wrapper explains why that fix
did not reach those jobs. Preserve that classification and failed status.
Add `metadata.stop_reason: provider_safety_filter` and `metadata.stop_message`
with the exact provider diagnostic. The summary names OpenAI's safety filter
and warns that resuming this history is likely to fail again; suggest a fresh
session for unrelated steps. Other failures keep existing semantics. A fixture
retains the real error/turn-failure envelopes, with synthetic session/progress.

`tell` and `say` read `--prompt-file` using launch's whitespace trimming and
joining rule: positional text first, then file content, separated by a blank
line. The launching session corrected the brief on 2026-10-24: launch already
accepts combined input, and all three commands should behave consistently.
`inbox` still takes no arguments. Messaging does not gain implicit stdin reads.

After the existing binary override and developer source paths, the POSIX
wrapper consults Claude Code's installed-plugins record and forwards once to
the active wrapper when different. Preserve arguments and exit status, and
use an exported marker to prevent redirect loops. Missing/unreadable records,
missing targets, or unavailable JSON readers fall back to this install.
Use the wrapper's cache location to identify its harness and marketplace;
never redirect a Codex install through Claude's separate registry. Local Codex
inspection found plugin enablement in config.toml and versioned cache entries,
but no installed-plugins record equivalent naming an active install path; do
not guess a version from cache directories.

Adds: two optional result metadata fields, message-file input, and one internal
redirect marker. No persistent cache, service, required dependency, version
bump, release, or new operational refusal. Existing source and explicit binary
choices retain precedence. JSON parsing uses an available reader and otherwise
keeps the current executable usable.

Verification: focused Go adapter/application/input/wrapper tests while editing;
then the requested full Go suite, vet, formatting, shell syntax/smoke checks,
and diff checks once on the final source. Record exact commands, tested SHA,
and consolidated log in the provided scratch directory. Do not release or
edit installed caches; provide a cache backpatch for the owner.


## Verification and handback

Implemented on the requested branch without release or cache changes. The
final source passed `go test ./... -count=1 -json`: 332 top-level tests and
624 cases including subtests, across 14 tested packages (two further packages
have no tests). This includes all ten focused regression tests. `go vet ./...`,
Go formatting, `sh -n` and `bash -n` for every shell script/wrapper, curated
wrapper/skill synchronization, and `git diff --check` passed.

The compact single-job status result initially omitted the new stop metadata;
the new CLI regression caught that omission and the status path now copies it.
Result, wait, status and archived-result checks pass. Both JSON-reader paths,
missing-reader fallback, PID-scoped loop prevention, separate harness caches,
and unchanged arguments/exit status are covered by wrapper tests.

The supplied jq backpatch was checked against scratch copies of all three
installed wrapper versions, 0.12.2, 0.12.3 and 0.13.0. The owner can insert the
snippet after the source-checkout branch, before committed-platform lookup.
The installed caches were only read. The internal marker carries the forwarding
process ID, so an inherited marker cannot suppress resolution in child commands.

The requested tested SHA and consolidated evidence are recorded in
`/storage/scratch/peeragent/sweep-filter-tell-active/verification.log`; the exact
cache snippet is `cache-backpatch.sh` beside it. Native macOS/Windows execution
and release artifact generation were not run. The full release validator
creates release artifacts, so the requested Go and shell checks ran directly.
No additional work was backlogged. Release and cache patch application belong
to the launching owner.
