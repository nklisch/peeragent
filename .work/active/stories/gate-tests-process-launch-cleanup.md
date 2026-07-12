---
id: gate-tests-process-launch-cleanup
kind: story
stage: implementing
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
