# Lifecycle Status Dictionary

Pure mapping and terminal-membership functions bridge persisted job lifecycle strings and public `result.Status` values.

## Rationale

Persisted state (`running`, `complete`, `failed`, `cancelled`) and wire state (`running`, `success`, `failed`, `cancelled`, `blocked`) overlap but are not the same contract. Named mappings keep cancellation and completion transitions consistent and conservative on unknown values.

## Examples

- `internal/app/jobs.go:159` — `ResultStatusFromJob` maps persisted lifecycle to wire status.
- `internal/app/jobs.go:174` — `IsTerminalJobStatus` delegates persisted terminal membership.
- `internal/app/jobs.go:180` — `JobStatusFromResult` maps wire results back to persisted state.
- `internal/app/jobs.go:252` — terminal membership on the result side derives from persisted mappings.
- `internal/jobs/store.go:265` — `IsTerminalStatus` is the authoritative persisted terminal set.

## When to use

Use named mappings whenever job and result state cross application/persistence boundaries.

## When not to use

Do not add transient implementation states to the persisted or public dictionary without first extending the contract.

## Common violations

- Re-spelling terminal sets inline.
- Treating unknown persisted values as successful or terminal.
- Updating one side of the dictionary without race and transition tests.
