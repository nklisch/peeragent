---
id: epic-safety-permissions-profile-reporting
kind: feature
stage: implementing
tags: [security, infra]
parent: epic-safety-permissions
depends_on: [epic-safety-permissions-full-access, epic-safety-permissions-worktree]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Profile And Access Reporting

## Brief

This feature adds Codex profile pass-through and access-mode metadata. Claude can tell whether a run used default, full-access, or worktree posture and which Codex profile was requested.

The feature exists so safety decisions are visible in results and downstream logs. It does not redefine the result contract beyond adding minimal metadata fields.

## Epic Context

- Parent epic: `epic-safety-permissions`
- Position in epic: metadata and pass-through layer after permission modes exist.

## Foundation References

- `docs/CONTRACT.md` — `--profile` and access metadata.
- `docs/SPEC.md` — reporting what Codex did and preserving user control.

## Architectural Choice

Add `--profile <name>` to input parsing and Codex options. The executor appends `--profile <name>` to `codex exec` when present. The CLI output includes `access` and `profile` metadata so Claude can report the execution posture.

Alternative considered: defer metadata until the result-contract epic. Rejected because access posture is safety-critical and should be visible as soon as alternate modes exist. The result-contract epic can reshape fields later.

## Implementation Units

### Unit 1: Profile Parsing

**File**: `internal/input/input.go`

Add `Profile string` to `Request` and parse `--profile <name>`.

**Acceptance Criteria**:
- [ ] `--profile codex-subagent` sets `Request.Profile`.
- [ ] Missing profile value errors.

### Unit 2: Codex Profile Arg

**File**: `internal/codex/exec.go`

Add `Profile string` to `Options` and append `--profile <name>` when present.

**Acceptance Criteria**:
- [ ] Unit test verifies profile arg is included.

### Unit 3: Access Metadata

**File**: `cmd/codex-implement/main.go`

Emit `access` as `default`, `full-access`, or `worktree`; emit `profile` when supplied.

**Acceptance Criteria**:
- [ ] JSON output includes access metadata.
- [ ] Full-access output reports `full-access`.
- [ ] Worktree unsupported output reports `worktree`.

## Implementation Order

1. Add input fields and tests.
2. Add Codex options and tests.
3. Add output metadata.
4. Run tests and a worktree error smoke test.

## Testing

### Unit Tests

Cover profile parsing and Codex argv.

### Smoke Test

Run `go run ./cmd/codex-implement --worktree task` and verify JSON includes worktree access.

## Risks

The result-contract epic may rename fields. Keep this metadata small and obvious so it can be folded into the final schema cleanly.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
