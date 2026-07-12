---
id: epic-mcp-server-plugin-distribution
kind: feature
stage: implementing
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

## Design decisions

- **Config files**: Add `.mcp.claude.json` and `.mcp.codex.json` at each plugin root and point each ecosystem manifest's `mcpServers` field at its own file. Claude uses `${CLAUDE_PLUGIN_ROOT}`; Codex uses `${PLUGIN_ROOT}`. Do not depend on undocumented cross-host variable compatibility. Verified 2026-07-12 against Claude Code's [Plugins reference — MCP servers](https://code.claude.com/docs/en/plugins-reference#mcp-servers) and OpenAI's [Build plugins — Bundled MCP servers](https://developers.openai.com/codex/plugins/build#bundled-mcp-servers-and-lifecycle-hooks) plus [Codex MCP — Plugin-provided servers](https://developers.openai.com/codex/mcp#plugin-provided-mcp-servers). Both document `mcpServers` path references; Codex documents direct or wrapped server maps.
- **Executable path**: Both configs execute the packaged `bin/peeragent` shim with argument `mcp`. The shim already selects the correct committed platform binary and produces actionable unsupported-platform failures.
- **Default availability**: Installing/enabling either plugin makes the MCP server discoverable under server key `peeragent`; users retain host-native enablement and approval controls.
- **Approval posture**: Documentation recommends approval for `delegate` and `job_cancel`, while status/result can be auto-approved. Tool annotations and host policy reinforce each other but neither is presented as a security boundary.
- **Skills coexistence**: Keep `/peer` and `/peer-review`. Skills can use the MCP tools when available but retain wrapper fallback for hosts or versions without plugin MCP support; peer-review remains host orchestration.

## Architectural choice

Ship host-specific declarative MCP configs beside one binary and one tool implementation. Extend the packaging script's curated-copy contract so source manifests/configs are authoritative and `plugin/` is generated. Validate both declarative configs and run the same protocol smoke test through the packaged shim.

Alternatives rejected:

1. **One shared `.mcp.json`** — simpler tree, but no single documented plugin-root variable is portable across both hosts.
2. **Shell indirection to detect root variables** — adds quoting and shell portability risk solely to avoid two tiny declarative files.
3. **Require global user MCP setup** — avoids plugin changes but contradicts installable plugin support and creates manual drift.
4. **Replace skills with MCP** — removes useful review orchestration and breaks older host installations unnecessarily.

## Implementation units

### Unit 1: Host MCP configurations and manifests
**Files**: `.mcp.claude.json`, `.mcp.codex.json`, `.claude-plugin/plugin.json`, `.codex-plugin/plugin.json`, `scripts/package-plugin.sh`, `plugin/.mcp.claude.json`, `plugin/.mcp.codex.json`, `plugin/.claude-plugin/plugin.json`, `plugin/.codex-plugin/plugin.json`
**Story**: `epic-mcp-server-plugin-distribution-config`

```json
// .mcp.claude.json
{
  "mcpServers": {
    "peeragent": {
      "command": "${CLAUDE_PLUGIN_ROOT}/bin/peeragent",
      "args": ["mcp"]
    }
  }
}
```

```json
// .mcp.codex.json
{
  "peeragent": {
    "command": "${PLUGIN_ROOT}/bin/peeragent",
    "args": ["mcp"]
  }
}
```

Add `"mcpServers": "./.mcp.claude.json"` to the Claude manifest and `"mcpServers": "./.mcp.codex.json"` to the Codex manifest. Package both files into the curated plugin root. Update descriptions/capabilities to mention MCP delegation without claiming remote service support.

**Acceptance criteria**:
- [ ] Claude plugin validation accepts the custom MCP config path.
- [ ] Codex plugin manifest and direct-server-map config match current documented schemas.
- [ ] A local Claude plugin load/reload discovers the bundled `peeragent` server and its four tools.
- [ ] A local Codex marketplace install discovers the bundled `peeragent` server and its four tools; record the tested host versions in implementation notes.
- [ ] Packaged configs reference `plugin/bin/peeragent`, not source-build paths.
- [ ] Regenerating `plugin/` yields no curated-tree drift.

### Unit 2: Packaging and protocol validation
**Files**: `scripts/validate.sh`, `scripts/package-plugin.sh`, `cmd/peeragent/mcp_stdio_test.go`, `.github/workflows/build-binaries.yml`
**Story**: `epic-mcp-server-plugin-distribution-config`

Extend validation to assert both source and packaged MCP files exist, parse as JSON, point manifests to the right files, carry only stdio command/args, and contain the expected host root variable. Run `claude plugin validate` when installed, with deterministic structural checks as the CI baseline because Codex CLI currently has no `plugin validate` subcommand. Add `.mcp*.json` to binary-refresh workflow path filters.

Run one canonical MCP initialize/list-tools subprocess smoke against the packaged shim and assert four tools plus clean protocol stdout. Structurally validate both host configs, their root variables, and their resolved shim path rather than repeating the same server subprocess for each declarative file.

**Acceptance criteria**:
- [ ] Validation fails on missing/stale configs, wrong root variables, wrong manifest pointers, or missing package copies.
- [ ] Build CI reruns when MCP configs change.
- [ ] The in-memory/source-server integration test and one packaged-shim subprocess smoke both pass initialize/list-tools checks.
- [ ] Existing release archives include the configs through the plugin package.

### Unit 3: User and skill guidance
**Files**: `README.md`, `docs/CONTRACT.md`, `docs/SPEC.md`, `docs/ARCHITECTURE.md`, `skills/peer/SKILL.md`, `skills/peer-review/SKILL.md`, `plugin/skills/peer/SKILL.md`, `plugin/skills/peer-review/SKILL.md`
**Story**: `epic-mcp-server-plugin-distribution-guidance`

Document:

- automatic MCP availability after Claude Code or Codex plugin installation;
- the four tool names and blocking/async workflow;
- standalone `peeragent mcp` configuration for non-plugin MCP hosts;
- async-first guidance for agent work likely to exceed tool timeouts;
- Codex plugin-scoped enablement/approval examples and equivalent Claude controls;
- explicit `full_access`, destructive cancellation, and optional cross-repository `cwd` posture;
- plugin reload/restart and protocol-debug troubleshooting;
- skill behavior: prefer MCP tools when present, fall back to the bundled wrapper, never recursively delegate through peeragent.

**Acceptance criteria**:
- [ ] Installation docs require no separate global MCP setup for either bundled plugin.
- [ ] Standalone examples use `peeragent mcp` and clearly require an installed executable.
- [ ] Guidance does not claim MCP review orchestration or HTTP support.
- [ ] Canonical and packaged skills stay byte-identical.

## Implementation order

1. `epic-mcp-server-plugin-distribution-config`
2. `epic-mcp-server-plugin-distribution-guidance`

## Testing

- Parse all manifests/configs in validation and assert exact cross-file pointers.
- Run Claude plugin validation opportunistically and deterministic JSON/path checks everywhere.
- Manually install/load each local host plugin once, verify server/tool discovery, and record host versions in the item implementation notes.
- Exercise packaged shims on all committed platform targets through existing cross-build/release checks; run one live protocol smoke on the current platform.
- Grep documentation for stale claims that peeragent has only skills/CLI or that Codex models cannot use MCP.
- Assert canonical and curated skill/config mirrors.

## Risks

- **Root interpolation**: Host variables are similar but not interchangeable. Separate files are deliberate; tests must prevent accidental consolidation.
- **Binary refresh sequencing**: MCP configs can land before CI refreshes committed platform binaries. Packaged-shim smoke must fail clearly until the matching binary exists, and release must wait for refresh.
- **Tool timeout expectations**: Plugin manifests do not universally control host tool-call timeouts. Documentation must make async the reliable path rather than promising blocking completion.
- **Host version drift**: Claude has a plugin validator; current Codex does not. Keep structural validation in-repo and treat live host validation as additional evidence, not the only gate.
