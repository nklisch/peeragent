# Architecture

## Overview

Codex Implement is a Claude Code plugin that bridges Claude Code to Codex CLI through a small wrapper command.

```text
Claude Code session
  -> codex-implement skill
    -> bin/codex-implement
      -> scripts/codex-implement runtime
        -> codex exec
          -> current repository working tree
```

Claude decides when delegation is useful. The skill constrains Claude's behavior around delegation. The wrapper constrains process execution, prompt construction, result capture, logging, and return formatting. Codex performs the implementation work.

## Plugin Layout

```text
.claude-plugin/
  plugin.json
skills/
  codex-implement/
    SKILL.md
bin/
  codex-implement
scripts/
  codex-implement.mjs
  lib/
docs/
  VISION.md
  SPEC.md
  ARCHITECTURE.md
  CONTRACT.md
```

The exact script decomposition is internal to the wrapper. The plugin-level contract is the skill and the `codex-implement` executable.

## Skill Role

`skills/codex-implement/SKILL.md` tells Claude:

- When implementation delegation is appropriate.
- How to pass arbitrary task text to the wrapper.
- Which flags are available.
- That blocking mode is the default.
- That Codex output should be returned or summarized according to the wrapper result.
- That Claude remains responsible for continuing the user conversation after Codex returns.

The skill is deliberately thin. It does not ask Claude to manually inspect the repository before every delegation, re-plan Codex's work, or operate a command suite. Its job is to make delegation natural and reliable.

## Wrapper Role

`bin/codex-implement` is the executable entrypoint. It delegates to the implementation under `scripts/`.

The wrapper:

- Parses flags and arbitrary task text.
- Resolves the working directory.
- Checks that Codex CLI is available.
- Builds the Codex prompt.
- Chooses the Codex permission mode.
- Runs Codex.
- Captures final output and useful diagnostics.
- Emits a concise result for Claude.
- Stores async job metadata when async mode is used.

## Prompt Construction

The wrapper sends Codex a task prompt that preserves Claude's requested implementation text and adds stable execution instructions:

- Work in the current repository.
- Make the requested code changes directly.
- Follow project instructions discovered by Codex.
- Run relevant verification.
- Keep the final answer concise.
- Report changed files and verification status.
- Stop and report blockers instead of guessing around missing credentials or unsafe operations.

Claude's task text remains the primary input. The wrapper does not impose a rigid schema on the implementation request.

## Permission Model

Default execution uses the current checkout with classifier-compatible Codex permissions. This gives Codex direct access to the same working tree while preserving Codex approval review at policy boundaries.

Full-access execution is explicit. The wrapper exposes it as an option for trusted contexts and labels it clearly in the result.

Worktree execution is optional and explicit. It is a task-level escape hatch, not the default architecture.

## Blocking Flow

```text
1. Claude invokes the skill with task text.
2. The skill calls `codex-implement`.
3. The wrapper runs `codex exec` in the current working directory.
4. Codex edits, runs commands, and returns a final message.
5. The wrapper formats the result.
6. Claude reads the result and continues the session.
```

Blocking mode does not create persistent job state beyond optional logs.

## Async Flow

```text
1. Claude invokes `codex-implement --async <task>`.
2. The wrapper creates a local job record.
3. The wrapper starts Codex detached from Claude's blocking tool call.
4. The wrapper returns a job id and log location.
5. Claude checks status or result through `codex-implement --status` or `--result`.
```

Async jobs are local to the repository or plugin data directory. Async output is concise by default and log-backed for diagnostics.

## Extension Points

The architecture supports these extensions without changing the user-facing skill:

- Codex app-server transport for streamed events and resumable threads.
- Structured output schemas once stable enough for reliable final result parsing.
- Optional worktree mode.
- Optional review-after-implementation hooks.
- Richer async status and cancellation.
- Project-specific Codex config profiles.

These are extensions, not separate product surfaces.

