---
id: committed-binary-distribution-docs-skills
kind: story
stage: review
tags: [docs, infra]
parent: committed-binary-distribution
depends_on: [committed-binary-distribution-shim, committed-binary-distribution-packaging]
release_binding: null
gate_origin: null
created: 2026-05-31
updated: 2026-05-31
---

# Skills and foundation docs

## Scope

Implements Unit 5 of `committed-binary-distribution`. Update host skills and roll the
foundation docs forward in place.

- **`skills/peer/SKILL.md`, `skills/peer-review/SKILL.md`**: add a "not installed"
  case to Result Handling / Guardrails — if the wrapper reports `status: failed` with
  exit code `3`, instruct the user to install peeragent from the GitHub releases page
  for their OS/arch (set `PEERAGENT_BIN`, or place the binary at
  `<plugin>/bin/<goos>-<goarch>/peeragent`), mention `PEERAGENT_TARGET_OVERRIDE` for a
  misdetected platform, and do not retry in a loop.
- **`docs/CONTRACT.md`**: add exit code `3: wrapper binary unavailable for this
  platform` to the Exit Codes section.
- **`docs/ARCHITECTURE.md`**: roll the Overview flow and Wrapper Role to the new
  resolution order (override → local build → `go run` → committed platform binary →
  install error); remove download/cache language.
- **`docs/SPEC.md`**: roll Runtime Context and Components to describe committed
  per-platform binaries in the plugin artifact and the supported-platform set;
  release tarballs remain the manual-install source.
- **`README.md`**: install section — supported platforms run with no download/build;
  other platforms install from releases (`PEERAGENT_BIN` / target-override guidance).
  Remove stale download env-var docs. Keep the `make build`, marketplace-add, and
  `make release VERSION=...` examples that `validate.sh` greps for.

## Acceptance Criteria

- [ ] Both skills describe the not-installed handling and the install URL.
- [ ] `CONTRACT.md` documents exit `3`; `ARCHITECTURE.md` / `SPEC.md` describe the
      committed-binary model with no download/cache language.
- [ ] `scripts/validate.sh` documentation-examples and skill-metadata steps pass.

## Notes

Rolling-foundation: edit docs in place, no "previously/now" prose. `CONTRACT.md` also
carries stale `/codex-implement` skill names (pre-existing drift) — out of scope here;
leave for gate-docs unless trivially adjacent.

## Implementation notes

- `skills/peer/SKILL.md`: added not-installed case under Result Handling (exit code 3
  branch with GitHub releases URL and `PEERAGENT_BIN`/`PEERAGENT_TARGET_OVERRIDE`
  guidance) and a reinforcing one-liner in Guardrails.
- `skills/peer-review/SKILL.md`: same not-installed block under Result Handling and a
  Guardrails bullet for exit code 3.
- `docs/CONTRACT.md`: added `- \`3\`: Wrapper binary unavailable for this platform.`
  in numeric order between `2` and `4`. The external agent had already added a
  `--sandbox` option entry — that change was preserved intact.
- `docs/ARCHITECTURE.md`: updated Overview fenced diagram to show the three-way
  resolution (`dist/peeragent`, `go run`, `bin/<goos>-<goarch>/peeragent`); rewrote
  Wrapper Role paragraph to describe the full resolution order ending in exit code 3.
- `docs/SPEC.md`: updated Runtime Context bullet to name the four committed platforms
  and manual-install path; added `plugin/bin/<target>/peeragent` to Components list.
  External agent edits to Sandbox section were preserved intact.
- `README.md`: replaced stale "downloads binary / falls back to go run" paragraph in
  Install section with committed-binary model description; updated Troubleshooting to
  point to GitHub releases instead of stale download language. External agent had added
  a `--sandbox` sentence — that was preserved intact. All `validate.sh`-grepped
  strings confirmed present; no forbidden substrings introduced.
