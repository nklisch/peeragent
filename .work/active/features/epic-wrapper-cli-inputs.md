---
id: epic-wrapper-cli-inputs
kind: feature
stage: drafting
tags: [infra]
parent: epic-wrapper-cli
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Wrapper Inputs

## Brief

This feature teaches the Go wrapper to collect task text from command-line arguments, standard input, and `--prompt-file`. It also resolves the working directory through `--cwd` or the current process directory.

The feature exists so Claude can pass short tasks ergonomically while long tasks and generated prompts have stable input paths. It does not invoke Codex.

## Epic Context

- Parent epic: `epic-wrapper-cli`
- Position in epic: foundation feature for prompt construction and execution.

## Foundation References

- `docs/SPEC.md` — invocation modes and Codex CLI strategy.
- `docs/CONTRACT.md` — CLI synopsis, `--prompt-file`, and `--cwd`.

## Design Decisions

- **Task input forms**: Support CLI args, stdin, and `--prompt-file`.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->

