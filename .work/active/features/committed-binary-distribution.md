---
id: committed-binary-distribution
kind: feature
stage: drafting
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

Make the `peeragent` plugin runnable straight from a git checkout with no
network fetch and no Go toolchain on the common developer platforms. Today
`bin/peeragent` resolves the wrapper by trying `$PEERAGENT_BIN`, then a
gitignored `dist/peeragent`, then `go run` in a source checkout, then a
`curl`-from-GitHub-releases download into `~/.cache/peeragent/`. The download
path is the friction we are removing: a fresh plugin install on a machine
without Go silently reaches out to the network to fetch a release asset.

This feature commits prebuilt Go binaries for the four common developer
platforms (linux x86_64/arm64, darwin x86_64/arm64) directly into the repo so a
clone is self-contained for those platforms. `bin/peeragent` is rewritten to
detect the host OS/arch, exec the committed binary, and — when no committed
binary matches the platform — emit a clear, actionable error directing the user
to install from GitHub releases instead of attempting an automatic download. The
host-facing skills are updated so that when delegation fails because peeragent is
not available for the platform, the agent relays the install instruction to the
user rather than silently failing.

The wrapper stays written in Go (no rewrite); this is a packaging and
distribution change, not a reimplementation. The CLI's behavior, flags, result
contract, and async-job machinery are unchanged.

## Strategic decisions

- **Rust rewrite vs. commit existing Go binaries**: Commit Go binaries — the
  no-download goal is met by committing prebuilt artifacts; a full Rust rewrite
  (~3,800 LoC + tests + async-job machinery) is out of scope here and can be a
  separate future epic if smaller artifacts are wanted.
- **Which platforms ship committed binaries**: linux x86_64/arm64 and darwin
  x86_64/arm64 (four binaries) — covers the common Claude Code / Codex
  developer machines. Other platforms (Windows, etc.) fall to the manual-install
  path.
- **How committed binaries are stored**: Committed directly in the repo tree
  (no git-lfs, no separate release ref). A clone is fully self-contained for the
  supported platforms with zero extra tooling. Accepted tradeoff: git history
  grows by the committed binary set on each version bump.
- **Fallback behavior when no committed binary matches**: No automatic download.
  Detect whether a committed binary exists for the host platform; if it does,
  exec it; if it does not, emit a clear error telling the user to install from
  GitHub releases. The host-facing skills must instruct the user how to install
  when peeragent is unavailable.

## Foundation References

These foundation assertions describe the current (download-based) distribution
model and must be rolled forward in place when this feature is implemented:

- `docs/ARCHITECTURE.md` — "Overview" flow (`dist/peeragent or go run
  cmd/peeragent`) and "Wrapper Role" (invokes `dist/peeragent` when present,
  falls back to `go run`). The resolution order and the implicit download flow
  change to: committed platform binary first, dev `go run` for source
  checkouts, then a manual-install error.
- `docs/SPEC.md` — "Runtime Context" ("a platform-compatible `peeragent` wrapper
  binary supplied by the plugin") and "Components". Update to describe committed
  per-platform binaries and the supported-platform set; the GitHub release
  tarballs remain the manual-install source for unsupported platforms.

The wrapper implementation components (`cmd/peeragent/`, `internal/` Go
packages) are unchanged.

## Implementation surface (for feature-design)

Indicative, not prescriptive — `feature-design` owns the decomposition,
interfaces, and child stories.

- **Committed binary location and layout.** `dist/` is currently gitignored.
  Pick a committed location/naming for the four platform binaries (e.g.
  `bin/<os>-<arch>/peeragent` or a tracked `dist/<os>-<arch>/` path) and adjust
  `.gitignore` accordingly. The committed binary is always the manifest version
  it ships with, so no version-matching logic is needed on the committed path.
- **`bin/peeragent` rewrite.** New resolution order: `$PEERAGENT_BIN` override →
  committed platform binary → `go run` for source checkouts with Go (dev
  convenience) → clear GitHub-install error. Remove the `curl`/`tar` download
  block and the `~/.cache/peeragent/` cache logic.
- **Error message contract.** The "not installed for this platform" error should
  be unambiguous and point to the GitHub releases install instructions, so both
  a human at a terminal and a host agent reading the wrapper output can act on
  it.
- **Skill updates.** Update the host-facing skills (`peer`, `peer-review`, and
  the codex/claude/gemini implement skills) so the agent detects the
  not-available condition and instructs the user to install peeragent from
  GitHub releases, rather than silently failing the delegation.
- **Release / refresh process.** Committed binaries must be regenerated and
  committed on every version bump. Extend `scripts/build.sh` /
  `scripts/release.sh` / `scripts/bump.sh` and `.github/workflows/release.yml`
  to (cross-)build the four binaries into the committed location as part of the
  release flow. GitHub release tarballs stay as the manual-install source for
  unsupported platforms.
- **Binary size mitigation.** Consider stripping (`-ldflags="-s -w"`) to shrink
  the committed Go binaries (~3.9MB each) and reduce git history growth.

## Risks

- **Git history bloat.** Four committed binaries per version bump grows the repo
  permanently. Accepted by decision; mitigate with stripped builds and by
  keeping the supported-platform set to four.
- **Binary/source drift.** A committed binary built from a different commit than
  it ships with would be a silent correctness hazard. The release/refresh
  process must rebuild and commit the binaries atomically with the version bump,
  and validation should confirm the committed binary matches the current source.
- **Platform coverage gaps.** Windows and less-common arches rely entirely on the
  manual-install path and the skills' install guidance; verify that path is
  discoverable and the error is clear.

<!-- The design pass on this feature (`/agile-workflow:feature-design`) will fill
in interfaces, the committed-binary layout, the shim resolution logic, the skill
edits, the release-process changes, and child stories with depends_on chains. -->
