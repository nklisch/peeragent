---
id: gate-tests-job-id-validation
kind: story
stage: done
tags: [testing]
parent: null
depends_on: [gate-security-job-id-path-traversal]
release_binding: 0.5.0
gate_origin: tests
created: 2026-07-12
updated: 2026-07-12
---

# Verify job-id traversal and grammar rejection

## Priority
High

## Spec reference
Item: `gate-security-job-id-path-traversal`
Acceptance criterion: "Add CLI, MCP, and store regression tests for traversal, separators, malformed ids, and valid generated ids."

## Gap type
Missing security regression tests for an explicit bound validation contract.

## Suggested test
```go
func TestRejectsUnsafeJobIDs(t *testing.T) {
    for _, id := range []string{"../escape", "..", "/abs/path", "a/b", "a\\b", "."} {
        // Store, application, CLI, and MCP boundaries reject before filesystem access.
    }
    // A generated job id remains accepted.
}
```

## Test location (suggested)
`internal/jobs/store_test.go`, `internal/app/jobs_test.go`, `internal/mcp/jobs_test.go`, and CLI parsing tests.

## Implementation Notes

Landed with the security remediation in commit `e1186d8`. Coverage now exercises the authoritative grammar and valid generated ids, traversal/dot/absolute paths, both separator styles, malformed timestamp/hex/length values, Unicode and NUL at store, application, MCP, and CLI/input boundaries. Tests also verify malformed ids do not resolve the working directory or probe the store, while valid missing ids retain structured exit-code-4 behavior.

## Verification

- `go test ./internal/jobs ./internal/app ./internal/input ./internal/mcp ./cmd/peeragent`
- `go test -race ./internal/jobs ./internal/app ./internal/mcp ./internal/input ./cmd/peeragent`
- `go test ./...`
- `go vet ./...`
- `go build ./...`

All passed in land mode after dependency `gate-security-job-id-path-traversal` reached done.

## Review

Approved from the independent security review of commit `e1186d8`, which explicitly verified store/application/MCP/CLI coverage, cross-platform separators, malformed grammar cases, valid generated ids, missing-job semantics, and race/full-suite results.
