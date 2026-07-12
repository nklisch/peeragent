---
id: epic-mcp-server-job-control-tools
kind: story
stage: done
tags: [infra]
parent: epic-mcp-server-job-control
depends_on: [epic-mcp-server-job-control-application-services]
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# Add MCP async job tools

## Scope

Register typed `job_status`, `job_result`, and `job_cancel` tools over the shared application service. Include generated schemas, read/destructive annotations, structured peeragent results, and server instructions for the complete async workflow.

## Acceptance criteria

- [x] Tool discovery exposes delegate plus all three job tools with correct schemas.
- [x] Read-only annotations distinguish status/result from destructive cancellation.
- [x] Running, terminal, missing, corrupt, and cancellation-race outcomes map correctly.
- [x] Repository cwd defaults and explicit overrides use the same normalizer.
- [x] At least eight concurrent status/result/cancel calls against one job preserve consistent job/result files and produce one allowed terminal winner.
- [x] Application and MCP packages pass `go test -race` in validation.

## Implementation notes
- Execution capability: highest, retained from the autopilot caller because typed protocol contracts, destructive cancellation annotations, and concurrent MCP calls exercise the process-race boundary implemented by the prerequisite story.
- Review weight: standard (autopilot default).
- Files changed: `internal/mcp/jobs.go`, `internal/mcp/server.go`, `internal/mcp/tools.go`, `internal/mcp/server_test.go`, `internal/mcp/jobs_test.go`.
- Tests added: four-tool discovery/schema/annotation assertions; structured status/result/cancel calls; empty-id validation; default and explicit cwd integration; and eight-call concurrent terminal-state coverage against one real application job store.
- Design decisions: `NewServer` now requires the combined typed `ServerService`, preventing an MCP server from advertising an incomplete async surface; job handlers validate ids at the protocol boundary and delegate cwd normalization/storage/process behavior to `internal/app`.
- Discrepancies from design: none.
- Verification: `go test ./internal/mcp ./cmd/peeragent` passed (41 tests); race, full-suite, vet, and build evidence will be appended to the parent implementation summary.
- Adjacent issues parked: none.

## Review notes

- Effective review weight: standard; fresh-context deep review covered the complete async MCP contract.
- Evidence: generated schema/annotation inspection, structured call tests, cwd normalization, eight-way concurrency, full tests, race tests, vet, build, and manual terminal-race analysis.
- Verdict: approve with comments; no blocking or important finding applied to the MCP job tools.
