---
name: claude-implement
description: >
  Delegate implementation, research, or review work from Codex to Claude Code
  through the bundled alt-subagent wrapper. Use when Codex should hand arbitrary
  task text to Claude as an autonomous worker in the current repository. Use
  Claude sonnet for normal work, opus plus xhigh effort for deeper work, and
  haiku plus high effort for lightweight work.
metadata:
  short-description: Delegate to Claude
allowed-tools: Bash
---

# Claude Implement

Use this skill when implementation, research, or review work should be
delegated to Claude Code while Codex remains responsible for the user
conversation.

## Default Behavior

Run the bundled wrapper with the Claude backend:

```bash
bin/alt-subagent --agent claude "$ARGUMENTS"
```

The wrapper runs in the current repository and returns JSON by default. Read
that JSON before responding to the user.

## Result Handling

- If `status` is `success`, report what changed and what verification ran.
- If `status` is `failed`, surface the failure reason and useful log details.
- If `status` is `running`, report the async job id and how to check it.
- Do not claim implementation success unless the wrapper reports success.

## Options

- `--full-access` for explicit trusted full-access execution.
- `--model sonnet` for the normal Claude task pass; effort defaults to `xhigh`.
- `--model opus --effort xhigh` for a deeper Claude implementation, research,
  or review pass.
- `--model haiku --effort high` for lightweight or fast Claude work.
- `--effort high` for a lighter Claude pass when `xhigh` is unnecessary.
- `--effort xhigh` for the default/deeper Claude pass.
- `--async` for long-running work.
- `--prompt-file <path>` for large prompts.

Full access is never implied by ordinary delegation.
