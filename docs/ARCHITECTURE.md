# Architecture

## Overview

Alt Subagent is a plugin-packaged wrapper that lets one coding assistant invoke
another local coding agent for implementation, research, or review work.

```text
Host assistant session
  -> host skill
    -> bin/peeragent
      -> dist/peeragent or go run cmd/peeragent
        -> target CLI
          -> current repository working tree
```

Supported target CLIs:

- `codex exec` for Codex.
- `agy --print` for Gemini through Antigravity CLI.
- `claude --print` for Claude Code.

The host decides when delegation is useful. The skill constrains host behavior
around delegation. The wrapper constrains process execution, prompt
construction, result capture, logging, and return formatting. The target agent
performs the implementation, research, or review work.

## Plugin Layout

```text
.claude-plugin/
  plugin.json
.codex-plugin/
  plugin.json
skills/
  codex-implement/
    SKILL.md
  claude-implement/
    SKILL.md
  gemini-implement/
    SKILL.md
bin/
  peeragent
cmd/
  peeragent/
internal/
  claude/
  codex/
  executil/
  gemini/
  input/
  jobs/
  prompt/
  result/
docs/
```

The plugin-level contract is the skill set and the `peeragent` executable.

## Skill Role

Each skill tells the host:

- When implementation, research, or review delegation is appropriate.
- Which target agent it invokes.
- How to pass arbitrary task text to the wrapper.
- Which flags are available.
- That blocking mode is the default.
- That the wrapper result must be read before responding.
- That the host remains responsible for continuing the user conversation.

The skills are deliberately thin. They do not ask the host to manually inspect
the repository before every delegation, re-plan the target's work, or operate a
large command suite.

## Wrapper Role

`bin/peeragent` is the executable entrypoint. It invokes the compiled Go
wrapper from `dist/peeragent` when present and falls back to `go run` during
development.

The wrapper:

- Parses flags and arbitrary task text.
- Resolves the working directory.
- Selects the target agent backend.
- Checks that the target CLI is available.
- Builds a target-specific prompt.
- Chooses the target permission mode.
- Runs the target CLI.
- Captures final output and useful diagnostics.
- Emits a concise result for the host.
- Stores async job metadata when async mode is used.

## Prompt Construction

The wrapper sends the target agent a task prompt that preserves the requested
task text and adds stable execution instructions:

- Work in the current repository.
- Make requested code changes directly when the task calls for edits.
- Follow project instructions discovered by the target.
- Run relevant verification or inspection commands.
- Keep the final answer concise.
- Report changed files, verification or inspection status, and blockers.
- Stop and report blockers instead of guessing around missing credentials or
  unsafe operations.

The host's task text remains the primary input. The wrapper does not impose a
rigid schema on the delegated task request.

## Permission Model

Default execution uses the current checkout with the target CLI's bounded mode
where one is available.

Full-access execution is explicit. The wrapper exposes it as `--full-access` and
labels it in the result.

Worktree execution is recognized but not implemented yet.

## Blocking Flow

```text
1. Host invokes a skill with task text.
2. The skill calls `peeragent --agent <target>`.
3. The wrapper runs the target CLI in the current working directory.
4. The target edits, runs commands, and returns a final message.
5. The wrapper formats the result.
6. The host reads the result and continues the session.
```

## Async Flow

```text
1. Host invokes `peeragent --agent <target> --async <task>`.
2. The wrapper creates a local job record.
3. The wrapper starts a detached child wrapper process.
4. The wrapper returns a job id and log location.
5. The host checks status or result through `--status` or `--result`.
```

Async jobs are local to the repository under `.peeragent/jobs/`.

## Extension Points

The architecture supports these extensions without changing the user-facing
skills:

- Structured output parsing when target CLIs provide reliable schemas.
- Optional worktree mode.
- Optional review-after-implementation hooks.
- Richer async status and cancellation.
- Target-specific config profiles.
