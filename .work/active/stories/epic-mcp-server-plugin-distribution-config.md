---
id: epic-mcp-server-plugin-distribution-config
kind: story
stage: implementing
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

- [ ] Both source and packaged plugin manifests point to the correct MCP configs.
- [ ] Packaging keeps manifests/configs synchronized.
- [ ] Deterministic validation catches schema, pointer, root-variable, and mirror drift.
- [ ] Claude plugin validation passes when available; Codex config follows its documented direct server map.
- [ ] Built binary and packaged shim pass initialize/list-tools smoke tests.
- [ ] MCP config changes trigger committed-binary CI.
