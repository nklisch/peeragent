---
id: epic-mcp-server
kind: epic
stage: implementing
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

## Design decisions

- **Capability split**: Establish blocking/async delegation and shared application services first, add job controls second, then package the complete server for both hosts. This keeps protocol foundations in one owner while making distribution depend on a stable tool surface.
- **Plugin configuration**: Use separate Claude Code and Codex MCP config files referenced by their respective manifests. Their native plugin-root variables differ, and relying on undocumented cross-host interpolation would make installed plugins brittle.
- **SDK/API selection**: Verify the current official Go MCP SDK during delegation-feature design before committing to concrete protocol APIs. The epic fixes behavior and boundaries, not a stale library signature.
- **Long work**: Keep blocking delegation for parity, but guide MCP hosts toward async launch for agent runs likely to exceed host tool-call timeouts.

## Decomposition

The epic is split by delivered capability rather than implementation layer. The first feature establishes a usable delegation server and shared application boundary; the second completes its async job lifecycle; the third makes the complete server installable through both plugin ecosystems.

### Child features

- `epic-mcp-server-delegation` — stdio server, shared application services, blocking delegation, and async launch — depends on: `[]`
- `epic-mcp-server-job-control` — MCP status, result, and cancellation tools — depends on: `[epic-mcp-server-delegation]`
- `epic-mcp-server-plugin-distribution` — Claude Code/Codex bundled MCP configuration, packaging, validation, and docs — depends on: `[epic-mcp-server-delegation, epic-mcp-server-job-control]`

### Decomposition risks

- The current CLI composition root contains execution and job behavior; extraction must preserve exit-code and terminal-race semantics rather than creating MCP-only copies.
- MCP stdout is a protocol channel, while several current paths write results or exit directly. The application boundary must return values and typed errors instead of writing or terminating the process.
- Host MCP calls commonly have shorter timeouts than delegated agent runs. Async launch must be the documented path for substantial work even though blocking delegation remains available.
- Claude Code and Codex use different plugin-root variables. Separate configs avoid interpolation assumptions but increase packaging drift risk, so validation must assert both.
