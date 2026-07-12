---
id: gate-tests-job-id-validation
kind: story
stage: implementing
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
