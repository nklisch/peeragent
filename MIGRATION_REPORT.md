# Migration Report

## Source Shape

greenfield

The project has foundation docs in `docs/` and no source tree, legacy roadmap, task list, or existing agile-workflow substrate.

## Foundation Docs Detected

- `docs/VISION.md`
- `docs/SPEC.md`
- `docs/ARCHITECTURE.md`
- `docs/CONTRACT.md`

Foundation docs were preserved as-is.

## Items Seeded

No active, backlog, release, or archive items were seeded. This is an empty greenfield substrate.

## Files Created

- `.work/CONVENTIONS.md`
- `.work/bin/work-view`
- `.claude/rules/agile-workflow.md`
- `CLAUDE.md`
- `MIGRATION_REPORT.md`

## Conventions Chosen

- Release mapping: `tag-based`
- Slug convention: kebab-case with parent-prefix for child items when useful
- Stage overrides: none
- Gates: `security`, `tests`, `cruft`, `docs`, `patterns`
- Starter tags: `security`, `tests`, `infra`, `docs`, `refactor`, `perf`

## Files Left In Place

All existing files were preserved.

## Next Steps

Run `/agile-workflow:epicize` to decompose the foundation docs into active epics.

