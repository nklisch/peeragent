---
id: epic-mcp-server-delegation-stdio-server
kind: story
stage: review
tags: [infra]
parent: epic-mcp-server-delegation
depends_on: [epic-mcp-server-delegation-application-services]
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# Add stdio MCP delegate tool

## Scope

Raise the Go baseline to 1.25, add the official MCP Go SDK v1.6.1 in this story (and no earlier story), route `peeragent mcp` to a stdio server, and expose the typed `delegate` tool over the shared application service. The tool supports blocking and async calls and returns the existing result contract as structured output.

## Acceptance criteria

- [ ] MCP initialization, instructions, list-tools, generated schemas, and delegate calls work through an in-memory client/server test.
- [ ] Blocking and async results preserve the peeragent result schema.
- [ ] Invalid delegation input is rejected before execution.
- [ ] Context cancellation reaches blocking target execution.
- [ ] A subprocess smoke test proves protocol stdout purity.
- [ ] CI workflows, build scripts, and installation docs are audited for independent Go 1.22 assumptions; the supported minimum is stated as Go 1.25.
- [ ] Existing CLI behavior remains green under Go 1.25.

## Design correction

Implementation verified that the `github.com/modelcontextprotocol/go-sdk@v1.6.1` tag declares `go 1.25.0`; the earlier design had inspected the repository's moving default branch and incorrectly recorded Go 1.23. Autopilot resolves the contradiction in favor of the fixed current stable SDK and raises peeragent's baseline to Go 1.25. CI already derives the toolchain from `go.mod`, and the local environment provides Go 1.26.4.

## Implementation notes
- Execution capability: highest, selected by the autopilot caller because this changes the MCP protocol boundary, stdio process behavior, CLI entrypoint, generated schemas, and the Go dependency contract.
- Review weight: standard (project default; caller did not override it).
- Dependency readiness: the application-services predecessor was rechecked at `stage: review`; implementation proceeded in the explicit caller-provided story sequence without marking that predecessor done.
- Dispatch rationale: direct-read only; the corrected story, feature design, SDK API evidence, existing application service, and CLI tests fully identified the integration surface.
- Files changed: `go.mod`, `go.sum`, `README.md`, `cmd/peeragent/main.go`, `cmd/peeragent/main_mcp_test.go`, `internal/mcp/server.go`, `internal/mcp/tools.go`, `internal/mcp/server_test.go`.
- Tests added: official SDK in-memory initialization/tool discovery and generated-schema checks; blocking structured result; async running result; semantic and schema-invalid input rejection; infrastructure errors; context cancellation propagation; subprocess stdio protocol-purity smoke coverage.
- Go baseline audit: `go.mod` now declares Go 1.25.0 and directly requires `github.com/modelcontextprotocol/go-sdk v1.6.1`; CI workflows derive their toolchain from `go.mod`, and Make/scripts/install documentation contain no independent Go 1.22 assumption. README now states Go 1.25 as the supported minimum.
- Discrepancies from design: the SDK's own stdio transport batches protocol writes until a response is read, so the subprocess smoke test follows the real request/response sequence instead of closing stdin immediately after a burst. No product-contract discrepancy.
- Adjacent issues parked: none.
- Verification: focused MCP tests passed (11 tests); CLI/MCP subprocess tests passed; `go test -race ./internal/mcp` passed; `go test ./...` passed (159 tests across 12 packages); `go build -o /tmp/peeragent-mcp ./cmd/peeragent` passed.
