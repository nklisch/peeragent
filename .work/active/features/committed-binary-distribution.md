---
id: committed-binary-distribution
kind: feature
stage: implementing
tags: [infra, docs]
parent: null
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-31
updated: 2026-05-31
---

# Committed Binary Distribution

## Brief

Make the `peeragent` plugin runnable straight from a marketplace install with no
network fetch and no Go toolchain on the common developer platforms. Today
`bin/peeragent` resolves the wrapper by trying `$PEERAGENT_BIN`, then a
gitignored `dist/peeragent`, then `go run` in a source checkout, then a
`curl`-from-GitHub-releases download into `~/.cache/peeragent/`. The marketplace
artifact (`plugin/`, built by `scripts/package-plugin.sh`) ships no Go source and
no binary, so a marketplace install *always* hits that download path — that is the
friction we are removing.

This feature commits prebuilt Go binaries for the four common developer platforms
(linux amd64/arm64, darwin amd64/arm64) into the marketplace artifact at
`plugin/bin/<goos>-<goarch>/peeragent`, so an installed plugin is self-contained
for those platforms. `bin/peeragent` is rewritten to detect the host OS/arch, exec
the committed binary, and — when no committed binary matches — emit a contract-shaped
JSON failure (new exit code `3`) directing the user to install from GitHub releases
instead of downloading automatically. A CI workflow rebuilds and commits the four
binaries whenever the Go source changes, so the human release flow stays
source-only. The `peer` and `peer-review` skills are updated so the host agent
relays the install instruction to the user when peeragent is unavailable.

The wrapper stays written in Go (no rewrite); this is a packaging and distribution
change. The CLI's behavior, flags, result contract, and async-job machinery are
unchanged — `cmd/peeragent/` and the `internal/` packages are not touched.

## Strategic decisions

- **Rust rewrite vs. commit existing Go binaries**: Commit Go binaries — the
  no-download goal is met by committing prebuilt artifacts; a full Rust rewrite is
  out of scope and can be a separate future epic if smaller artifacts are wanted.
- **Which platforms ship committed binaries**: linux amd64/arm64 and darwin
  amd64/arm64 (four binaries) — the common Claude Code / Codex developer machines.
  Other platforms fall to the manual-install path.
- **How committed binaries are stored**: Committed directly in the repo tree (no
  git-lfs, no separate release ref). Accepted tradeoff: git history grows by the
  binary set whenever the Go source changes.
- **Fallback when no committed binary matches**: No automatic download. Detect the
  host platform; exec the committed binary if present; otherwise emit a clear
  install error. The host skills instruct the user how to install.

## Design decisions

- **Source-checkout dev path**: Keep `go run` (and the local `dist/peeragent`
  build) for source checkouts with Go, so developers run their own edits. Committed
  binaries serve marketplace / no-Go installs. — preserves the dev workflow without
  reintroducing a download.
- **Not-installed error format**: The shim emits a contract-shaped JSON failure
  (`status: failed`, `summary`/`details` with the GitHub releases URL) and a new
  dedicated exit code `3` ("wrapper binary unavailable"); `--text` prints the same
  guidance to stderr. — host agents surface it through the JSON path they already
  parse; `3` is distinguishable from the existing usage error (`2`).
- **CI builds and commits the binaries**: A `build-binaries.yml` workflow
  cross-builds the four targets and commits them back with `[skip ci]` on Go-source
  changes (modelled on `../skills/.github/workflows/build-work-view.yml`). — the
  human bump/release commit stays source-only; CI is the single binary producer
  using the pinned `go.mod` toolchain.
- **Single committed location**: Binaries live only at `plugin/bin/<target>/peeragent`
  (the marketplace artifact), not duplicated at repo root. — halves git bloat
  (~11 MB/refresh, four stripped binaries) and gives one source of truth. Source
  checkouts use `go run`; they do not need repo-root binaries.
- **No byte-for-byte drift check**: Freshness is procedural (CI rebuilds on source
  change); validation smokes that the committed host-arch binary *runs*, rather than
  byte-comparing against a fresh build. — cross-machine Go output is not guaranteed
  reproducible across toolchain patch versions, so a byte-compare would flake.
- **Platform escape hatch**: `PEERAGENT_TARGET_OVERRIDE=<goos>-<goarch>` overrides
  the detected platform string for the committed-binary path. — lets misdetected
  platforms point at a present binary and makes the not-installed branch testable.

## Foundation References

Rolled forward in place (rolling-foundation) when implemented:

- `docs/ARCHITECTURE.md` — "Overview" flow (`dist/peeragent or go run cmd/peeragent`)
  and "Wrapper Role" (the download/cache flow). New resolution order: override →
  local build → `go run` (source) → committed platform binary → install error.
- `docs/SPEC.md` — "Runtime Context" and "Components": committed per-platform
  binaries in the plugin artifact + supported-platform set; release tarballs remain
  the manual-install source for unsupported platforms.
- `docs/CONTRACT.md` — "Exit Codes": add `3: wrapper binary unavailable for this
  platform`.

The wrapper implementation (`cmd/peeragent/`, `internal/`) is unchanged. Only the
shim, scripts, CI, skills, docs, and the committed binaries change.

## Architectural choice

**Committed binaries in the marketplace artifact, produced by CI, resolved by the
shim.** Considered:

1. *Commit binaries at repo root and copy into `plugin/`.* Duplicates the bytes
   (repo-root + `plugin/`) and creates two freshness owners. Rejected.
2. *Build binaries locally in `bump.sh`/`package-plugin.sh`.* Forces every releaser
   to cross-compile and commit ~11 MB; couples packaging to a Go toolchain. Rejected
   in favor of CI.
3. *Chosen: single location `plugin/bin/<target>/peeragent`, CI-built and
   CI-committed.* `package-plugin.sh` regenerates the curated tree without wiping
   the binary subdirs; the CI workflow is the sole binary producer; the shim resolves
   the committed binary for the packaged (no-Go) tree. One source of truth, minimal
   bloat, no manual build step.

Target naming is `<goos>-<goarch>` (e.g. `linux-amd64`, `darwin-arm64`), matching
`scripts/release.sh`'s build-dir convention. Binaries are stripped/static
(`CGO_ENABLED=0 -trimpath -ldflags="-s -w"`), identical flags to the release tarballs.

## Implementation Units

### Unit 1: Shim resolution rewrite
**File**: `bin/peeragent`
**Story**: `committed-binary-distribution-shim`

Replace the manifest-version / cache / `curl` download block (current lines ~15-86)
with committed-binary resolution and a JSON install error.

```sh
#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PLUGIN_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
RELEASES_URL="https://github.com/nklisch/peeragent/releases"

# 1. Explicit override.
if [ -n "${PEERAGENT_BIN:-}" ] && [ -x "$PEERAGENT_BIN" ]; then
  exec "$PEERAGENT_BIN" "$@"
fi

# 2. Local ephemeral build (developer `make build`).
if [ -x "$PLUGIN_ROOT/dist/peeragent" ]; then
  exec "$PLUGIN_ROOT/dist/peeragent" "$@"
fi

# 3. Source checkout with Go: run current source so edits are reflected.
if [ -d "$PLUGIN_ROOT/cmd/peeragent" ] && command -v go >/dev/null 2>&1; then
  exec go run "$PLUGIN_ROOT/cmd/peeragent" "$@"
fi

# 4. Committed platform binary (marketplace / no-Go install).
detect_target() {
  if [ -n "${PEERAGENT_TARGET_OVERRIDE:-}" ]; then printf '%s' "$PEERAGENT_TARGET_OVERRIDE"; return; fi
  case "$(uname -s 2>/dev/null || echo unknown)" in
    Linux) goos=linux ;; Darwin) goos=darwin ;; *) goos=unknown ;;
  esac
  case "$(uname -m 2>/dev/null || echo unknown)" in
    x86_64|amd64) goarch=amd64 ;; arm64|aarch64) goarch=arm64 ;; *) goarch=unknown ;;
  esac
  printf '%s-%s' "$goos" "$goarch"
}
TARGET=$(detect_target)
CANDIDATE="$PLUGIN_ROOT/bin/$TARGET/peeragent"
if [ -x "$CANDIDATE" ]; then
  exec "$CANDIDATE" "$@"
fi

# 5. Not installed for this platform — fail per the result contract (exit 3).
wants_text() { for a in "$@"; do [ "$a" = "--text" ] && return 0; done; return 1; }
if wants_text "$@"; then
  printf 'peeragent is not installed for this platform (%s).\nDownload the matching asset from %s, then set PEERAGENT_BIN to its path\nor place the binary at %s.\n' \
    "$TARGET" "$RELEASES_URL" "$CANDIDATE" >&2
else
  printf '{"status":"failed","summary":"peeragent is not installed for %s","changed_files":[],"verification":[],"details":"No committed peeragent binary for %s. Download the matching asset from %s, then set PEERAGENT_BIN to its path or place it at %s.","metadata":{"exit_code":3,"target":"%s"}}\n' \
    "$TARGET" "$TARGET" "$RELEASES_URL" "$CANDIDATE" "$TARGET"
fi
exit 3
```

**Implementation Notes**:
- JSON goes to stdout (host parsers read stdout, matching the Go binary); the
  `--text` variant goes to stderr.
- The JSON emitter assumes `$PLUGIN_ROOT` contains no `"`/`\` (pathological on unix);
  `$TARGET` is a safe `[a-z0-9-]` string. Acceptable; note it inline.
- All download/cache/version-manifest logic and the `PEERAGENT_VERSION`,
  `PEERAGENT_CACHE_DIR`, `PEERAGENT_RELEASE_BASE`, `PEERAGENT_SKIP_DOWNLOAD` env
  vars are removed.

**Acceptance Criteria**:
- [ ] Packaged shim (`plugin/bin/peeragent`, no `cmd/`) execs the committed
      host-arch binary and `--status missing-job` exits `4` with contract JSON.
- [ ] `PEERAGENT_TARGET_OVERRIDE=plan9-sparc plugin/bin/peeragent --agent codex x`
      exits `3` with `{"status":"failed",...}` JSON containing the releases URL.
- [ ] `--text` produces a human-readable install message on stderr and exits `3`.
- [ ] In a source checkout with Go, the shim still uses `go run` (no committed
      binary required for the dev loop).
- [ ] No `curl`, cache, or release-download code remains in `bin/peeragent`.

---

### Unit 2: Packaging preserves committed binaries
**File**: `scripts/package-plugin.sh`
**Story**: `committed-binary-distribution-packaging`

`package-plugin.sh` currently does `rm -rf "$PLUGIN"` and regenerates everything,
which would wipe the CI-committed binaries. Change it to regenerate only the curated
parts and leave `plugin/bin/<target>/` intact.

```sh
# Regenerate curated tree WITHOUT removing committed platform binaries.
rm -rf "$PLUGIN/.claude-plugin" "$PLUGIN/.codex-plugin" "$PLUGIN/skills"
rm -f  "$PLUGIN/bin/peeragent"
mkdir -p "$PLUGIN/.claude-plugin" "$PLUGIN/.codex-plugin" "$PLUGIN/bin" "$PLUGIN/skills"
# ... existing manifest + shim + skills copies ...
# plugin/bin/<target>/peeragent dirs are owned by CI (build-binaries.yml); untouched.
```

Also: seed the initial four binaries once locally and commit them (so the artifact
is self-contained before CI's first run), and update the `scripts/release.sh` GitHub
release note string from "Marketplace installs can fetch these assets on first use."
to a manual-install description. Confirm `scripts/bump.sh` stays source-only and that
its `git add` (switch the explicit `plugin/*` lines to `git add plugin`) captures the
regenerated curated tree without disturbing the committed binaries.

**Implementation Notes**:
- Seed binaries with the same flags CI uses:
  `CGO_ENABLED=0 GOOS=<os> GOARCH=<arch> go build -trimpath -ldflags="-s -w" -o plugin/bin/<target>/peeragent ./cmd/peeragent`.
- `bump.sh` must not build binaries; a pure version bump does not change them.

**Acceptance Criteria**:
- [ ] After `scripts/package-plugin.sh`, `plugin/bin/<target>/peeragent` for all four
      targets are still present and executable.
- [ ] Initial four committed binaries exist and are tracked by git.
- [ ] `scripts/release.sh` release note no longer claims first-use download.
- [ ] `bin/peeragent` shim is regenerated into `plugin/bin/peeragent`.

---

### Unit 3: CI build-and-commit workflow
**File**: `.github/workflows/build-binaries.yml`
**Story**: `committed-binary-distribution-ci-refresh`

```yaml
name: Build committed binaries

on:
  push:
    branches: [main]
    paths: ["cmd/**", "internal/**", "go.mod", "go.sum", "bin/peeragent",
            ".github/workflows/build-binaries.yml"]
  pull_request:
    paths: ["cmd/**", "internal/**", "go.mod", "go.sum", "bin/peeragent",
            ".github/workflows/build-binaries.yml"]
  workflow_dispatch:

env:
  SIZE_BUDGET_BYTES: 8388608   # 8 MB per binary

jobs:
  build-binaries:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version-file: go.mod }
      - name: Cross-build committed binaries
        run: |
          set -eu
          for pair in "linux amd64" "linux arm64" "darwin amd64" "darwin arm64"; do
            set -- $pair; goos=$1; goarch=$2; target="$goos-$goarch"
            dest="plugin/bin/$target"; mkdir -p "$dest"
            CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
              go build -trimpath -ldflags="-s -w" -o "$dest/peeragent" ./cmd/peeragent
            size=$(stat -c '%s' "$dest/peeragent"); echo "$target: $size bytes"
            [ "$size" -le "$SIZE_BUDGET_BYTES" ] || { echo "ERROR: $target over budget ($size)"; exit 1; }
          done
      - name: Commit refreshed binaries
        if: github.event_name != 'pull_request'
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add plugin/bin
          if git diff --cached --quiet; then
            echo "No binary changes to commit."
          else
            git commit -m "chore: refresh committed peeragent binaries [skip ci]"
            git push
          fi
```

**Implementation Notes**:
- PRs build + size-guard (verify buildability) but do not commit. Push to `main`
  and `workflow_dispatch` commit + push with `[skip ci]` to avoid retriggering.
- Requires branch protection to permit the `github-actions[bot]` push (or a PAT) —
  flag in Risks.

**Acceptance Criteria**:
- [ ] Workflow cross-builds all four targets and enforces the per-binary size budget.
- [ ] Commit step is skipped on `pull_request` and uses `[skip ci]` otherwise.
- [ ] `actionlint` (existing GH-actions lint) passes on the new workflow.

---

### Unit 4: Validation smokes
**File**: `scripts/validate.sh`
**Story**: `committed-binary-distribution-validation`

Add a step after "plugin package" that asserts the four committed binaries exist and
exercises both shim branches through the packaged shim.

```sh
step "committed platform binaries"
for t in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64; do
  test -x "plugin/bin/$t/peeragent" && test -s "plugin/bin/$t/peeragent"
done

# committed-binary resolution (host arch on CI = linux-amd64), via the packaged shim
set +e; out=$(plugin/bin/peeragent --status missing-job 2>&1); code=$?; set -e
[ "$code" -eq 4 ] || { echo "committed smoke expected 4 got $code"; echo "$out"; exit 1; }
printf '%s\n' "$out" | grep -q '"status":"failed"'
printf '%s\n' "$out" | grep -q '"exit_code":4'

# not-installed path via target override
set +e; out3=$(PEERAGENT_TARGET_OVERRIDE=plan9-sparc plugin/bin/peeragent --agent codex x 2>&1); code3=$?; set -e
[ "$code3" -eq 3 ] || { echo "not-installed smoke expected 3 got $code3"; echo "$out3"; exit 1; }
printf '%s\n' "$out3" | grep -q '"exit_code":3'
printf '%s\n' "$out3" | grep -q 'releases'
```

**Implementation Notes**:
- Keep the existing repo-root `shim smoke` (`bin/peeragent --status missing-job`),
  which exercises the `go run` path in CI.
- Do not byte-compare binaries (per Design decisions).

**Acceptance Criteria**:
- [ ] `scripts/validate.sh` passes with the four committed binaries present.
- [ ] The not-installed smoke asserts exit `3` and the releases URL.

---

### Unit 5: Skills + foundation docs
**Files**: `skills/peer/SKILL.md`, `skills/peer-review/SKILL.md`,
`docs/ARCHITECTURE.md`, `docs/SPEC.md`, `docs/CONTRACT.md`, `README.md`
**Story**: `committed-binary-distribution-docs-skills`

- **Skills**: in the Result Handling / Guardrails sections of both `peer` and
  `peer-review`, add a "not installed" case: if the wrapper reports `status: failed`
  with exit code `3` (peeragent not installed for this platform), tell the user to
  install peeragent from the GitHub releases page for their OS/arch — set
  `PEERAGENT_BIN` or place the binary at `<plugin>/bin/<goos>-<goarch>/peeragent`,
  and mention `PEERAGENT_TARGET_OVERRIDE` for a misdetected platform. Do not retry
  in a loop.
- **Foundation docs**: roll `ARCHITECTURE.md` (overview + Wrapper Role resolution
  order), `SPEC.md` (runtime context + components), and `CONTRACT.md` (add exit
  code `3`) forward in place; remove download/cache references.
- **README**: install section — supported platforms run with no download/build;
  other platforms install from releases (with `PEERAGENT_BIN` / target-override
  guidance). Remove any stale download env-var docs. Keep the `make build`,
  marketplace-add, and `make release` examples that `validate.sh` greps for.

**Acceptance Criteria**:
- [ ] `peer` and `peer-review` describe the not-installed handling and the install URL.
- [ ] `CONTRACT.md` documents exit code `3`; `ARCHITECTURE.md`/`SPEC.md` describe the
      committed-binary model with no download/cache language.
- [ ] `scripts/validate.sh` documentation-examples step still passes.

## Implementation Order

1. `committed-binary-distribution-shim` (Unit 1) — resolution + exit-3 contract.
2. `committed-binary-distribution-packaging` (Unit 2) — preserve binaries, seed
   initial four, release-note/bump tweaks. (Parallel with 1.)
3. `committed-binary-distribution-ci-refresh` (Unit 3) — after packaging pins the
   `plugin/bin/<target>/` layout.
4. `committed-binary-distribution-validation` (Unit 4) — after shim + packaging.
5. `committed-binary-distribution-docs-skills` (Unit 5) — after shim + packaging.

## Testing

Shell/integration via `scripts/validate.sh` (no Go code changes, so no `go test`
changes). Smokes: committed-binary resolution (exit 4), not-installed branch (exit
3 + releases URL), four binaries present/executable, existing `go run` shim smoke,
plugin-package and release-artifact steps. The CI workflow is exercised by PR builds
(buildability + size guard) and validated statically by the existing `actionlint`
workflow.

## Risks

- **Git history bloat** — four stripped binaries (~11 MB) re-committed by CI on each
  Go-source change. Accepted; mitigated by stripping, the four-platform cap, and the
  8 MB-per-binary size guard.
- **CI push permissions** — the commit-back job needs `contents: write` and branch
  protection that allows the `github-actions[bot]` push (or a PAT). If blocked, the
  binaries silently stop refreshing; the buildability check still runs.
- **Stale binaries between source change and CI commit** (and on open PRs) — the
  committed binary lags the working source until CI commits. Marketplace ships
  released tags, not bleeding-edge, so this is benign; validation only asserts the
  binary *runs*, not that it matches HEAD.
- **No embedded version** — committed binaries carry no version string, so freshness
  is purely procedural (CI rebuilds on source change; off-path manual tags could ship
  a stale binary). Residual; documented, not mitigated in this feature.
- **Source checkout without Go** — has neither `go run` nor repo-root committed
  binaries, so it hits the exit-3 install error. Edge case (developers have Go); the
  no-Go path is the marketplace install, which carries the binaries.
