---
id: epic-mcp-server
kind: epic
stage: done
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

## Design review

A fresh-context GLM 5.2 review challenged SDK/plugin assumptions, cancellation cleanup, application boundaries, concurrency evidence, and over-complex test seams. Accepted changes:

- recorded official SDK v1.6.1 and host plugin documentation evidence in the owning feature designs;
- added manual install/discovery acceptance for both plugin hosts;
- required post-commit cancellation cleanup to survive MCP caller disconnection;
- fixed the application-service constructor contract and simplified process-control injection;
- made MCP use of canonical validation, stdout/exit boundaries, Go toolchain audit, cross-repository `cwd` risk, and race/concurrency tests explicit;
- reduced duplicate subprocess smoke coverage to one packaged execution plus structural checks for both configs;
- retained separate status/result tools because compact polling should not return potentially large target details.

Rejected as out of scope or contrary to the existing contract: renaming the established Alt Subagent text surface, removing reserved `blocked` status, and requiring byte-identical JSON whitespace rather than semantic result compatibility.

## Implementation summary

All three child features and seven child stories are at `stage: done`:

- `epic-mcp-server-delegation` — shared application boundary, canonical input normalization, official MCP Go SDK v1.6.1, Go 1.25 baseline, stdio server, typed blocking/async `delegate`, generated schema and stdout-purity coverage.
- `epic-mcp-server-job-control` — shared status/result/cancel services, disconnect-safe terminal cleanup, typed `job_status`, `job_result`, and destructive/idempotent `job_cancel`, concurrency and race coverage.
- `epic-mcp-server-plugin-distribution` — host-specific Claude/Codex MCP configs, manifest wiring, packaged shim validation, CI triggers, MCP-first skills with wrapper fallback, and complete user/safety guidance.

Implementation deviations were resolved in-item: SDK v1.6.1 requires Go 1.25 rather than the initially researched moving-branch minimum; child completion remains rooted in the CLI adapter but delegates terminal persistence to `internal/app`; local host CLIs cannot non-interactively enumerate bundled tools, so host server discovery is paired with exact four-tool packaged-protocol smoke evidence.

Verification across implementation and review:

- `scripts/validate.sh` passed end-to-end, including tests, build, packaging, release artifacts, deterministic MCP config validation, packaged protocol smoke, strict Claude plugin validation, docs checks, and shim smokes.
- Full Go suite passed with 182+ tests across 12 packages; focused race tests passed for application, MCP, and CLI packages; `go vet ./...` and command builds passed.
- Claude Code 2.1.201 accepted the plugin strictly; Codex CLI 0.144.1 installed the local marketplace plugin and discovered the bundled server.
- Three independent fresh-context GLM 5.2 feature reviews approved the implementation. Accepted findings (misleading test name and extraction-residue wrappers) were fixed before child-feature approval; remaining comments were nits or disclosed host limitations.

Effective worker capability: highest, selected for cross-cutting protocol, process-lifecycle, generated-contract, and plugin-distribution risk. Effective review weight: standard (autopilot default).

## Review notes

- Effective review weight: standard; lane: fresh-context deep aggregate review.
- Reviewer: GLM 5.2. It independently reran vet, focused tests, race tests, schema inspection, packaged protocol validation, mirror checks, and substrate-state checks across the complete epic.
- Verdict: approve. No blocker, important finding, or material cross-feature gap.
- Low comments accepted as non-blocking follow-up opportunities rather than release blockers: MCP blocking calls do not create a CLI-style log artifact; the documented `peeragent mcp` subcommand must remain first; one defensive cancellation fallback is unreachable; plugin-default cwd assumes standard host project launch behavior. None violates the accepted contract, loses result data, or weakens safety.
- Aggregate acceptance verified: CLI compatibility, four generated MCP tools, async and cancellation invariants, stdout purity, Go 1.25 baseline, both plugin install paths, packaging/release inclusion, approval/cwd/full-access guidance, skill fallback, and terminal substrate consistency.
