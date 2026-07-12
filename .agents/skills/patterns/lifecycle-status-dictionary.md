# Lifecycle Status Dictionary

Pure mapping and terminal-membership functions bridge persisted job lifecycle strings and public `result.Status` values.

## Rationale

Persisted state (`running`, `complete`, `failed`, `cancelled`) and wire state (`running`, `success`, `failed`, `cancelled`, `blocked`) overlap but are not the same contract. Named mappings keep cancellation and completion transitions consistent and conservative on unknown values.

## Examples

- `internal/app/jobs.go:156` — `ResultStatusFromJob` maps persisted lifecycle to wire status.
- `internal/app/jobs.go:171` — `IsTerminalJobStatus` defines terminal persisted states.
- `internal/app/jobs.go:182` — `JobStatusFromResult` maps wire results back to persisted state.
- `internal/app/jobs.go:254` — terminal membership on the result side.
- `internal/jobs/store.go:212` — guarded-save terminal membership at the persistence boundary.

## When to use

Use named mappings whenever job and result state cross application/persistence boundaries.

## When not to use

Do not add transient implementation states to the persisted or public dictionary without first extending the contract.

## Common violations

- Re-spelling terminal sets inline.
- Treating unknown persisted values as successful or terminal.
- Updating one side of the dictionary without race and transition tests.
