---
id: epic-safety-permissions-worktree
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

# Worktree Opt-In

## Brief

This feature defines the `--worktree` opt-in surface. The default product does not isolate work, but callers can request worktree behavior. If the first implementation does not create worktrees yet, it should fail clearly instead of silently ignoring the flag.

The feature exists to keep isolation as a deliberate escape hatch without complicating the default path.

## Epic Context

- Parent epic: `epic-safety-permissions`
- Position in epic: explicit alternate execution posture beside full access.

## Foundation References

- `docs/VISION.md` — no worktree unless explicitly requested.
- `docs/SPEC.md` — same checkout default.
- `docs/CONTRACT.md` — `--worktree`.

## Architectural Choice

Add `--worktree` to input parsing, but return a clear unsupported error for now. This preserves the explicit public surface without quietly changing repository behavior or creating a partial isolation implementation.

Alternative considered: create a temporary git worktree immediately. Rejected because worktree naming, cleanup, result application, and user expectations deserve a dedicated implementation pass.

## Implementation Units

### Unit 1: Input Flag

**File**: `internal/input/input.go`

Add `Worktree bool` to `Request` and parse `--worktree`.

**Acceptance Criteria**:
- [ ] `--worktree` sets `Request.Worktree`.

### Unit 2: Unsupported Mode Error

**File**: `cmd/codex-implement/main.go`

If `req.Worktree` is true, emit JSON failure explaining that worktree mode is recognized but not implemented yet.

**Acceptance Criteria**:
- [ ] `--worktree` does not silently run in the raw checkout.
- [ ] Error output is JSON and exits non-zero.

## Implementation Order

1. Add input flag and tests.
2. Add early main guard.
3. Run tests.

## Testing

### Unit Tests

Test flag parsing. Main-level behavior can be covered by smoke test or later command tests.

## Risks

A recognized-but-unsupported flag is less capable than ideal, but it is safer than silently ignoring isolation requests.

## Implementation Notes

- Added `--worktree` parsing.
- Added an early JSON failure for worktree mode so isolation requests are not silently run in the raw checkout.
- Added unit coverage for parsing.

## Review

Approved. Worktree requests are recognized and fail clearly rather than silently using the raw checkout. That is the right safe behavior until real worktree creation and cleanup are designed.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
