---
id: epic-plugin-foundation-skill
kind: feature
stage: done
tags: [infra, docs]
parent: epic-plugin-foundation
depends_on: [epic-plugin-foundation-entrypoint]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Codex Implement Skill

## Brief

This feature creates `skills/codex-implement/SKILL.md`, the Claude-facing delegation instructions. The skill explains when Claude should delegate implementation to Codex, how to invoke the wrapper, how to pass arbitrary task text, and how to interpret the structured result.

The feature exists to make delegation feel natural instead of exposing a large Codex command surface. It does not implement wrapper internals or async behavior.

## Epic Context

- Parent epic: `epic-plugin-foundation`
- Position in epic: consumer of the command entrypoint; its instructions must match the real executable surface.

## Foundation References

- `docs/VISION.md` — Claude as primary collaborator and Codex as autonomous implementor.
- `docs/SPEC.md` — skill name and invocation shape.
- `docs/ARCHITECTURE.md` — skill role.
- `docs/CONTRACT.md` — skill contract and CLI synopsis.

## Design Decisions

- **Distribution posture**: Distributable Claude Code plugin from day one.

## Architectural Choice

Create a thin Claude Code skill at `skills/codex-implement/SKILL.md`. The skill describes when Claude should delegate, how to pass arbitrary task text to the `codex-implement` command, and how to treat the JSON result.

Alternative considered: expose a broad command manual for all wrapper modes. Rejected because the product goal is natural delegation, not a Codex control panel. The skill should bias toward the blocking implementation path and mention advanced flags only as escape hatches.

## Implementation Units

### Unit 1: Skill Definition

**File**: `skills/codex-implement/SKILL.md`

```markdown
---
name: codex-implement
description: Delegate implementation work to OpenAI Codex CLI through the bundled codex-implement wrapper.
allowed-tools: Bash
---
```

**Implementation Notes**:
- The skill should accept arbitrary `$ARGUMENTS`.
- It should instruct Claude to call `codex-implement` and return the JSON result.
- It should preserve Claude's responsibility for interpreting failed or blocked results.
- It should mention full-access and async as explicit options, not defaults.

**Acceptance Criteria**:
- [ ] `skills/codex-implement/SKILL.md` exists.
- [ ] The frontmatter name is `codex-implement`.
- [ ] The skill directs Claude to the bundled `codex-implement` executable.
- [ ] The skill makes JSON output the default expectation.

## Implementation Order

1. Create `skills/codex-implement/SKILL.md`.
2. Check the skill frontmatter and command references.

## Testing

### Static Validation

Read the skill file and verify it names the right skill, command, default behavior, and escalation posture.

## Risks

If the skill becomes too explicit, it recreates the control-panel feel the project avoids. Keep the default path short and delegation-oriented.

## Implementation Notes

- Created `skills/codex-implement/SKILL.md`.
- The skill delegates arbitrary `$ARGUMENTS` to `codex-implement`.
- The skill treats JSON as the default result and documents explicit escape hatches for `--full-access`, `--worktree`, `--async`, and `--prompt-file`.

## Review

Approved. The skill is thin, delegation-oriented, names the correct wrapper command, and keeps advanced modes explicit. This completes the Claude-facing foundation for the plugin.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
