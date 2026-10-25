---
id: retain-cross-directory-job-metadata
kind: story
stage: review
tags: [infra, tests, docs]
parent: null
depends_on: []
release_binding: null
created: 2026-10-24
updated: 2026-10-24
---

# Retain cross-directory job metadata after worktree removal

The existing job index stores only directories. Complete terminal results are
already archived for 30 days, but listings and resume-option inheritance only
scan live job directories. Removing a worktree therefore hides the session from
`--jobs --all`, even while a result snapshot remains available by known ID.

Extend the existing per-job index entry to compact JSON metadata. Read legacy
path-only entries too; do not add a daemon or another canonical job store.
Capture ID, agent/model/effort, directory, last known status, session, start and
finish times, and a bounded final summary. Existing local files remain the
source of truth; the indexed snapshot is used only when local metadata is absent,
never when local metadata is corrupt or unreadable. Full result archives retain
their existing precedence and contents. Metadata-only output is explicitly
marked, and missing local mailboxes are never recreated during reads or resume.

Write snapshots at launch, when a live session is learned, and at completion.
The job index retains at most 10,000 entries updated within 30 days, with
summaries limited to 600 Unicode characters. Prune on launch/completion, not on
every progress event. Session-directory mappings keep their existing format.
`--jobs --all` also promotes legacy archived results into compact entries on
first encounter, skipping archives already represented in the metadata index.
This permits discovery of jobs completed before this update without repeatedly
reading complete handbacks. Full archives keep their existing 30-day lifetime.

Resume discovery can use indexed model/effort/session metadata while excluding
those entries from live-process checks and inbox carryover. A removed directory
still needs an explicit `--cwd` pointing at the replacement worktree; retained
metadata does not recreate the old worktree or pretend its files are available.

Adds optional indexed-only warnings and finish times to listing/result output,
plus compact JSON entries in the existing job index. Legacy archive times use
the timestamp encoded in the job ID and the archive file modification time. Retires no supported
commands or full-archive behavior. Index-write failures retain the existing
local-operation fallback and expose warnings rather than rejecting completed
work. The implementation stays in the current jobs/application boundaries.

The peer skill's reply example uses single quotes or a prompt file so the shell
does not expand backticks before peeragent receives the message.

Verification: focused storage, listing, removed-worktree and resume regressions;
then the requested full Go suite, vet, formatting, shell syntax/smoke checks and
`git diff --check`. Record one consolidated log and tested SHA in the supplied
scratch directory. No merge, push, tag, release or installed-cache edits.


## Verification and handback

The final source passed 23 focused top-level tests (69 cases including subtests),
then `go test ./... -count=1 -json`: 338 top-level tests and 640 cases, across
14 tested packages. Two packages have no tests. `go vet ./...`, the Windows
amd64 cross-build, Go formatting, `sh -n` and `bash -n` for every repository
shell script, wrapper/skill synchronization, and `git diff --check` passed.

Six added regression tests cover the retention bounds, Unicode summaries,
legacy archive promotion (including replacing a running snapshot without
producing duplicate rows), removed worktrees across eight lifecycle states,
metadata-only CLI reads and resumed target arguments, local-file precedence,
corruption reporting, and returning from waits on indexed-only running history.
Resume tests verify that old mailboxes and removed directories are not recreated.

Full result archives retain their existing contents. The new index writes happen
at existing launch/session/completion boundaries; listing reuses metadata it
already loaded. Historical imports do not overwrite the session directory
pointer. Missing session-directory mappings can instead be recovered from the
latest indexed job for that agent/session.

The tested SHA, exact final commands and consolidated output are in
`/storage/scratch/peeragent/sweep-3/verification.log`. Native Windows/macOS
execution and release artifact generation were not run. A legacy job with
neither original files nor a retained archive cannot be recovered. No additional
work was backlogged. No merge, push, tag, release, deployment or cache mutation
was performed; landing and plugin release/update remain with the owner.
