---
id: epic-mcp-server-job-control
kind: feature
stage: drafting
tags: [infra]
parent: epic-mcp-server
depends_on: [epic-mcp-server-delegation]
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# MCP async job control

## Brief

Expose peeragent's existing async job lifecycle through MCP: status inspection, result retrieval, and cancellation. The tools must preserve terminal-state race handling, job lookup semantics, repository scoping, and the shared result schema rather than reimplementing filesystem or process behavior inside protocol handlers.

This feature extends the server and application boundary established by `epic-mcp-server-delegation`. It does not add a dashboard, job listing, remote transport, or review orchestration.

## Epic context
- Parent epic: `epic-mcp-server`
- Position in epic: consumer of the shared application and MCP server foundation

## Foundation references
- `docs/SPEC.md` — async lifecycle and MCP tool surface
- `docs/ARCHITECTURE.md` — async flow, cancellation, and MCP adapter role
