---
id: gate-patterns-inconsistency-runner-test-double
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

# Consolidate duplicated target runner test support

## Existing pattern
`runner-test-double-per-package`

## Bundle divergence

`recordingRunner`, `stubLookPath`, and runner capture logic are duplicated byte-for-byte in Codex, Claude, Gemini, and Z.AI adapter tests.

## Reconciliation direction

Introduce a focused shared test-support package or helper that preserves package-specific lookPath restoration and argv assertions without coupling production adapters. Keep behavior and test coverage unchanged.

## Design

- Add an internal test-support package exposing a recording `executil.Runner`; keep each adapter's package-local `lookPath` assignment/restoration helper because unexported variables cannot be set externally.
- Replace four duplicate runner structs/methods with the shared double while retaining local argv assertions and cleanup.
- Do not move production code or target-specific fixtures.
- Run all target adapter tests and the full suite.

## Implementation Notes
- Added `internal/testsupport.RecordingRunner` as the shared offline `executil.Runner` test double.
- Updated Codex, Claude, Gemini, and Z.AI adapter tests to use the shared recorder and exported capture fields.
- Retained each adapter's package-local `stubLookPath` and cleanup restoration because `lookPath` remains package-private.
- Left production adapters, target fixtures, and argv assertions in their existing packages.

## Verification
- `go test ./internal/claude ./internal/codex ./internal/gemini ./internal/zai`

## Review

Approved by independent source review of commit `9e316ff`. The shared double satisfies `executil.Runner`, is imported only from test files, preserves local `lookPath` cleanup and every target assertion, and passed all adapter tests.
