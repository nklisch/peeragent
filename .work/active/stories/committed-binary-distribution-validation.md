---
id: committed-binary-distribution-validation
kind: story
stage: implementing
tags: [infra]
parent: committed-binary-distribution
depends_on: [committed-binary-distribution-shim, committed-binary-distribution-packaging]
release_binding: null
gate_origin: null
created: 2026-05-31
updated: 2026-05-31
---

# Validation smokes

## Scope

Implements Unit 4 of `committed-binary-distribution`. Extend `scripts/validate.sh`
with a "committed platform binaries" step that runs after "plugin package":

- Assert all four `plugin/bin/<target>/peeragent` exist, are executable, and nonzero.
- Committed-binary resolution smoke via the packaged shim:
  `plugin/bin/peeragent --status missing-job` → exit `4`, contract JSON.
- Not-installed smoke:
  `PEERAGENT_TARGET_OVERRIDE=plan9-sparc plugin/bin/peeragent --agent codex x` →
  exit `3`, JSON with `"exit_code":3` and the releases URL.
- Keep the existing repo-root `shim smoke` (exercises the `go run` path in CI).

See feature body Unit 4. No byte-for-byte comparison (per Design decisions).

## Acceptance Criteria

- [ ] `scripts/validate.sh` passes with the four committed binaries present.
- [ ] The committed-binary smoke asserts exit `4`; the not-installed smoke asserts
      exit `3` and the releases URL.
- [ ] The existing build / plugin-package / release-artifact / shim-smoke steps still
      pass.
