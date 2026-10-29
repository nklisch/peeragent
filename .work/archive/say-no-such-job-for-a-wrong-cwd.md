---
id: say-no-such-job-for-a-wrong-cwd
kind: story
stage: done
tags: [results, wording]
created: 2026-10-28
updated: 2026-10-29
---

# Say "no such job under this folder" when `--status` gets the wrong `--cwd`

voxlar-75 (friction sweep 2026-10-28): `peeragent --status 20261028T183520Z-46cddb21 --cwd
/storage/worktrees/voxlar/75-traversal-capture` returned `"status":"failed"`, exit 4, with a `job.json` lookup error.
The job was running fine in `/storage/worktrees/voxlar/75-movers-unit3`; the folder was the caller's mistake.

**Expected:** a lookup miss reads as a lookup miss, such as "no job 20261028T183520Z-46cddb21 under
/storage/worktrees/voxlar/75-traversal-capture", with a status that isn't a job outcome. Where the locator index
knows the job's real folder, name it. A reader shouldn't take a live job for a dead one.

**Not this item:** the exit code for a missing job (4) can stay; only the status and wording change.

## Resolution (2026-10-29)

Missing unindexed jobs return `not_found`, exit 4, and the searched folder. Missing selected state with an indexed location reads that location and reports both folders in `metadata.lookup_warning`. `TestWrongDirectoryUsesIndexOrReportsNotFound` proves both paths.

Implemented in the friction batch; release and installed binaries remain with the owner after review.
