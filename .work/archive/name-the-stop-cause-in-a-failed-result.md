---
id: name-the-stop-cause-in-a-failed-result
kind: story
stage: done
tags: [results]
created: 2026-10-27
updated: 2026-10-29
---

# Name the stop cause in a failed job's result

voxlar-a6 (friction sweep 2026-10-27): Codex job `20261027T092354Z-d045bfc7` (cwd `/storage/worktrees/voxlar/06-bt-e`)
was stopped by OpenAI's safety filter during sanitizer fuzz qualification. `--result` showed `status: failed` with
`final_message` set to the last interim message ("Waiting on the sanitized editing build."); the real cause, the
filter's text, was only in `--status`'s `summary`, and target.log's last lines showed a separate "Selected model is
at capacity" error. Expected: a failed result names its stop cause (filter, capacity, sign-in, crash) in a field
the caller reads first, and never presents an interim message as the final one. Workaround: read `--status`.

## Resolution (2026-10-29)

Failed execution summaries retain the stop cause instead of being overwritten by the last agent update. Failed jobs keep their last message in details and leave final_message null. Provider-filter, capacity, authentication, worker-exit and failed-message regressions cover stop wording and retained session/message context.

Implemented in the friction batch; release and installed binaries remain with the owner after review.
