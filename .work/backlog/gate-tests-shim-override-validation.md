---
id: gate-tests-shim-override-validation
kind: story
tags: [testing]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: tests
created: 2026-07-12
updated: 2026-07-12
---

# Cover malformed platform override tokens

## Priority
Low

## Spec reference
Item: `committed-binary-distribution-shim`
Acceptance criterion: override requires a dash and a sanitized token.

## Gap type
Missing shell-boundary tests for malformed `PEERAGENT_TARGET_OVERRIDE` values.

## Suggested test
Extend `scripts/validate.sh` to verify underscore, no-dash, traversal, and uppercase overrides are rejected or fall back safely without influencing candidate paths.

## Test location (suggested)
`scripts/validate.sh`
