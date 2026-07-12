---
id: gate-patterns-inconsistency-agent-metadata
kind: story
stage: done
tags: [refactor]
parent: null
depends_on: [gate-cruft-unused-agent-display-name]
release_binding: 0.5.0
gate_origin: patterns
created: 2026-07-12
updated: 2026-07-12
---

# Unify target agent metadata switches

## Existing pattern
`target-executor-adapter`

## Bundle divergence

Agent prompt names and display names are separately switched in `internal/app/execute.go` and `internal/app/service.go`; the dead CLI copy is handled by `gate-cruft-unused-agent-display-name`.

## Reconciliation direction

Create one authoritative typed target registry for canonical id, prompt identity, display name, and target adapter dispatch metadata. Derive validation/routing/display behavior where practical without changing observable CLI or MCP behavior.

## Design

- Add an `internal/agent` registry at the lowest dependency layer with canonical ids and aliases plus prompt/display metadata.
- Derive `input.normalizeAgent`, application prompt/display names, and agent-id validation from the registry; keep executor calls in `internal/app` to avoid dependency inversion.
- Preserve every accepted alias, user-facing name, default, and error message.
- Add registry/normalization parity tests and run full/race verification.
- Keep the change behavior-preserving; do not add targets or alter CLI/MCP schemas.

## Implementation notes
- Execution capability: inline/high reasoning; this is a cohesive metadata reconciliation and delegation was explicitly disallowed.
- Review weight: standard (project default); stop at review as requested for independent follow-up review.
- Files changed: `internal/agent/registry.go`, `internal/agent/registry_test.go`, `internal/input/input.go`, `internal/input/input_test.go`, `internal/app/execute.go`, `internal/app/service.go`, `internal/app/agent_metadata_test.go`.
- Tests added: registry metadata/alias table, input normalization parity table, exact unsupported-agent error assertion, and application prompt/display parity table.
- Discrepancies from design: canonical target IDs serve as the registry's adapter dispatch metadata, avoiding a second duplicated adapter-key field; executor adapter calls remain explicitly in `internal/app`.
- Adjacent issues parked: none.
- Verification: focused tests, focused race tests, full tests, `go vet ./...`, and `go build ./...` all passed.

## Review

Approved by independent deep review of commit `289ac21` with direct parent verification of `execute.go`, `service.go`, and application metadata parity tests. All aliases/defaults, exact validation text, prompt identities, display names, target dispatch, and dependency direction remain unchanged. Independent focused race tests, vet, and build passed.
