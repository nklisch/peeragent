---
id: epic-safety-permissions-full-access
kind: feature
stage: done
tags: [security, infra]
parent: epic-safety-permissions
depends_on: [epic-safety-permissions-defaults]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Full Access Opt-In

## Brief

This feature adds explicit `--full-access` behavior. When selected, the wrapper uses Codex's full-access/bypass mode instead of the classifier-compatible defaults and reports that access posture in output.

The feature exists for trusted contexts where the caller intentionally wants maximum autonomy. It does not make full access implicit.

## Epic Context

- Parent epic: `epic-safety-permissions`
- Position in epic: explicit alternate to default permissions.

## Foundation References

- `docs/VISION.md` — explicit full-access mode.
- `docs/SPEC.md` — full access is not default.
- `docs/CONTRACT.md` — `--full-access`.

## Architectural Choice

Add `--full-access` to input parsing and carry it through `codex.Options`. When selected, Codex argv uses `--dangerously-bypass-approvals-and-sandbox` instead of default approval/sandbox flags.

Alternative considered: combine `--sandbox danger-full-access` with `--ask-for-approval never`. Rejected because Codex exposes an explicit bypass flag for this posture and the name makes risk visible in argv.

## Implementation Units

### Unit 1: Input Flag

**File**: `internal/input/input.go`

Add `FullAccess bool` to `Request` and parse `--full-access`.

**Acceptance Criteria**:
- [ ] `--full-access` sets `Request.FullAccess`.
- [ ] Normal calls leave `FullAccess` false.

### Unit 2: Codex Options

**File**: `internal/codex/exec.go`

Add `FullAccess bool` to `Options`. When true, build args with `--dangerously-bypass-approvals-and-sandbox` and omit default approval/sandbox/reviewer flags.

**Acceptance Criteria**:
- [ ] Full-access argv contains `--dangerously-bypass-approvals-and-sandbox`.
- [ ] Full-access argv omits default approval flags.

## Implementation Order

1. Add input flag and tests.
2. Add Codex option and tests.
3. Wire `main.go`.
4. Run tests.

## Testing

### Unit Tests

Test flag parsing and full-access argv construction.

## Risks

Full access weakens normal safety boundaries. Keep the flag explicit and avoid any shorthand aliases.

## Implementation Notes

- Added `--full-access` parsing.
- Added `codex.Options.FullAccess`.
- Full-access argv uses `--dangerously-bypass-approvals-and-sandbox` and omits default approval/reviewer flags.
- Added unit tests for parsing and argv construction.

## Review

Approved. Full access is explicit, tested, and maps to Codex's clearly named bypass flag without adding ambiguous shorthand modes.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
