---
id: epic-packaging-docs-build-artifacts
kind: feature
stage: implementing
tags: [docs, infra]
parent: epic-packaging-docs
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Build Artifacts

## Brief

This feature makes the plugin distributable from a fresh checkout. It covers the expected compiled Go wrapper artifact, the executable shim behavior, and a repeatable local build path that produces the binary Claude will call.

The capability delivered here is packaging mechanics, not new wrapper behavior. It should preserve the existing fallback from `bin/codex-implement` to `go run` for development while documenting or automating the preferred `dist/codex-implement` path for distribution.

This feature does not publish to a registry or implement cross-platform release automation unless the existing project shape makes that trivial.

## Epic Context

- Parent epic: `epic-packaging-docs`
- Position in epic: independent packaging foundation that validation and install docs can reference.

## Foundation References

- `docs/SPEC.md` — platform-compatible wrapper binary requirement.
- `docs/ARCHITECTURE.md` — plugin layout and executable entrypoint.

## Architectural Choice

Use a small repository-local shell build script as the source of truth for distribution artifacts, with an optional `Makefile` target as a convenience alias. The script builds the Go wrapper into `dist/codex-implement`, matching the existing `bin/codex-implement` shim preference.

Options considered:

- Shell script only: simple and portable enough for the current repo, but less discoverable for users who expect `make`.
- Makefile only: discoverable, but pushes platform logic into make syntax.
- Script plus Makefile alias: keeps logic in one executable script while offering a conventional command. Chosen.

## Implementation Units

### Unit 1: Build Script

**File**: `scripts/build.sh`

```sh
#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
mkdir -p "$ROOT/dist"
go build -o "$ROOT/dist/codex-implement" "$ROOT/cmd/codex-implement"
```

**Implementation Notes**:
- Use paths relative to the script location so it works from any current directory.
- Leave cross-compilation flags to future release automation.
- Ensure the script is executable.

**Acceptance Criteria**:
- [ ] `scripts/build.sh` creates `dist/codex-implement`.
- [ ] The built binary exits with JSON argument errors instead of failing to launch.

---

### Unit 2: Makefile Alias

**File**: `Makefile`

```make
.PHONY: build test clean

build:
	./scripts/build.sh

test:
	go test ./...

clean:
	rm -rf dist
```

**Implementation Notes**:
- Keep the Makefile thin; `scripts/build.sh` remains the single build implementation.
- Do not add a release target until packaging validation defines the release checklist.

**Acceptance Criteria**:
- [ ] `make build` delegates to `scripts/build.sh`.
- [ ] `make test` runs the existing Go suite.

---

### Unit 3: Shim Alignment Check

**File**: `bin/codex-implement`

```sh
if [ -x "$PLUGIN_ROOT/dist/codex-implement" ]; then
  exec "$PLUGIN_ROOT/dist/codex-implement" "$@"
fi
```

**Implementation Notes**:
- No code change is expected unless implementation finds drift.
- Smoke the shim after building to verify it uses the compiled artifact path.

**Acceptance Criteria**:
- [ ] `bin/codex-implement --status missing-job` reaches the built wrapper after `scripts/build.sh`.

## Implementation Order

1. Add `scripts/build.sh`.
2. Add `Makefile` aliases.
3. Build the binary and smoke the shim.

## Testing

### Unit Tests

No new Go unit tests are required; this feature is packaging glue.

### Verification Commands

- `go test ./...`
- `scripts/build.sh`
- `test -x dist/codex-implement`
- `bin/codex-implement --status missing-job`

## Risks

The smoke command intentionally returns a missing-job failure, so validation should check that the wrapper starts and emits contract-shaped JSON rather than expecting a successful lookup.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
