---
id: epic-result-contract
kind: epic
stage: done
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

## Design Decisions

- **Should the wrapper default output be human-readable text or JSON?** JSON by default. Claude is the primary consumer, so the default output should be structured and easy to parse. Human-readable output remains available through an explicit option.

## Decomposition

Split by result-surface responsibility. The result model defines the stable fields, the formatter owns JSON/text rendering, and execution detail capture maps Codex process outcomes into that model. Changed-file and verification extraction stays light in this epic because robust extraction needs real Codex behavior and can evolve after the blocking path is exercised.

### Child features

- `epic-result-contract-model` — shared result struct, statuses, metadata shape, and exit-code mapping — depends on: `[]`
- `epic-result-contract-formatters` — JSON default and explicit text formatter — depends on: `[epic-result-contract-model]`
- `epic-result-contract-execution-details` — map Codex stdout/stderr/exit/errors into result fields and concise failure details — depends on: `[epic-result-contract-formatters]`

### Decomposition risks

The main risk is overfitting the schema before real Codex output patterns are known. Keep the schema stable but conservative: status, summary, cwd/access/profile, exit code, stdout/stderr details, and placeholders for changed files and verification.

## Review

Approved. The result contract now has a shared model, JSON/text formatters, and execution-detail mapping. JSON is the default, text is explicit, errors preserve useful details, and metadata includes access, profile, and effort.

<!-- The design pass on each child feature will fill in real specifics. -->
