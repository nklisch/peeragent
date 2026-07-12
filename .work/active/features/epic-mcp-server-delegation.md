---
id: epic-mcp-server-delegation
kind: feature
stage: drafting
tags: [infra]
parent: epic-mcp-server
depends_on: []
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# MCP delegation server

## Brief

Add the shared application boundary and stdio MCP server needed to expose peeragent delegation outside the CLI. This feature owns protocol initialization, instructions and tool discovery, the blocking delegation call, and async delegation launch. It must derive tool inputs from the same validation rules as CLI requests and return the existing result contract without leaking target or diagnostic output onto protocol stdout.

This feature also establishes the server entry mode and the internal application services that both CLI and MCP adapters call. It does not expose status, result, or cancellation tools, and it does not package the server into host plugins.

## Epic context
- Parent epic: `epic-mcp-server`
- Position in epic: foundation capability — job-control and plugin-distribution features depend on its application and protocol boundaries

## Foundation references
- `docs/SPEC.md` — MCP stdio and execution contracts
- `docs/ARCHITECTURE.md` — inbound adapters, application services, and stdout purity
