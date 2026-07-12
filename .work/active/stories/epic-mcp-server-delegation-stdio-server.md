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

Raise the Go baseline to 1.23, add the official MCP Go SDK v1.6.1, route `peeragent mcp` to a stdio server, and expose the typed `delegate` tool over the shared application service. The tool supports blocking and async calls and returns the existing result contract as structured output.

## Acceptance criteria

- [ ] MCP initialization, instructions, list-tools, generated schemas, and delegate calls work through an in-memory client/server test.
- [ ] Blocking and async results preserve the peeragent result schema.
- [ ] Invalid delegation input is rejected before execution.
- [ ] Context cancellation reaches blocking target execution.
- [ ] A subprocess smoke test proves protocol stdout purity.
- [ ] Existing CLI behavior remains green under Go 1.23.
