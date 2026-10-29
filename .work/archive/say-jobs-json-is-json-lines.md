---
id: say-jobs-json-is-json-lines
kind: story
stage: done
tags: [cli]
created: 2026-10-27
updated: 2026-10-29
---

# Say that `--jobs --json` prints JSON Lines

voxlar-54 (friction sweep 2026-10-27): `peeragent --jobs --json` prints one JSON object per line, not a JSON array,
so `json.load` on its output fails. Expected: either `--help` says the listing is JSON Lines, or `--jobs --json`
emits a single array. The other `--json` outputs print one object, so a reader assumes a document. Prefer the help
change unless something consumes the line form. The existing `--jobs --all --json` test in
`cmd/peeragent/indexed_jobs_test.go` shows the current shape.

## Resolution (2026-10-29)

Help states `--jobs --json` prints JSON Lines, one object per line. `TestHelpNamesLimitsJSONLinesAndTextFlag` checks the wording; existing indexed listing tests keep the line-oriented output contract.

Implemented in the friction batch; release and installed binaries remain with the owner after review.
