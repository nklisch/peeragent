---
id: epic-plugin-foundation-manifest
kind: feature
stage: done
tags: [infra]
parent: epic-plugin-foundation
depends_on: []
release_binding: 0.5.0
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Plugin Manifest

## Brief

This feature creates the distributable Claude Code plugin identity for Codex Implement. It delivers `.claude-plugin/plugin.json` with the plugin name, description, version, author metadata, and discovery-compatible structure.

The feature exists so Claude Code can treat this project as a plugin from the beginning rather than as a repo-local skill. It does not implement the skill body, wrapper CLI behavior, or packaging validation.

## Epic Context

- Parent epic: `epic-plugin-foundation`
- Position in epic: independent foundation feature; other plugin files live alongside it but do not depend on its implementation details.

## Foundation References

- `docs/VISION.md` — product definition as a Claude Code plugin.
- `docs/SPEC.md` — component layout and distributable plugin assumption.
- `docs/ARCHITECTURE.md` — plugin layout.

## Design Decisions

- **Distribution posture**: Distributable Claude Code plugin from day one.

## Architectural Choice

Use the standard Claude Code plugin manifest at `.claude-plugin/plugin.json` with no extra runtime indirection. This keeps discovery aligned with Claude Code's plugin conventions and makes later packaging work additive.

Alternative considered: defer manifest creation until packaging. Rejected because distributability is a core project constraint and downstream features need a concrete plugin root.

## Implementation Units

### Unit 1: Plugin Manifest

**File**: `.claude-plugin/plugin.json`

```json
{
  "name": "codex-implement",
  "version": "0.1.0",
  "description": "Delegate Claude Code implementation tasks to OpenAI Codex CLI",
  "author": {
    "name": "nklisch"
  },
  "license": "MIT"
}
```

**Implementation Notes**:
- Keep the manifest minimal until packaging work introduces repository or marketplace metadata.
- The plugin name matches the skill and CLI name.

**Acceptance Criteria**:
- [ ] `.claude-plugin/plugin.json` exists.
- [ ] Manifest JSON parses successfully.
- [ ] Manifest name is `codex-implement`.

## Implementation Order

1. Create `.claude-plugin/plugin.json`.
2. Validate JSON syntax.

## Testing

### Validation

Use a JSON parser such as `python3 -m json.tool .claude-plugin/plugin.json` to validate syntax.

## Risks

Claude Code plugin manifest fields may grow as packaging is finalized. Keep this manifest intentionally small rather than guessing at distribution metadata now.

## Implementation Notes

- Created `.claude-plugin/plugin.json` with the distributable plugin identity.
- Validated the manifest as JSON.

## Review

Approved. The manifest is intentionally minimal, valid JSON, and matches the distributable plugin decision. This makes the plugin discoverable as a first-class Claude Code plugin root.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
