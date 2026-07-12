---
id: epic-wrapper-cli
kind: epic
stage: done
tags: [infra]
parent: null
depends_on: [epic-plugin-foundation]
release_binding: 0.5.0
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Wrapper CLI

## Brief

This epic delivers the blocking `codex-implement` command that Claude calls when delegating implementation work. The wrapper accepts arbitrary task text, resolves the working directory, checks Codex CLI availability, constructs the Codex prompt, and invokes `codex exec` in the current checkout.

The capability delivered here is the core delegation path: Claude calls one command, Codex works in the same repository, and the wrapper returns enough information for Claude to continue. Blocking mode is the default and is treated as the primary product path.

This epic does not finalize the permission model, full result schema, or async job control. It provides the executable foundation those capabilities depend on.

## Foundation References

- `docs/VISION.md` — default execution mode and success criteria.
- `docs/SPEC.md` — Codex CLI strategy and invocation modes.
- `docs/ARCHITECTURE.md` — wrapper role, prompt construction, and blocking flow.
- `docs/CONTRACT.md` — CLI synopsis and working-directory contract.

## Design Decisions

- **Should the wrapper be Node, Go, or compiled Bun?** Use Go for the wrapper CLI. Go gives a durable compiled command with strong process handling and no npm/Bun runtime dependency at execution time. The tradeoff is packaging platform-specific binaries for distributable plugin installs.
- **Why not Node?** Node is easy for plugin scripting and matches some existing Claude/OpenAI plugin examples, but it makes the wrapper feel like a script runtime rather than a standalone CLI and pushes dependency/runtime assumptions onto users.
- **Why not compiled Bun?** Compiled Bun can produce a convenient binary, but it adds a less-standard toolchain and runtime surface for a small process wrapper. Go is the more conservative compiled CLI choice.
- **Should arbitrary task text be accepted as CLI args, stdin, or both?** Support both. CLI args are ergonomic for short calls; stdin and `--prompt-file` are preferred for long prompts.

## Decomposition

Split by the core blocking execution pipeline. Input collection establishes the task text and cwd. Prompt construction turns that task into a stable Codex instruction. Blocking execution discovers Codex and runs `codex exec` with the constructed prompt. Result formatting stays intentionally light here because `epic-result-contract` owns the full output contract.

### Child features

- `epic-wrapper-cli-inputs` — CLI args, stdin, `--prompt-file`, and cwd resolution — depends on: `[]`
- `epic-wrapper-cli-prompt` — autonomous implementation prompt envelope — depends on: `[epic-wrapper-cli-inputs]`
- `epic-wrapper-cli-blocking-exec` — Codex discovery and blocking `codex exec` invocation — depends on: `[epic-wrapper-cli-prompt]`

### Decomposition risks

The main risk is overlapping with the later result-contract and safety-permissions epics. This epic should keep output and permission behavior minimal, with explicit seams for those downstream capabilities.

## Review

Approved. The wrapper CLI now accepts task input, builds a stable Codex prompt, and has a blocking `codex exec` execution path with minimal JSON output. Safety policy refinement and richer result shaping remain correctly isolated in downstream epics.

<!-- The design pass on each child feature will fill in real specifics. -->
