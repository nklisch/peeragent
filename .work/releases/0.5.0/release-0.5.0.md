---
id: release-0.5.0
kind: release
stage: released
tags: []
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# Release 0.5.0

## Bound items

- `epic-async-jobs` — Async Jobs
- `epic-mcp-server` — MCP server support
- `epic-packaging-docs` — Packaging And Documentation
- `epic-plugin-foundation` — Plugin Foundation
- `epic-result-contract` — Result Contract
- `epic-safety-permissions` — Safety And Permissions
- `epic-wrapper-cli` — Wrapper CLI
- `committed-binary-distribution` — Committed Binary Distribution
- `epic-async-jobs-cancel` — Async Cancel
- `epic-async-jobs-launch` — Async Launch
- `epic-async-jobs-status-result` — Async Status And Result
- `epic-async-jobs-store` — Async Job Store
- `epic-mcp-server-delegation` — MCP delegation server
- `epic-mcp-server-job-control` — MCP async job control
- `epic-mcp-server-plugin-distribution` — MCP plugin distribution
- `epic-packaging-docs-build-artifacts` — Build Artifacts
- `epic-packaging-docs-user-guide` — User Guide
- `epic-packaging-docs-validation` — Validation
- `epic-plugin-foundation-entrypoint` — Wrapper Entrypoint
- `epic-plugin-foundation-go-skeleton` — Go Skeleton
- `epic-plugin-foundation-manifest` — Plugin Manifest
- `epic-plugin-foundation-skill` — Codex Implement Skill
- `epic-result-contract-execution-details` — Execution Detail Mapping
- `epic-result-contract-formatters` — Result Formatters
- `epic-result-contract-model` — Result Model
- `epic-safety-permissions-defaults` — Default Permission Flags
- `epic-safety-permissions-full-access` — Full Access Opt-In
- `epic-safety-permissions-profile-reporting` — Profile And Access Reporting
- `epic-safety-permissions-worktree` — Worktree Opt-In
- `epic-wrapper-cli-blocking-exec` — Blocking Codex Exec
- `epic-wrapper-cli-inputs` — Wrapper Inputs
- `epic-wrapper-cli-prompt` — Prompt Construction
- `async-job-robustness-job-source-of-truth` — Job is the source of truth for the async child
- `async-job-robustness-process-lifecycle` — PID sidecar, Setsid, group cancel, terminal guards
- `async-job-robustness-stdin-gate` — stdin gate + --job-run allow-list
- `committed-binary-distribution-ci-refresh` — CI build-and-commit workflow
- `committed-binary-distribution-docs-skills` — Skills and foundation docs
- `committed-binary-distribution-packaging` — Packaging preserves committed binaries
- `committed-binary-distribution-shim` — Shim resolution rewrite
- `committed-binary-distribution-validation` — Validation smokes
- `epic-mcp-server-delegation-application-services` — Extract delegation application services
- `epic-mcp-server-delegation-stdio-server` — Add stdio MCP delegate tool
- `epic-mcp-server-job-control-application-services` — Extract async job-control services
- `epic-mcp-server-job-control-tools` — Add MCP async job tools
- `epic-mcp-server-plugin-distribution-config` — Bundle and validate plugin MCP configuration
- `epic-mcp-server-plugin-distribution-guidance` — Document MCP use and skill integration
- `story-fix-gemini-sandbox-false-positive` — Gemini delegation returns a false auth/timeout failure caused by the default --sandbox flag
- `story-fix-test-helper-main-inheritance` — Prevent inherited test-helper state from launching peeragent during go test
- `async-job-robustness` — Async Job Robustness — late-bound archived stub; archived_atop: —

## Gate runs

- **gate-security** (2026-07-12) — 8 findings (0 critical, 0 high, 1 medium, 7 low); emitted 1 active remediation story and 7 backlog items.

- **gate-tests** (2026-07-12) — 4 coverage gaps (0 critical, 1 high, 2 medium, 1 low); emitted 3 active remediation stories and 1 backlog item; no tautological tests found.

- **gate-cruft** (2026-07-12) — 2 findings (1 high confidence, 1 medium confidence, 0 low); emitted 2 active cleanup stories.

- **gate-docs** (2026-07-12) — 1 medium-confidence README staleness finding; emitted 1 active documentation story; no foundation assertion, skill, pattern, generated-file, or changelog drift.

- **gate-patterns** (2026-07-12) — 5 structural patterns codified and indexed; 3 inconsistencies emitted as active refactor stories; generated rules digest and Claude compatibility symlink created.

All configured gates have run.

## Gate disposition

All 10 active gate-produced remediation stories reached `stage: done` after implementation and independent review. Low-severity findings remain explicitly parked in `.work/backlog/` for later prioritization.

## Final validation

- All 60 bound non-release items are at `stage: done`.
- `scripts/validate.sh` passed on 2026-07-12, including full Go tests, build, plugin packaging, MCP protocol smoke, strict Claude plugin validation, committed binaries, release artifact generation, metadata checks, documentation examples, and shim smokes.
- Focused race verification passed for jobs, application services, MCP, agent registry, and input normalization.
- `go vet ./...` and `go build ./...` passed.
- Release mapping: tag-based (`v0.5.0`).

## Shipping

- Date shipped: 2026-07-12
- Mapping: tag-based
- Items shipped: 60
- Gate findings: security 8, tests 4, cruft 2, docs 1, patterns 5 plus 3 pattern inconsistencies
- Publication: GitHub release `v0.5.0` with checksums and four Linux/macOS archives
- Workflow: https://github.com/nklisch/peeragent/actions/runs/29213889788
- Release: https://github.com/nklisch/peeragent/releases/tag/v0.5.0


## Shipped items

Bodies live in git history. Use `git show <git-ref>:<former-path>` for rows with a recorded ref; the `v0.5.0` tag history also preserves the late-bound archived item.

| id | title | kind | archived_atop | git ref |
|----|-------|------|---------------|---------|
| `async-job-robustness` | Async Job Robustness | feature | — | `—` |
| `async-job-robustness-job-source-of-truth` | Job is the source of truth for the async child | story | — | `1e80c07` |
| `async-job-robustness-process-lifecycle` | PID sidecar, Setsid, group cancel, terminal guards | story | — | `1e80c07` |
| `async-job-robustness-stdin-gate` | stdin gate + --job-run allow-list | story | — | `1e80c07` |
| `committed-binary-distribution` | Committed Binary Distribution | feature | — | `1e80c07` |
| `committed-binary-distribution-ci-refresh` | CI build-and-commit workflow | story | — | `1e80c07` |
| `committed-binary-distribution-docs-skills` | Skills and foundation docs | story | — | `1e80c07` |
| `committed-binary-distribution-packaging` | Packaging preserves committed binaries | story | — | `1e80c07` |
| `committed-binary-distribution-shim` | Shim resolution rewrite | story | — | `1e80c07` |
| `committed-binary-distribution-validation` | Validation smokes | story | — | `1e80c07` |
| `epic-async-jobs` | Async Jobs | epic | — | `1e80c07` |
| `epic-async-jobs-cancel` | Async Cancel | feature | — | `1e80c07` |
| `epic-async-jobs-launch` | Async Launch | feature | — | `1e80c07` |
| `epic-async-jobs-status-result` | Async Status And Result | feature | — | `1e80c07` |
| `epic-async-jobs-store` | Async Job Store | feature | — | `1e80c07` |
| `epic-mcp-server` | MCP server support | epic | — | `1e80c07` |
| `epic-mcp-server-delegation` | MCP delegation server | feature | — | `1e80c07` |
| `epic-mcp-server-delegation-application-services` | Extract delegation application services | story | — | `1e80c07` |
| `epic-mcp-server-delegation-stdio-server` | Add stdio MCP delegate tool | story | — | `1e80c07` |
| `epic-mcp-server-job-control` | MCP async job control | feature | — | `1e80c07` |
| `epic-mcp-server-job-control-application-services` | Extract async job-control services | story | — | `1e80c07` |
| `epic-mcp-server-job-control-tools` | Add MCP async job tools | story | — | `1e80c07` |
| `epic-mcp-server-plugin-distribution` | MCP plugin distribution | feature | — | `1e80c07` |
| `epic-mcp-server-plugin-distribution-config` | Bundle and validate plugin MCP configuration | story | — | `1e80c07` |
| `epic-mcp-server-plugin-distribution-guidance` | Document MCP use and skill integration | story | — | `1e80c07` |
| `epic-packaging-docs` | Packaging And Documentation | epic | — | `1e80c07` |
| `epic-packaging-docs-build-artifacts` | Build Artifacts | feature | — | `1e80c07` |
| `epic-packaging-docs-user-guide` | User Guide | feature | — | `1e80c07` |
| `epic-packaging-docs-validation` | Validation | feature | — | `1e80c07` |
| `epic-plugin-foundation` | Plugin Foundation | epic | — | `1e80c07` |
| `epic-plugin-foundation-entrypoint` | Wrapper Entrypoint | feature | — | `1e80c07` |
| `epic-plugin-foundation-go-skeleton` | Go Skeleton | feature | — | `1e80c07` |
| `epic-plugin-foundation-manifest` | Plugin Manifest | feature | — | `1e80c07` |
| `epic-plugin-foundation-skill` | Codex Implement Skill | feature | — | `1e80c07` |
| `epic-result-contract` | Result Contract | epic | — | `1e80c07` |
| `epic-result-contract-execution-details` | Execution Detail Mapping | feature | — | `1e80c07` |
| `epic-result-contract-formatters` | Result Formatters | feature | — | `1e80c07` |
| `epic-result-contract-model` | Result Model | feature | — | `1e80c07` |
| `epic-safety-permissions` | Safety And Permissions | epic | — | `1e80c07` |
| `epic-safety-permissions-defaults` | Default Permission Flags | feature | — | `1e80c07` |
| `epic-safety-permissions-full-access` | Full Access Opt-In | feature | — | `1e80c07` |
| `epic-safety-permissions-profile-reporting` | Profile And Access Reporting | feature | — | `1e80c07` |
| `epic-safety-permissions-worktree` | Worktree Opt-In | feature | — | `1e80c07` |
| `epic-wrapper-cli` | Wrapper CLI | epic | — | `1e80c07` |
| `epic-wrapper-cli-blocking-exec` | Blocking Codex Exec | feature | — | `1e80c07` |
| `epic-wrapper-cli-inputs` | Wrapper Inputs | feature | — | `1e80c07` |
| `epic-wrapper-cli-prompt` | Prompt Construction | feature | — | `1e80c07` |
| `gate-cruft-gemini-print-timeout-field` | Replace dead Gemini PrintTimeout option with a constant | story | — | `1e80c07` |
| `gate-cruft-unused-agent-display-name` | Remove unused CLI agentDisplayName helper | story | — | `1e80c07` |
| `gate-docs-pi-package-repository-shape` | Include the Pi package in repository-shape documentation | story | — | `1e80c07` |
| `gate-patterns-0.5.0` | Patterns extracted for 0.5.0 | story | — | `1e80c07` |
| `gate-patterns-inconsistency-agent-metadata` | Unify target agent metadata switches | story | — | `1e80c07` |
| `gate-patterns-inconsistency-lifecycle-status` | Centralize lifecycle terminal-state definitions | story | — | `1e80c07` |
| `gate-patterns-inconsistency-runner-test-double` | Consolidate duplicated target runner test support | story | — | `1e80c07` |
| `gate-security-job-id-path-traversal` | Reject path traversal in async job ids | story | — | `1e80c07` |
| `gate-tests-cancel-corrupt-result` | Cover cancellation with corrupt persisted result | story | — | `1e80c07` |
| `gate-tests-job-id-validation` | Verify job-id traversal and grammar rejection | story | — | `1e80c07` |
| `gate-tests-process-launch-cleanup` | Cover real launcher cleanup failures | story | — | `1e80c07` |
| `story-fix-gemini-sandbox-false-positive` | Gemini delegation returns a false auth/timeout failure caused by the default --sandbox flag | story | — | `1e80c07` |
| `story-fix-test-helper-main-inheritance` | Prevent inherited test-helper state from launching peeragent during go test | story | — | `1e80c07` |
