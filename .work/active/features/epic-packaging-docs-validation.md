---
id: epic-packaging-docs-validation
kind: feature
stage: done
tags: [docs, infra]
parent: epic-packaging-docs
depends_on: [epic-packaging-docs-build-artifacts, epic-packaging-docs-user-guide]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Validation

## Brief

This feature adds a lightweight release-readiness check for the plugin surface. It should verify the Go tests, build path, executable shim, manifest basics, and documented command examples enough to catch obvious packaging drift before a user installs the plugin.

The capability delivered here is confidence in the distributable repo state. It should be runnable locally and should report failures clearly without depending on a live Codex implementation pass when a cheaper smoke check is available.

This feature does not replace future release gates or CI; it creates the first pragmatic validation loop for this repository.

## Epic Context

- Parent epic: `epic-packaging-docs`
- Position in epic: final packaging/docs child; depends on the build and documentation surfaces it validates.

## Foundation References

- `docs/SPEC.md` — packaging and runtime assumptions.
- `docs/CONTRACT.md` — CLI behavior and output contract.
- `.claude-plugin/plugin.json` — plugin metadata.

## Architectural Choice

Add `scripts/validate.sh` as the release-readiness entry point and expose it through `make validate`. The script should use only repository-local files and common POSIX tools so it remains cheap to run before packaging.

Options considered:

- Manual checklist in README: easy to write, but too easy to skip and drift.
- CI-only validation: premature because no CI exists yet.
- Local script plus Makefile alias: chosen because it is executable now and can be reused by CI later.

## Implementation Units

### Unit 1: Validation Script

**File**: `scripts/validate.sh`

```sh
#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT"

go test ./...
scripts/build.sh
test -x dist/codex-implement
test -x bin/codex-implement
grep -q '"name": "codex-implement"' .claude-plugin/plugin.json
grep -q 'name: codex-implement' skills/codex-implement/SKILL.md
grep -q 'make build' README.md
grep -q -- '--effort high' README.md
```

**Implementation Notes**:
- Capture `bin/codex-implement --status missing-job`; require exit code `4` and JSON containing `"job_id":"missing-job"`.
- Scan docs for stale `--status [job-id]` and `--result [job-id]` syntax.
- Keep output terse but label each phase.

**Acceptance Criteria**:
- [ ] `scripts/validate.sh` passes on the current repository.
- [ ] Missing async job smoke proves the compiled shim starts and returns JSON.
- [ ] Docs drift checks cover effort and async job syntax.

---

### Unit 2: Makefile Alias

**File**: `Makefile`

```make
.PHONY: validate

validate:
	./scripts/validate.sh
```

**Implementation Notes**:
- Leave `build`, `test`, and `clean` intact.
- `validate` should be the single command recommended for release readiness.

**Acceptance Criteria**:
- [ ] `make validate` runs the same checks as `scripts/validate.sh`.

---

### Unit 3: README Validation Reference

**File**: `README.md`

```markdown
## Validation

make validate
```

**Implementation Notes**:
- Keep this concise; the script is the source of truth.

**Acceptance Criteria**:
- [ ] README points contributors to `make validate`.

## Implementation Order

1. Add `scripts/validate.sh`.
2. Add `make validate`.
3. Add README validation section.
4. Run `make validate`.

## Testing

### Verification Commands

- `scripts/validate.sh`
- `make validate`

## Risks

The validation script should not require a live Codex implementation call. It should only run the wrapper against cheap local failure paths and avoid network-dependent checks.

## Implementation Notes

Added `scripts/validate.sh` as the local release-readiness check, wired `make validate`, and documented validation in the README. The script runs Go tests, builds the wrapper, checks executable artifacts, verifies plugin metadata, scans documented command examples for drift, and smokes the shim through the missing-job JSON path.

Verification:

- `scripts/validate.sh`
- `make validate`

## Review

Approved. Validation is executable, local, and does not require a live Codex implementation run. It checks the core packaging and documentation surfaces and passed in review with `make validate`.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
