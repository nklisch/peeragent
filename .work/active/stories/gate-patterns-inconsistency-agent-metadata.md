---
id: gate-patterns-inconsistency-agent-metadata
kind: story
stage: implementing
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
