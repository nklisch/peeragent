---
id: gate-tests-cancel-corrupt-result
kind: story
stage: drafting
tags: [testing]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: tests
created: 2026-07-12
updated: 2026-07-12
---

# Cover cancellation with corrupt persisted result

## Priority
Medium

## Spec reference
Item: `epic-mcp-server-job-control-application-services`
Acceptance criterion: "Corrupt storage remains an infrastructure error."

## Gap type
Missing corrupt-state coverage on the cancel path.

## Suggested test
```go
func TestCancelJobReportsCorruptResultAsInfrastructureError(t *testing.T) {
    // Create a running job, write malformed result.json, invoke CancelJob,
    // and assert a decode/infrastructure error without fabricating cancellation.
}
```

## Test location (suggested)
`internal/app/cancel_test.go`
