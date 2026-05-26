---
id: epic-result-contract
kind: epic
stage: drafting
tags: [infra, docs]
parent: null
depends_on: [epic-wrapper-cli]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Result Contract

## Brief

This epic makes the handoff back to Claude reliable. The wrapper emits a concise result with status, summary, changed files, verification outcomes, useful failure details, and metadata. Claude can read the result and continue the user conversation without scraping noisy process output.

The capability delivered here includes both human-readable and JSON output shapes, stable exit codes, failure reporting, and log excerpt handling. It also defines how partial edits and verification failures are surfaced.

This epic does not implement async job persistence. Async consumes this result shape after the blocking result contract exists.

## Foundation References

- `docs/VISION.md` — compact result criteria.
- `docs/SPEC.md` — output requirements.
- `docs/ARCHITECTURE.md` — wrapper role and blocking flow.
- `docs/CONTRACT.md` — result shape, exit codes, and failure reporting.

## Anticipated Child Features

- Human-readable result formatter.
- JSON result formatter.
- Exit-code mapping.
- Changed-file and verification summary capture.
- Failure detail and log excerpt reporting.

<!-- The design pass on each child feature will fill in real specifics. -->

