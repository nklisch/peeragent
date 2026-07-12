---
id: epic-mcp-server-plugin-distribution
kind: feature
stage: drafting
tags: [infra, docs]
parent: epic-mcp-server
depends_on: [epic-mcp-server-delegation, epic-mcp-server-job-control]
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# MCP plugin distribution

## Brief

Bundle the peeragent stdio server as an MCP component in both the Claude Code and Codex plugins. Each host receives an MCP configuration that resolves the packaged platform shim through its native plugin-root variable, while packaging and validation keep source and curated plugin trees synchronized.

Document plugin-provided usage, standalone MCP configuration, tool approval and timeout guidance, and troubleshooting. Validate both plugin manifests/configurations and smoke-test server startup from packaged layouts. This feature does not add another transport or alter delegation semantics.

## Epic context
- Parent epic: `epic-mcp-server`
- Position in epic: distribution consumer — ships after the complete MCP tool surface is available

## Foundation references
- `docs/VISION.md` — bundled MCP product shape
- `docs/SPEC.md` — host plugin components
- `docs/ARCHITECTURE.md` — host-specific MCP config and packaged binary resolution
