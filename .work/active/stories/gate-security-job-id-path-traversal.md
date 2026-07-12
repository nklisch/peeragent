---
id: gate-security-job-id-path-traversal
kind: story
stage: done
tags: [security]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: security
created: 2026-07-12
updated: 2026-07-12
---

# Reject path traversal in async job ids

## Severity
Medium

## Domain
Input validation / path injection

## Location
`internal/app/jobs.go:89`, `internal/jobs/store.go:187`

## Evidence
```go
raw.JobID = strings.TrimSpace(raw.JobID)
if raw.JobID == "" {
    return JobRequest{}, errors.New("job id is required")
}

func (s Store) jobDir(id string) string {
    return filepath.Join(s.Root, id)
}
```

`filepath.Join` cleans traversal segments instead of rejecting them. CLI and MCP callers can provide a crafted id that escapes `.peeragent/jobs/`, probes path existence, and reaches job-store reads/writes outside the intended root.

## Remediation direction

Validate externally supplied job ids against peeragent's generated id grammar before any store access, and independently make `jobs.Store` reject unsafe path components as defense in depth. Add CLI, MCP, and store regression tests for traversal, separators, malformed ids, and valid generated ids.

## Design

- Make the generated job-id grammar authoritative in `internal/jobs` with an exported validation function used by both generation tests and store path construction.
- Reject malformed ids before every store path join; application job request normalization calls the same validator for fail-fast CLI/MCP errors.
- MCP handlers continue delegating to application normalization rather than duplicating grammar.
- Cover valid generated ids, traversal, absolute paths, both separators, dot segments, wrong timestamp/hex lengths, Unicode, and NUL at store and application/MCP boundaries.
- Return a tool/CLI input error without leaking target file contents; preserve exit-code-4 only for a syntactically valid but missing job.

## Implementation Notes

- Added the authoritative `jobs.ValidateID` grammar: an exact UTC timestamp (`YYYYMMDDTHHMMSSZ`) followed by eight lowercase hexadecimal bytes. Timestamp parsing rejects impossible dates; separators, dot segments, absolute paths, Unicode, NUL, and malformed lengths/hex are rejected before any job-store path join.
- `jobs.Store` validates through `jobDir` for every id-bearing operation. Application normalization and CLI input normalization reuse the same validator; MCP continues to delegate validation to the application boundary.
- Existing missing-job behavior remains a structured exit-code-4 result for syntactically valid ids. Invalid ids return input errors and do not create or probe the store root.

## Verification

- Focused tests: `go test ./internal/jobs ./internal/app ./internal/input ./internal/mcp ./cmd/peeragent`
- Race tests: `go test -race ./internal/jobs ./internal/app ./internal/mcp ./internal/input ./cmd/peeragent`
- Full tests: `go test ./...`
- Static/build checks: `go vet ./...`; `go build ./...`
- All checks passed. Story advanced from `implementing` to `review`; review owns approval.

## Review

Approved in an independent source review of commit `e1186d8`. The reviewer verified fail-fast boundary validation, store defense in depth, cross-platform separator rejection, generated-id compatibility, valid-missing exit-code-4 behavior, and focused/race/full verification. One informational error-context asymmetry was non-blocking.
