---
id: make-a-text-mode-launch-show-its-job-id
kind: story
stage: done
tags: [cli]
created: 2026-10-25
updated: 2026-10-29
---

# Make a text-mode launch show its job id

`--text` is an output-format flag, and the task text is positional. A caller ran `peeragent --resume … --text "prompt"`
expecting `--text` to take the prompt; the job did launch with that text, but the launch printed text instead of
JSON, so the caller lost the job id and had to find it under `.peeragent/jobs`. Outcome: an async launch in text mode
always prints its job id on a line of its own (and the `--wait` command to use), and either a `--prompt TEXT` option
exists or `--text` refuses to be read as taking a value. Acceptance: a test where `--async --text` launch output
contains the job id; the chosen option or refusal has a test.

## Resolution (2026-10-29)

Text results print `Job: <id>` on its own line and running results print a `peeragent --wait <id>` command. Help says `--text` takes no value; task text remains positional. `TestTextAsyncLaunchNamesJobAndWait` and `TestHelpNamesLimitsJSONLinesAndTextFlag` cover the behavior.

Implemented in the friction batch; release and installed binaries remain with the owner after review.
