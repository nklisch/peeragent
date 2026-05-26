---
id: epic-packaging-docs-user-guide
kind: feature
stage: review
tags: [docs]
parent: epic-packaging-docs
depends_on: [epic-packaging-docs-build-artifacts]
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# User Guide

## Brief

This feature adds the practical user documentation for installing and using Codex Implement as a Claude Code plugin. It should cover prerequisites, installation shape, blocking delegation, async jobs, full-access escalation, effort selection, JSON/text output, and troubleshooting for Codex CLI availability or authentication issues.

The capability delivered here is operational clarity for a developer trying the plugin from the repository. It should align the public README and the bundled skill instructions so Claude and humans see the same defaults.

This feature does not introduce new CLI modes or promise unsupported behavior such as default worktree isolation.

## Epic Context

- Parent epic: `epic-packaging-docs`
- Position in epic: consumes build artifact decisions so setup instructions point at the real distributable path.

## Foundation References

- `docs/VISION.md` — autonomous implementor expectations.
- `docs/SPEC.md` — runtime assumptions and modes.
- `docs/CONTRACT.md` — CLI synopsis and result behavior.
- `skills/codex-implement/SKILL.md` — Claude-facing invocation guidance.

## Architectural Choice

Use `README.md` as the human-facing entry point and keep `skills/codex-implement/SKILL.md` as the Claude-facing operating guide. The README should teach setup and examples; the skill should stay compact and focused on when Claude should call the wrapper and how to interpret results.

Options considered:

- README only: good for humans, but risks Claude-facing drift.
- Skill only: useful after install, but poor for repository discovery and setup.
- README plus skill alignment: chosen because the plugin has two audiences.

## Implementation Units

### Unit 1: Human User Guide

**File**: `README.md`

```markdown
# Codex Implement

## Prerequisites
## Build
## Claude Code plugin layout
## Usage
## Async jobs
## Safety and access
## Troubleshooting
```

**Implementation Notes**:
- Document `make build` / `scripts/build.sh` and `dist/codex-implement`.
- Show explicit commands for blocking, `--effort high`, `--async`, `--status`, `--result`, `--cancel`, `--full-access`, `--text`, and `--prompt-file`.
- State that `--effort` supports only `medium` and `high`, with `medium` default.
- State that `--worktree` is recognized but not implemented yet.

**Acceptance Criteria**:
- [ ] A new user can identify prerequisites and build the wrapper.
- [ ] Usage examples match actual supported flags.
- [ ] Troubleshooting covers missing Codex CLI/authentication and async lookup failures.

---

### Unit 2: Skill Alignment

**File**: `skills/codex-implement/SKILL.md`

```markdown
## Options
- `--effort high`
- `--async`
- `--status <job-id>`
- `--result <job-id>`
- `--cancel <job-id>`
```

**Implementation Notes**:
- Keep the skill concise; do not turn it into a full manual.
- Mention status/result/cancel so Claude can reconnect to async jobs.
- Preserve the default blocking behavior and explicit full-access guardrail.

**Acceptance Criteria**:
- [ ] Claude-facing instructions mention async follow-up commands.
- [ ] Skill defaults match README and contract docs.

---

### Unit 3: Contract Drift Cleanup

**File**: `docs/CONTRACT.md`

```text
codex-implement --status <job-id>
codex-implement --result <job-id>
```

**Implementation Notes**:
- The parser requires explicit job ids; update the synopsis to match implementation.
- Do not change the wrapper behavior in this feature.

**Acceptance Criteria**:
- [ ] Contract synopsis no longer implies optional status/result job ids.

## Implementation Order

1. Add `README.md`.
2. Update skill async follow-up guidance.
3. Correct contract synopsis.
4. Verify examples against the wrapper where cheap.

## Testing

### Verification Commands

- `go test ./...`
- `make build`
- `bin/codex-implement --status missing-job`

## Risks

Docs may imply behavior the wrapper does not yet support. Keep examples limited to implemented or explicitly recognized behavior, and label `--worktree` as not implemented.

## Implementation Notes

Added `README.md` with prerequisites, build instructions, plugin layout, blocking usage, text/JSON behavior, effort selection, async lifecycle commands, safety/full-access posture, and troubleshooting. Updated the Claude-facing skill with async follow-up commands and corrected `docs/CONTRACT.md` so status/result require explicit job ids.

Verification:

- `go test ./...`
- `make build`
- `bin/codex-implement --status missing-job`
- `rg -- "--status \\[|--result \\[|xhigh|extra-high" README.md docs/CONTRACT.md skills/codex-implement/SKILL.md`

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
