---
id: epic-mcp-server-plugin-distribution-config
kind: story
stage: done
tags: [infra]
parent: epic-mcp-server-plugin-distribution
depends_on: []
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# Bundle and validate plugin MCP configuration

## Scope

Add host-specific Claude Code and Codex MCP config files, reference them from each manifest, copy them into the curated plugin package, and extend CI/validation. Both configs execute the packaged shim with `mcp`; each uses only its documented native plugin-root variable.

## Acceptance criteria

- [x] Both source and packaged plugin manifests point to the correct MCP configs.
- [x] Packaging keeps manifests/configs synchronized.
- [x] Deterministic validation catches schema, pointer, root-variable, and mirror drift.
- [x] Claude plugin validation passes when available; Codex config follows its documented direct server map.
- [x] Local Claude validation and local Codex marketplace install discover the bundled server; packaged protocol smoke verifies all four tools, with host enumeration limitations recorded below.
- [x] The source integration test and one packaged-shim subprocess smoke pass initialize/list-tools checks.
- [x] MCP config changes trigger committed-binary CI.

## Implementation notes

- Execution capability: highest, selected by the autopilot caller because this changes two host manifest contracts, packaged executable resolution, deterministic validation, and the MCP protocol smoke boundary.
- Review weight: standard (explicit caller override).
- Dispatch rationale: direct-read only; the story, parent feature, foundation docs, existing manifests/package flow, host help output, and MCP tests fully identified the integration surface. No agent delegation was used.
- Files changed: `.mcp.claude.json`, `.mcp.codex.json`, `.claude-plugin/plugin.json`, `.codex-plugin/plugin.json`, `plugin/.mcp.claude.json`, `plugin/.mcp.codex.json`, `plugin/.claude-plugin/plugin.json`, `plugin/.codex-plugin/plugin.json`, `scripts/package-plugin.sh`, `scripts/validate-plugin-config.py`, `scripts/validate.sh`, `cmd/peeragent/main_mcp_test.go`, `.github/workflows/build-binaries.yml`.
- Tests and validation: source MCP subprocess test now asserts initialize/tools-list responses expose exactly `delegate`, `job_status`, `job_result`, and `job_cancel`; the packaged shim smoke performs the same checks with clean protocol stdout; JSON/schema, manifest-pointer, host-root-variable, and source/package mirror checks are deterministic; `claude plugin validate --strict plugin` passed.
- Host evidence: Claude Code `2.1.201` accepted the packaged Claude manifest through `claude plugin validate --strict plugin`. Codex CLI `0.144.1` installed the local marketplace plugin and `codex mcp list` discovered the enabled `peeragent` stdio server with `${PLUGIN_ROOT}/bin/peeragent mcp`.
- Environment limitations: Claude has no non-interactive command that reports bundled MCP tool discovery; a live plugin session would require an authenticated prompt. Codex `0.144.1` exposes server discovery via `codex mcp list` but no command to enumerate the server's tools. The protocol tool list was therefore verified directly through the packaged shim. The committed `plugin/bin/linux-amd64/peeragent` binary predates this MCP source change and is owned by the existing binary-refresh workflow; local smoke used `PEERAGENT_BIN=$ROOT/dist/peeragent` through the packaged shim without modifying committed platform binaries.
- Discrepancies from design: the design names `cmd/peeragent/mcp_stdio_test.go`; the repository's existing source integration test is `cmd/peeragent/main_mcp_test.go`, so that file was extended instead. The design's optional host validator is run for Claude; Codex has no equivalent validator command.
- Adjacent issues parked: none.

## Review notes

- Effective review weight: standard; fresh-context deep review selected for cross-host manifest and packaged-executable contracts.
- Reviewer: GLM 5.2. It independently verified manifest pointers, host-native root variables, source/package byte equality, shim resolution, exact four-tool protocol discovery, annotations, CI filters, and deterministic config validation.
- Verdict: approve. No blocking or important findings; host limitations remain accurately disclosed.
