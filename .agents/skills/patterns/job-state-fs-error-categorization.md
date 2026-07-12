# Job-State Filesystem Error Categorization

Every persisted job-file read translates `fs.ErrNotExist` into that call site's normal business state and wraps every other filesystem error as infrastructure failure.

## Rationale

Absence means different things by file: missing `job.json` is a lookup failure, missing `result.json` means still running, and missing `pid` means cleanup is already complete. Permission, decode, and I/O failures must not be mistaken for those normal states.

## Examples

- `internal/app/jobs.go:34` — missing job metadata becomes the structured exit-code-4 result.
- `internal/app/jobs.go:61` — missing result data becomes a running result.
- `internal/app/jobs.go:240` — absent stored terminal result returns `(found=false)`.
- `internal/app/cancel.go:43` — missing job during cancellation becomes lookup failure.
- `internal/app/cancel.go:127` — missing PID returns the already-decided cancellation result.

## When to use

Use at application call sites that understand what an absent job-state file means.

## When not to use

Do not apply to internal lock/temp-file control flow, where existence is part of synchronization rather than domain state.

## Common violations

- Returning raw non-absence errors without operation context.
- Treating corrupt or permission-denied state as merely missing.
- Centralizing all absence handling in the store, which cannot know call-site semantics.
