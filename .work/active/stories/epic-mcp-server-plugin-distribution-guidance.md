---
id: epic-mcp-server-plugin-distribution-guidance
kind: story
stage: review
tags: [docs]
parent: epic-mcp-server-plugin-distribution
depends_on: [epic-mcp-server-plugin-distribution-config]
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# Document MCP use and skill integration

## Scope

Document automatic plugin MCP availability, tool contracts, standalone stdio setup, async-first operation, approval and timeout guidance, and troubleshooting. Update peer skills to prefer available MCP tools while retaining bundled-wrapper fallback and the no-recursion rule.

## Acceptance criteria

- [ ] Claude Code and Codex plugin users need no separate global MCP setup.
- [ ] All four tools and the async workflow are documented accurately.
- [ ] Full-access delegation and job cancellation are described as explicit write/destructive operations.
- [ ] Optional `cwd` is documented as intentional cross-repository reach and omitted by default unless the user requests it.
- [ ] Standalone setup, reload/restart, and troubleshooting guidance is actionable.
- [ ] Skills preserve peer-review orchestration and wrapper fallback.
- [ ] Canonical and packaged skill copies remain identical.

## Implementation notes

- Execution capability: highest, selected by the autopilot caller because this updates public MCP contracts, safety guidance, host-specific setup, troubleshooting, and both canonical/package skill surfaces.
- Review weight: standard (explicit caller override).
- Dispatch rationale: direct-read only; the story and parent design, existing MCP server contracts, manifests, host CLI help, and current docs/skills fully identified the writing surface. No agent delegation was used.
- Files changed: `README.md`, `docs/CONTRACT.md`, `docs/SPEC.md`, `docs/ARCHITECTURE.md`, `skills/peer/SKILL.md`, `skills/peer-review/SKILL.md`, and generated mirrors `plugin/skills/peer/SKILL.md`, `plugin/skills/peer-review/SKILL.md`.
- Guidance delivered: plugin installation automatically provides the local MCP server without global setup; all four tools and their blocking/async workflow are documented; `full_access`, `job_cancel`, approvals, timeout-aware async use, and intentional cross-repository `cwd` are explicit; standalone `peeragent mcp`, reload/restart, protocol stdout/stderr, Claude validation, and Codex discovery troubleshooting are documented; MCP does not provide HTTP or review orchestration.
- Skill behavior: both skills prefer MCP when available, use async polling for long passes, retain bundled-wrapper fallback, keep peer-review orchestration in the host, and prohibit recursive peeragent delegation.
- Host evidence: Claude Code `2.1.201` passed `claude plugin validate --strict plugin`. Codex CLI `0.144.1` installed the local marketplace plugin and reported the enabled bundled server through `codex mcp list` and `codex mcp get peeragent` as `${PLUGIN_ROOT}/bin/peeragent mcp`.
- Verification: `scripts/validate.sh` passed all steps (full Go tests, build, package, MCP JSON/mirror checks, packaged protocol smoke, strict Claude validation, release artifacts, metadata, skill constraints, documentation checks, and shim smokes); standalone `go test ./...` passed 182 tests; `scripts/build.sh` passed; canonical/package skill and config `cmp` checks passed.
- Environment limitations: neither Claude nor Codex exposes a non-interactive command in these versions that enumerates the four tools from an installed plugin. Codex server discovery was verified without faking tool enumeration; the four-tool protocol list was verified by the packaged-shim smoke using the fresh `dist/peeragent` through the packaged shim because committed platform binaries are owned by the existing refresh workflow and were intentionally not changed.
- Discrepancies from design: none.
- Adjacent issues parked: none.
