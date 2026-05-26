---
id: epic-plugin-foundation
kind: epic
stage: implementing
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

## Design Decisions

- **Should this be distributable from day one or start as a repo-local skill?** Distributable Claude Code plugin from day one. The plugin shape is part of the product, not a later packaging step.

## Decomposition

Split by plugin-facing capability. The manifest establishes installability, the Go skeleton establishes build structure, the executable entrypoint exposes the wrapper to Claude Code, and the skill defines when Claude should delegate to Codex. The skill depends on the entrypoint shape so its instructions name the real command behavior.

### Child features

- `epic-plugin-foundation-manifest` — plugin metadata and distributable Claude Code plugin identity — depends on: `[]`
- `epic-plugin-foundation-go-skeleton` — Go module, command package, and internal package layout — depends on: `[]`
- `epic-plugin-foundation-entrypoint` — `bin/codex-implement` executable shim that invokes the built Go wrapper — depends on: `[epic-plugin-foundation-go-skeleton]`
- `epic-plugin-foundation-skill` — `skills/codex-implement/SKILL.md` delegation instructions for Claude — depends on: `[epic-plugin-foundation-entrypoint]`

### Decomposition risks

The main risk is packaging the Go binary inside a Claude Code plugin across platforms. The foundation should keep the shim and build layout explicit so later packaging work can choose the final distribution strategy without changing the skill contract.

<!-- The design pass on each child feature will fill in real specifics. -->
