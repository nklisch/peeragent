---
id: name-the-stop-cause-in-a-failed-result
kind: story
tags: [results]
created: 2026-10-27
updated: 2026-10-27
---

# Name the stop cause in a failed job's result

voxlar-a6 (friction sweep 2026-10-27): Codex job `20261027T092354Z-d045bfc7` (cwd `/storage/worktrees/voxlar/06-bt-e`)
was stopped by OpenAI's safety filter during sanitizer fuzz qualification. `--result` showed `status: failed` with
`final_message` set to the last interim message ("Waiting on the sanitized editing build."); the real cause, the
filter's text, was only in `--status`'s `summary`, and target.log's last lines showed a separate "Selected model is
at capacity" error. Expected: a failed result names its stop cause (filter, capacity, sign-in, crash) in a field
the caller reads first, and never presents an interim message as the final one. Workaround: read `--status`.
