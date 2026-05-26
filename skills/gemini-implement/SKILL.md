---
name: gemini-implement
description: >
  Delegate implementation work to Gemini through Google Antigravity CLI using
  the bundled alt-subagent wrapper. Use when Claude Code or Codex should hand
  arbitrary implementation text to Gemini as an autonomous implementor in the
  current repository. Treat this wrapper target as fixed Gemini 3.5; only
  --model gemini-3.5 is accepted for explicit model metadata.
allowed-tools: Bash
metadata:
  short-description: Delegate to Gemini
---

# Gemini Implement

Use this skill when implementation work should be delegated to Gemini through
the local Antigravity CLI while the current host assistant remains responsible
for the user conversation.

## Default Behavior

Pass the user's implementation request to the bundled wrapper:

```bash
alt-subagent --agent gemini "$ARGUMENTS"
```

The wrapper runs in the current repository and returns JSON by default. Read
that JSON before responding to the user.

## Delegation Contract

The host delegates implementation intent; Gemini performs the implementation
pass. The task text is arbitrary natural language, not shell syntax. Do not
split the request into many wrapper calls unless the user explicitly asked for
separate implementation passes.

The default call is blocking. Wait for the command to return, then summarize
the result using the wrapper's status, summary, changed files, verification
details, and failure information.

## Result Handling

- If `status` is `success`, report what changed and what verification ran.
- If `status` is `blocked`, explain the blocker and continue from the host side.
- If `status` is `failed`, surface the failure reason and useful log details.
- If `status` is `running`, report the async job id and how to check it.
- Do not claim implementation success unless the wrapper reports success.

## Options

Use advanced modes only when the request calls for them:

- `--full-access` for explicit trusted full-access execution.
- `--worktree` for explicit isolated worktree execution.
- `--model gemini-3.5` to explicitly request the fixed Gemini 3.5 target. This
  is recorded in wrapper metadata; `agy --print` currently exposes no
  non-interactive model flag for the wrapper to pass through.
- `--async` for long-running work where the host should not block.
- `--prompt-file <path>` for large prompts.

Full access is never implied by ordinary delegation. If the wrapper reports that
full access is needed and the user did not already request it, ask before
retrying with `--full-access`.

## Guardrails

- Keep the handoff concise; Gemini receives the implementation task, not a full
  transcript recap unless needed.
- Do not run Gemini repeatedly in a loop after failed or blocked results.
- Preserve the host assistant's responsibility for explaining outcomes to the
  user.
- Treat the repository working tree as shared space with the host assistant.
