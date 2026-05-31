---
id: committed-binary-distribution-shim
kind: story
stage: done
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

## Implementation notes

Replaced the entire `bin/peeragent` shim. Removed: manifest-version lookup,
`~/.cache/peeragent` cache directory, `curl`/`tar` download block, and env vars
`PEERAGENT_VERSION`, `PEERAGENT_CACHE_DIR`, `PEERAGENT_RELEASE_BASE`,
`PEERAGENT_SKIP_DOWNLOAD`. Also removed the `.git`-directory gate that previously
restricted `go run` to source checkouts — step 3 now fires whenever `cmd/peeragent`
exists and `go` is on PATH, regardless of `.git`.

Resolution order matches spec exactly (PEERAGENT_BIN → dist/ → go run → committed
bin/$TARGET/ → contract JSON exit 3).

### Verification results

1. `sh -n bin/peeragent` — passes, no syntax errors.
2. `shellcheck` — not installed on this system; skipped.
3. `bin/peeragent --help` — exercises step 3 (go run); prints full usage text, exits 0.
   External agents had uncommitted edits to `cmd/peeragent/main.go` but the build
   compiled cleanly; no external breakage observed.
4. Not-installed JSON path (isolated shim, `PEERAGENT_TARGET_OVERRIDE=plan9-sparc`):
   emits `{"status":"failed",...,"metadata":{"exit_code":3,"target":"plan9-sparc"}}`,
   exits 3.
5. Not-installed `--text` path: writes human-readable message to stderr only
   (stdout empty), exits 3.
