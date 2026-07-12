---
id: epic-mcp-server-delegation-stdio-server
kind: story
stage: implementing
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
