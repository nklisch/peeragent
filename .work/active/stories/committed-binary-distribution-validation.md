---
id: committed-binary-distribution-validation
kind: story
stage: done
tags: [infra]
parent: committed-binary-distribution
depends_on: [committed-binary-distribution-shim, committed-binary-distribution-packaging]
release_binding: 0.5.0
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

## Implementation notes

Added the "committed platform binaries" step to `scripts/validate.sh` immediately after
the "plugin package" block and before the "release artifacts" block, as specified.

### Direct verification results

- Four platform binaries (`linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`):
  all exist, executable, and nonzero-size. ✓
- `plugin/bin/peeragent --status missing-job` → exit 4, JSON with `"status":"failed"`
  and `"exit_code":4`. ✓
- `PEERAGENT_TARGET_OVERRIDE=plan9-sparc plugin/bin/peeragent --agent codex x` → exit 3,
  JSON with `"exit_code":3` and `releases` URL. ✓

### Full validate.sh outcome

All steps passed end-to-end:

```
==> go tests
ok  	github.com/nklisch/peeragent/cmd/peeragent	5.094s
ok  	github.com/nklisch/peeragent/internal/claude	(cached)
ok  	github.com/nklisch/peeragent/internal/codex	(cached)
?   	github.com/nklisch/peeragent/internal/executil	[no test files]
ok  	github.com/nklisch/peeragent/internal/gemini	(cached)
ok  	github.com/nklisch/peeragent/internal/input	(cached)
ok  	github.com/nklisch/peeragent/internal/jobs	0.105s
ok  	github.com/nklisch/peeragent/internal/prompt	(cached)
ok  	github.com/nklisch/peeragent/internal/result	(cached)

==> build
==> plugin package
==> committed platform binaries
==> release artifacts
==> plugin metadata
==> skill metadata constraints
==> documentation examples
==> shim smoke
==> validation complete
```

No external blockers. All steps including go tests, build, plugin-package, the new
committed-platform-binaries step, release-artifacts, shim-smoke, and documentation
examples passed cleanly.
