---
id: gate-patterns-inconsistency-lifecycle-status
kind: story
stage: done
tags: [refactor]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: patterns
created: 2026-07-12
updated: 2026-07-12
---

# Centralize lifecycle terminal-state definitions

## Existing pattern
`lifecycle-status-dictionary`

## Bundle divergence

The terminal sets are re-spelled in `internal/app/jobs.go` for persisted and public statuses and in `internal/jobs/store.go` for guarded persistence.

## Reconciliation direction

Establish one authoritative persisted lifecycle type/registry at the lowest dependency layer and derive terminal membership and public mappings from it. Preserve unknown-state conservatism and all race semantics.

## Design

- Define typed lifecycle constants and terminal membership in `internal/jobs`, the persistence-owning package.
- Replace raw lifecycle literals in `jobs.Store`, `internal/app` mappings, cancellation, and child completion with constants/helpers.
- Keep `result.Status` as the public contract and centralize only persisted↔public conversion in application code.
- Preserve unknown-as-running/non-terminal behavior and terminal race winners.
- Extend mapping/guard tests and run race tests over jobs/app/MCP/CLI.

## Implementation notes
- Execution capability: inline/high reasoning; this is a cohesive lifecycle refactor and delegation was explicitly disallowed.
- Review weight: standard (project default); stop at review as requested for independent follow-up review.
- Files changed: `internal/jobs/store.go`, `internal/jobs/status_test.go`, `internal/jobs/store_test.go`, `internal/app/jobs.go`, `internal/app/jobs_test.go`, `internal/app/cancel.go`, `internal/app/cancel_test.go`.
- Tests added: authoritative persisted terminal-membership table; persisted-result mapping table.
- Discrepancies from design: `app.ResultStatusFromJob` and `app.IsTerminalJobStatus` retain their string-facing signatures for existing CLI/test compatibility, while converting to the typed `jobs.Status` internally; persisted `Job.Status` and guarded-save results are typed.
- Adjacent issues parked: none.
- Verification: focused tests, focused race tests, full tests, `go vet ./...`, and `go build ./...` all passed.

## Review

Approved by independent deep review of commit `8307e73` plus amendment `9a27e66`. Persisted JSON strings remain compatible; unknown states remain non-terminal/running; guarded cancellation/completion winners and dependency direction are unchanged. One residual MCP test literal and stale pattern line references were corrected. Independent focused race tests, vet, and build passed.
