---
name: codex-implement
description: >
  Delegate implementation, research, or review work to OpenAI Codex CLI through
  the bundled alt-subagent wrapper. Use when Claude should hand arbitrary task
  text to Codex as an autonomous worker in the current repository. Codex
  defaults to high effort; use xhigh for deeper implementation, research, or
  review passes.
allowed-tools: Bash
---

# Codex Implement

Use this skill when implementation, research, or review work should be
delegated to Codex while Claude remains responsible for the user conversation.

## Default Behavior

Pass the user's task request to the bundled wrapper:

```bash
alt-subagent --agent codex "$ARGUMENTS"
```

The wrapper runs in the current repository and returns JSON by default. Read
that JSON before responding to the user.

## Delegation Contract

Claude delegates task intent; Codex performs the focused task pass. The task
text is arbitrary natural language, not shell syntax. Do not split the request
into many wrapper calls unless the user explicitly asked for separate passes.

The default call is blocking. Wait for the command to return, then summarize
the result using the wrapper's status, summary, changed files, verification
details, and failure information.

## Result Handling

- If `status` is `success`, report what changed and what verification ran.
- If `status` is `blocked`, explain the blocker and continue from Claude's side.
- If `status` is `failed`, surface the failure reason and useful log details.
- If `status` is `running`, report the async job id and how to check it.
- Do not claim implementation success unless the wrapper reports success.

## Options

Use advanced modes only when the request calls for them:

- `--full-access` for explicit trusted full-access execution.
- `--worktree` for explicit isolated worktree execution.
- Omit `--effort` for the default Codex `high` effort.
- `--effort medium` for lightweight or fast tasks.
- `--effort xhigh` for deeper implementation, research, or review passes.
- `--async` for long-running work where Claude should not block.
- `--status <job-id>` to check an async job.
- `--result <job-id>` to fetch an async job's final result.
- `--cancel <job-id>` to stop an async job best-effort.
- `--prompt-file <path>` for large prompts.

Full access is never implied by ordinary delegation. If the wrapper reports that
full access is needed and the user did not already request it, ask before
retrying with `--full-access`.

## Guardrails

- Keep the handoff concise; Codex receives the implementation task, not a full
  transcript recap unless needed.
- Do not run Codex repeatedly in a loop after failed or blocked results.
- Preserve Claude's responsibility for explaining outcomes to the user.
- Treat the repository working tree as shared space with Claude.
