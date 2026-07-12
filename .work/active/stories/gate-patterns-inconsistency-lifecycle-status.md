---
id: gate-patterns-inconsistency-lifecycle-status
kind: story
stage: drafting
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
