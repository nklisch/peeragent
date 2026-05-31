---
id: committed-binary-distribution-shim
kind: story
stage: implementing
tags: [infra]
parent: committed-binary-distribution
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-31
updated: 2026-05-31
---

# Shim resolution rewrite

## Scope

Implements Unit 1 of `committed-binary-distribution`. Rewrite `bin/peeragent` so it
resolves the wrapper without any network fetch and fails with a clear, contract-shaped
install error on unsupported platforms.

New resolution order:
1. `$PEERAGENT_BIN` (if executable).
2. `$PLUGIN_ROOT/dist/peeragent` (local `make build` output).
3. `go run $PLUGIN_ROOT/cmd/peeragent` when `cmd/peeragent` exists and `go` is on PATH
   (source-checkout dev path; reflects edits).
4. Committed platform binary `$PLUGIN_ROOT/bin/<goos>-<goarch>/peeragent`, where the
   target is `$PEERAGENT_TARGET_OVERRIDE` or detected from `uname -s`/`uname -m`.
5. Not installed → contract-shaped JSON failure on stdout with
   `metadata.exit_code: 3`; `--text` prints human-readable guidance to stderr; exit 3.

Remove the manifest-version lookup, `~/.cache/peeragent` cache, the `curl`/`tar`
download block, and the `PEERAGENT_VERSION` / `PEERAGENT_CACHE_DIR` /
`PEERAGENT_RELEASE_BASE` / `PEERAGENT_SKIP_DOWNLOAD` env vars. See the feature body
Unit 1 for the full shim source.

## Acceptance Criteria

- [ ] Packaged shim path (no `cmd/`) execs the committed host-arch binary;
      `--status missing-job` exits `4` with contract JSON.
- [ ] `PEERAGENT_TARGET_OVERRIDE=plan9-sparc` with no matching binary exits `3` and
      emits `{"status":"failed",...}` containing the GitHub releases URL.
- [ ] `--text` on the not-installed path writes a human-readable message to stderr,
      exits `3`.
- [ ] Source checkout with Go still uses `go run` (no committed binary needed).
- [ ] No `curl`, cache, version-manifest, or download code remains.

## Notes

- JSON to stdout (matches the Go binary); text to stderr.
- `$TARGET` is `[a-z0-9-]`; the JSON emitter assumes `$PLUGIN_ROOT` has no `"`/`\`
  (note inline).
