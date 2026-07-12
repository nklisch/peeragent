---
id: epic-mcp-server-delegation-application-services
kind: story
stage: done
tags: [infra]
parent: epic-mcp-server-delegation
depends_on: []
release_binding: 0.5.0
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# Extract delegation application services

## Scope

Create the canonical delegation request normalizer and extract blocking execution plus async launch from `cmd/peeragent` into an `internal/app` service. Keep the CLI as an adapter that formats results and chooses exit codes; application code returns values and never writes stdout or exits the process.

## Acceptance criteria

- [x] CLI parsing and MCP delegation request validation both use `input.NormalizeDelegation`; no second validation path exists.
- [x] Blocking execution and async launch are callable through injected application ports.
- [x] Existing CLI, target routing, result, launch-cleanup, and async tests remain green.
- [x] No package under `internal/` calls `os.Exit` or writes `os.Stdout`; validation enforces this adapter boundary.
- [x] Application tests cover successful, target-failed, and infrastructure-failed paths without local agent CLIs.

## Implementation notes
- Execution capability: highest, selected by the autopilot caller because this extraction changes process execution, CLI boundaries, and shared contracts.
- Review weight: standard (project default; caller did not override it).
- Dispatch rationale: direct-read only; the feature design and existing CLI/job/target packages fully identified the integration surface, so no exploratory worker was needed.
- Files changed: `internal/input/delegation.go`, `internal/input/delegation_test.go`, `internal/input/input.go`, `internal/app/service.go`, `internal/app/execute.go`, `internal/app/service_test.go`, `cmd/peeragent/main.go`.
- Tests added: canonical delegation normalization coverage; application success, target-failure, infrastructure-failure, cancellation, async launch, and injected-launcher coverage.
- Discoveries: the CLI's raw target logging requires an application `DelegateWithExecution` seam in addition to the specified `Delegate` method; it preserves log fidelity without exposing raw output to MCP or duplicating target execution.
- Discrepancies from design: infrastructure errors return both the established failed result and the underlying error so CLI can preserve its result contract while MCP can map the same condition to a protocol tool error. No behavioral discrepancy for target exit failures or async launch cleanup.
- Adjacent issues parked: none.
- Verification: `go test ./...` passed (146 tests across 11 packages); internal packages contain no `os.Exit` or `os.Stdout` writes.

## Review notes

- Effective review weight: standard (autopilot default); escalated to fresh-context deep review because this story changes the shared caller/execution boundary.
- Evidence: fresh GLM 5.2 review plus `go build`, `go vet ./...`, `go test ./...`, and race tests over app/input/MCP/CLI packages.
- Verdict: approve with comments. Renamed the misleading executable-resolution test to describe the verified contract: the failed launch returns the id of the job record already persisted for that attempt.
- Remaining comments were non-blocking cleanup/style nits.
