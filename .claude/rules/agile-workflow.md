---
description: Agile-workflow substrate navigation rules
paths: ['.work/**', 'docs/**']
---

# Agile-Workflow Substrate Navigation

## Folder Structure

.work/active/{epics,features,stories}/  in-flight, scoped
.work/backlog/                           parked, unscoped
.work/releases/<version>/                shipped bundles
.work/archive/                           done items not bound to a release

## Item Kinds

epic     multi-feature arc; has children    parent of features
feature  design + implementation unit       parent of stories
story    single-session unit                leaf or has tasks
task     checklist line in parent body      not its own file
release  version bundle in releases/        binds items via release_binding

## Stages

epic     drafting -> implementing -> review -> done
feature  drafting -> implementing -> review -> done
story    implementing -> review -> done       (often skips drafting)
task     [ ] -> [x]
release  planned -> quality-gate -> released

## Frontmatter

id, kind, stage, tags[], parent, depends_on[], release_binding,
gate_origin, created, updated

## Querying With Work-View

`.work/bin/work-view` is the canonical query tool. Use it instead of hand-grepping frontmatter when filtering items. Filters compose with AND semantics. Run `--help` for the authoritative flag list.

### Filters

--stage <stage>      drafting | implementing | review | done | released
--tag <tag>          repeatable; AND across tags
--kind <kind>        epic | feature | story | release
--parent <id>        direct children of given item
--release <version>  items with release_binding: <version>
--gate <name>        items produced by gate <name>
--ready              stage:implementing AND all depends_on done
--blocked            stage:implementing AND unmet dependencies
--blocking <id>      items that depend on <id>

### Output Modes

(default tabular)    columns: ID  KIND  STAGE  TAGS  PARENT
--paths              one file path per line (pipe-friendly)
--cat                full item bodies, separated by ---
--count              match count only

### Common Queries

Items ready to work right now:

```bash
.work/bin/work-view --ready
```

Items awaiting user review:

```bash
.work/bin/work-view --stage review
```

All children of an epic:

```bash
.work/bin/work-view --parent <epic-id>
```

Children of an epic that are still blocked:

```bash
.work/bin/work-view --parent <epic-id> --blocked
```

Read full bodies of every item bound to a release:

```bash
.work/bin/work-view --release v1.2.0 --cat
```

Security-tagged items currently implementing:

```bash
.work/bin/work-view --stage implementing --tag security
```

Items that would unblock if `<id>` finishes:

```bash
.work/bin/work-view --blocking <id>
```

Pipe paths into another tool:

```bash
.work/bin/work-view --ready --paths | xargs grep -l 'TODO'
```

## Fallback: Raw Substrate Access

When work-view does not fit, such as searching item bodies instead of frontmatter:

```bash
grep -rn '<phrase>' .work/active/
```

Item history:

```bash
git log -p -- .work/active/features/<id>.md
```

Recent substrate changes:

```bash
git log --since='1 day ago' -- .work/
```

## Session Start Checklist

1. Read `.work/CONVENTIONS.md` for project-specific overrides.
2. Run `.work/bin/work-view --stage review` for items waiting on attention.
3. Run `.work/bin/work-view --ready` for items ready to implement.
4. Read relevant foundation docs in `docs/`.

## Editing Rules

- Update `stage:` only when work actually completes.
- Do not pre-populate future stage transitions.
- Use `depends_on` for sequencing and `parent` for hierarchy.
- Keep dependency graphs acyclic.
- Move done, unbound items to `.work/archive/`.
- Move shipped, bound items to `.work/releases/<version>/`.
- Use `git mv` for tier transitions so history is retained.

## Rolling Foundation

Foundation docs in `docs/` describe the system now. Do not add roadmap prose, legacy notes, or future-version promises to foundation docs. Update them in place when the system changes; Git history is the audit trail.

