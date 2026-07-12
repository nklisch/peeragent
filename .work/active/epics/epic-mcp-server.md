---
id: epic-mcp-server
kind: epic
stage: drafting
tags: [infra]
parent: null
depends_on: []
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# MCP server support

## Brief

Expose peeragent as a local Model Context Protocol server so MCP-capable coding hosts can delegate work without shelling out through a host-specific skill. The MCP surface must reuse peeragent's existing execution and async-job contracts rather than introducing a second orchestration engine.

The first release is a stdio server for local process configuration. It exposes focused delegation plus async launch, status, result, and cancellation. The existing CLI and skills remain supported and share the same application behavior with the MCP adapter.

## Strategic decisions

- **MCP role**: Peeragent is an MCP server that exposes delegation to hosts; it does not forward arbitrary MCP server configuration into delegated agents.
- **Transport**: The first version supports stdio only. It does not open a network listener or add daemon lifecycle and authentication concerns.
- **Tool surface**: Include blocking delegation and the existing async job lifecycle (launch, status, result, cancel). Iterative peer-review orchestration remains in the host skill rather than becoming an MCP tool.
- **Compatibility**: Preserve the current CLI and JSON result contract. MCP is an adapter over shared application services, not an alternate implementation.
- **Plugin integration**: Bundle and enable the stdio server as an MCP component in both the Claude Code and Codex plugin packages. Each ecosystem gets a plugin-root-safe MCP config rather than requiring users to edit global MCP settings.

## Scope boundaries

- Define a stable MCP tool schema derived from peeragent's request and result contracts.
- Add a stdio MCP server entry mode and protocol adapter.
- Extract shared application services from the CLI composition root where needed.
- Cover protocol initialization, tool discovery, calls, errors, cancellation, and stdout purity.
- Bundle host-specific MCP configuration in the Claude Code and Codex plugins, including portable resolution of the packaged peeragent binary.
- Document plugin-provided and standalone configuration examples for supported MCP-capable hosts.
- Do not add Streamable HTTP, remote access, arbitrary MCP forwarding, or a first-class peer-review workflow.
