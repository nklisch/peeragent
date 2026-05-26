---
id: epic-packaging-docs
kind: epic
stage: done
tags: [docs, infra]
parent: null
depends_on: [epic-plugin-foundation, epic-wrapper-cli, epic-result-contract]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Packaging And Documentation

## Brief

This epic prepares Codex Implement for practical use as a Claude Code plugin. It covers packaging polish, installation guidance, usage examples, command documentation, and release readiness checks.

The capability delivered here is the user-facing finish around the core implementation path. A developer can install the plugin, understand the default safety posture, invoke `codex-implement`, inspect results, and know when to choose full-access or async modes.

This epic does not add new implementation modes beyond what the functional epics provide. It documents, packages, and validates the product surface that already exists.

## Foundation References

- `docs/VISION.md` — user and product expectations.
- `docs/SPEC.md` — components, non-goals, and runtime assumptions.
- `docs/ARCHITECTURE.md` — plugin layout and extension boundaries.
- `docs/CONTRACT.md` — CLI contract and output behavior.

## Design Decisions

- **Should distributability be represented as docs only or an actual local artifact path?** Use an actual artifact path — the existing shim already prefers `dist/codex-implement`, so the packaging work should make that path repeatable instead of relying on `go run`.
- **Should validation require a live Codex implementation run?** No — keep release-readiness checks cheap and local by validating tests, build output, shim behavior, manifest shape, and documented command examples.

## Decomposition

Split by user-facing finish: build artifacts make the plugin distributable, user guide documents the surface, and validation checks that the documented distributable state still works.

### Child features

- `epic-packaging-docs-build-artifacts` — repeatable local build output and executable shim packaging — depends on: `[]`
- `epic-packaging-docs-user-guide` — install, usage, effort, async, full-access, and troubleshooting docs — depends on: `[epic-packaging-docs-build-artifacts]`
- `epic-packaging-docs-validation` — release-readiness checks for tests, build, shim, manifest, and examples — depends on: `[epic-packaging-docs-build-artifacts, epic-packaging-docs-user-guide]`

### Decomposition risks

Documentation can easily drift from the actual wrapper flags. The validation feature should prefer runnable command examples and manifest checks over a prose-only checklist.

## Review

Done. The plugin now has a repeatable build path, human and Claude-facing usage docs, contract cleanup, and an executable local validation command. Final validation for this epic passed with `make validate`.

<!-- The design pass on each child feature will fill in real specifics. -->
