---
id: gate-tests-cancel-corrupt-result
kind: story
stage: done
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

## Design

Add one focused service test that creates a valid running job, writes malformed `result.json`, invokes `CancelJob` with fake process control, and asserts an infrastructure/decode error. Assert no cancellation result is fabricated, no process is signaled, and persisted running state remains unchanged. Run application race tests and the full suite.

## Implementation notes
- Execution capability: highest, selected by the caller because cancellation error handling protects durable state and process safety.
- Review weight: deep, selected by the caller's highest-rigor requirement; independent review follows at `stage: review`.
- Files changed: `internal/app/cancel_test.go`, this item file.
- Tests added: `TestCancelJobReportsCorruptResultAsInfrastructureError`.
- Discrepancies from design: none.
- Adjacent issues parked: none.
- Verification: focused test passed with `go test ./internal/app -run '^TestCancelJobReportsCorruptResultAsInfrastructureError$' -count=1`.

## Review

Approved by independent source review of commit `32681ba`. The test honestly verifies decode-error propagation, zero signal calls, no fabricated result, byte-stable persisted state, and unchanged running/PID state. Final focused race verification passed.
