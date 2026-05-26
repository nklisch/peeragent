# Project Conventions

## Release Mapping

tag-based

Releases are represented by version tags. Release items bind active work to a version immediately before quality gates and shipping.

## Tag Taxonomy

- security: permission boundaries, command safety, trust decisions, secrets, and supply-chain concerns.
- tests: unit, integration, fixture, and end-to-end verification coverage.
- infra: plugin packaging, CLI runtime, install behavior, and local toolchain setup.
- docs: foundation docs, user-facing usage docs, contracts, and migration notes.
- refactor: structural cleanup with no intended behavior change.
- perf: latency, throughput, process startup cost, and long-running job efficiency.

## Slug Conventions

Use kebab-case slugs. Child items use parent-prefix slugs when it improves uniqueness and scanability, for example `codex-implement-wrapper-cli`.

## Stage Overrides

None. Use the standard agile-workflow stages:

- epic: drafting -> implementing -> review -> done
- feature: drafting -> implementing -> review -> done
- story: implementing -> review -> done
- release: planned -> quality-gate -> released

## Gate Config

gates_for_release: [security, tests, cruft, docs, patterns]

