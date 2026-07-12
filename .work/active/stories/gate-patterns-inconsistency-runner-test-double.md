---
id: gate-patterns-inconsistency-runner-test-double
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

# Consolidate duplicated target runner test support

## Existing pattern
`runner-test-double-per-package`

## Bundle divergence

`recordingRunner`, `stubLookPath`, and runner capture logic are duplicated byte-for-byte in Codex, Claude, Gemini, and Z.AI adapter tests.

## Reconciliation direction

Introduce a focused shared test-support package or helper that preserves package-specific lookPath restoration and argv assertions without coupling production adapters. Keep behavior and test coverage unchanged.
