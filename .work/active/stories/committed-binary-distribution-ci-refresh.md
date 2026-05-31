---
id: committed-binary-distribution-ci-refresh
kind: story
stage: done
tags: [infra]
parent: committed-binary-distribution
depends_on: [committed-binary-distribution-packaging]
release_binding: null
gate_origin: null
created: 2026-05-31
updated: 2026-05-31
---

# CI build-and-commit workflow

## Scope

Implements Unit 3 of `committed-binary-distribution`. Add
`.github/workflows/build-binaries.yml` (modelled on
`../skills/.github/workflows/build-work-view.yml`) that cross-builds the four Go
targets and commits them back to `plugin/bin/<target>/peeragent`.

- One `ubuntu-latest` job (Go cross-compiles all four with `CGO_ENABLED=0`); no
  per-arch runner matrix.
- Triggers: push to `main` and `pull_request` on `cmd/**`, `internal/**`, `go.mod`,
  `go.sum`, `bin/peeragent`, and the workflow file; plus `workflow_dispatch`.
- Per-binary size guard (8 MB budget).
- Commit step gated `if: github.event_name != 'pull_request'`, with
  `permissions: contents: write`, `fetch-depth: 0`, `github-actions[bot]` identity, a
  `git diff --cached --quiet` no-op guard, and a `[skip ci]` commit message to avoid
  retriggering. See feature body Unit 3 for the full YAML.

## Acceptance Criteria

- [ ] Workflow cross-builds all four targets and enforces the size budget.
- [ ] Commit step is skipped on `pull_request`; otherwise commits with `[skip ci]`
      and pushes.
- [ ] The existing `actionlint` workflow passes on the new file.

## Notes

Depends on packaging having pinned the `plugin/bin/<target>/` layout and seeded the
initial binaries (so the first CI run diffs cleanly). Requires branch protection to
permit the bot push (or a PAT) — see feature Risks.

## Implementation notes

- Created `.github/workflows/build-binaries.yml` with exact YAML specified in the
  story brief. One `ubuntu-latest` job cross-compiles all four targets
  (`linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`) with
  `CGO_ENABLED=0` using `go build -trimpath -ldflags="-s -w"`.
- YAML well-formedness verified with:
  `python3 -c 'import yaml,sys; yaml.safe_load(open(".github/workflows/build-binaries.yml")); print("yaml ok")'`
  Result: `yaml ok`
- Acceptance criterion "The existing `actionlint` workflow passes on the new file"
  cannot be fulfilled — this repo has no `actionlint` workflow. Substituted the
  python3 YAML check above as the closest available validation. The workflow
  structure was also reviewed manually for correctness against the reference
  `build-work-view.yml` from the skills repo.
- Sanity checks confirmed:
  - Commit step gated `if: github.event_name != 'pull_request'`
  - Commit message includes `[skip ci]`
  - `permissions: contents: write` present at job level
  - `fetch-depth: 0` in checkout step
  - `git diff --cached --quiet` no-op guard present
  - `github-actions[bot]` identity configured before commit
