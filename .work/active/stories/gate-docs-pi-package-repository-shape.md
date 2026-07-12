---
id: gate-docs-pi-package-repository-shape
kind: story
stage: review
tags: [documentation]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: docs
created: 2026-07-12
updated: 2026-07-12
---

# Include the Pi package in repository-shape documentation

## Drift category
readme-staleness

## Confidence
Medium

## Location
- Doc: `README.md:405`
- Code: `package.json:1`

## Current doc text
> This repo is shaped as both a Claude Code marketplace and a Codex marketplace.

## Reality

The repository is also a first-class Pi package. Root `package.json` declares `@nklisch/pi-peeragent` and loads `./plugin/skills`; validation and version bumping treat it as a peer manifest, and README installation guidance already documents `pi install`.

## Required edit

Roll current truth into the Repository Shape section and file listing, and add the Pi package manifest to the Components description in `docs/SPEC.md` where appropriate. Do not add historical prose.

## Design

- Update README's current repository-shape sentence to name Claude, Codex, and Pi.
- Add `package.json` to the repository-shape tree with its Pi skills role.
- Add the Pi package manifest to `docs/SPEC.md` Components.
- Preserve rolling-foundation wording and run documentation validation.

## Implementation Notes
- Updated README Repository Shape to identify Claude, Codex, and Pi packaging.
- Added the root `package.json` Pi manifest to the README file tree.
- Added the Pi package manifest and its `./plugin/skills` entrypoint to the SPEC Components list.

## Verification
- `git diff --check -- README.md docs/SPEC.md`
- `scripts/validate.sh` reached and passed the documentation examples checks, then failed at the pre-existing shim smoke expectation (`--status missing-job` returns exit 2 after job-id validation, while the script expects exit 4).
