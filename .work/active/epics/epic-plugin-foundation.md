---
id: epic-plugin-foundation
kind: epic
stage: drafting
tags: [infra]
parent: null
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Plugin Foundation

## Brief

This epic establishes Codex Implement as a Claude Code plugin with the expected file layout, manifest, skill name, and local execution assumptions. It creates the project surface that Claude Code can discover and use.

The capability delivered here is the plugin shell: `.claude-plugin/plugin.json`, `skills/codex-implement/SKILL.md`, `bin/codex-implement`, and the initial script structure. The skill gives Claude clear delegation guidance while keeping the implementation details in the wrapper CLI.

This epic does not implement the full Codex execution contract, async job handling, or detailed result formatting. Those belong to downstream epics once the plugin can be discovered and invoked.

## Foundation References

- `docs/VISION.md` — product definition and delegation goals.
- `docs/SPEC.md` — name, runtime context, and component layout.
- `docs/ARCHITECTURE.md` — plugin layout and skill role.

## Anticipated Child Features

- Plugin manifest and metadata.
- `codex-implement` skill instructions.
- Executable wrapper entrypoint.
- Initial script/runtime skeleton.

<!-- The design pass on each child feature will fill in real specifics. -->

