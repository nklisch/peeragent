---
id: gate-tests-process-launch-cleanup
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

# Cover real launcher cleanup failures

## Priority
Medium

## Spec reference
Item: `epic-mcp-server-delegation`
Acceptance criterion: "Launch cleanup still kills a started child when PID persistence or release fails."

## Gap type
Missing failure-path coverage for the production `ProcessLauncher` adapter.

## Suggested test
```go
func TestProcessLauncherKillsChildWhenPIDPersistenceFails(t *testing.T) {
    // Start a long-lived helper, force WritePID failure after Start,
    // and assert Launch returns an error and the process group exits.
}
```

Also cover the release-failure cleanup branch through an injectable process seam or the closest deterministic adapter-level test.

## Test location (suggested)
`internal/app/service_test.go`

## Design

- Refactor only enough process-launch infrastructure to inject PID persistence and process release failure seams deterministically; do not depend on chmod behavior under privileged CI.
- Exercise the production `ProcessLauncher` cleanup path with a long-lived helper and assert the child/group exits when PID persistence fails.
- Cover release failure through the same injectable seam and assert PID cleanup plus child termination.
- Preserve successful launch argv, detach behavior, and existing async integration tests; run race and full tests.

## Implementation notes
- Execution capability: highest, selected by the caller because detached process cleanup crosses OS process groups, persistence, and release failure paths.
- Review weight: deep, selected by the caller's highest-rigor requirement; independent review follows at `stage: review`.
- Files changed: `internal/app/service.go`, `internal/app/service_test.go`, this item file.
- Tests added: `TestProcessLauncherKillsChildWhenPIDPersistenceFails` and `TestProcessLauncherKillsChildAndRemovesPIDWhenReleaseFails`, using a long-lived Go test helper process and deterministic function dependencies.
- Discrepancies from design: none; the command factory seam avoids executable-permission failure injection while retaining production argv construction and detach attributes.
- Adjacent issues parked: none.
- Verification: focused process-launch cleanup tests passed with `go test ./internal/app -run 'TestProcessLauncher(KillsChildWhenPIDPersistenceFails|KillsChildAndRemovesPIDWhenReleaseFails)$' -count=1`; `go test -race ./internal/app ./internal/jobs`, `go test ./...`, `go vet ./...`, and `go build -o /tmp/peeragent-release-0.5.0 ./cmd/peeragent` all passed, and the temporary build artifact was removed.

## Review

Approved by independent source review of commit `047ef15`. The reviewer verified real long-lived process groups, cleanup on both failure branches, PID ordering/removal, deterministic unexported seams, Unix-specific honesty, and regression sensitivity. Final focused race verification passed.
