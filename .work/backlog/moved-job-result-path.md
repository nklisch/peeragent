---
id: moved-job-result-path
kind: story
tags: [infra, tests]
parent: null
depends_on: []
release_binding: null
created: 2026-10-19
updated: 2026-10-19
---

# Keep job status and result consistent after moving a checkout

## Evidence

The 2026-10-19 session-control review moved a completed fake-agent job into
another directory and supplied that directory through `--cwd`. `--status`
returned `success`, but `--result` returned `running`; `--wait` would keep waiting.
This predates the session locator index changes.

`internal/jobs/store.go` persists absolute `ResultPath`, `PromptPath`, and
`LogPath` values. `internal/app/jobs.go` locates `job.json` under the explicit
directory, but reads the result from the persisted old absolute `ResultPath`.
When the original path no longer exists, the missing result is treated as an
in-flight job. Supplying `--cwd` overrides lookup, not the stored file paths.

## Exposure and direction

This affects users who move completed job directories or their enclosing
checkout. Results remain on disk in the moved directory, but result retrieval
and waiting disagree with status. Frequency was not measured. The session
locator remains only a directory hint; it does not repair relocated jobs.

Resolve job-owned files relative to the located job directory, with an explicit
compatibility rule for existing persisted absolute paths. Cover a completed
fixture moved away from its original directory: status, result and wait should
agree when given the new `--cwd`. Check prompt/log consumers as part of that
future unit. Deferred by adjudication; no relocation behavior changed here.
