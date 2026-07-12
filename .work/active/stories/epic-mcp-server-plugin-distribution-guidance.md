---
id: epic-mcp-server-plugin-distribution-guidance
kind: story
stage: implementing
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
