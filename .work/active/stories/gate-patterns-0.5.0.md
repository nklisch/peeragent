---
id: gate-patterns-0.5.0
kind: story
stage: done
tags: [patterns]
parent: null
depends_on: []
release_binding: 0.5.0
gate_origin: patterns
created: 2026-07-12
updated: 2026-07-12
---

# Patterns extracted for 0.5.0

## New patterns codified

- `target-executor-adapter` — isolate each external agent CLI behind the shared runner/result port.
- `mcp-typed-tool-handler` — normalize typed MCP input and delegate one operation to application services.
- `job-state-fs-error-categorization` — distinguish normal file absence from infrastructure failures at each call site.
- `runner-test-double-per-package` — test target adapters offline through recording runners and lookPath seams.
- `lifecycle-status-dictionary` — bridge persisted and public status vocabularies through named mappings.

## Inconsistencies flagged

- Agent vocabulary is switched outside the target adapter/application metadata boundary.
- Runner test doubles are duplicated byte-for-byte across four target packages.
- Terminal lifecycle sets are re-spelled across application and persistence predicates.

## Pattern files written

- `.agents/skills/patterns/target-executor-adapter.md`
- `.agents/skills/patterns/mcp-typed-tool-handler.md`
- `.agents/skills/patterns/job-state-fs-error-categorization.md`
- `.agents/skills/patterns/runner-test-double-per-package.md`
- `.agents/skills/patterns/lifecycle-status-dictionary.md`
- `.agents/skills/patterns/SKILL.md`
- `.agents/rules/patterns.md`
- `.claude/skills/patterns` (compatibility symlink)
