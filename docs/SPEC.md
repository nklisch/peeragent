# Specification

## Name

The repository, bundled wrapper, and plugin name are `alt-subagent`.

The Claude-facing skills are:

```text
/codex-implement <task text>
/gemini-implement <task text>
```

The Codex-facing skills are:

```text
claude-implement
gemini-implement
```

All skills call the same wrapper with an explicit target:

```text
alt-subagent --agent codex <task text>
alt-subagent --agent gemini <task text>
alt-subagent --agent claude <task text>
```

## Runtime Context

Alt Subagent runs on a developer machine with:

- A repository or working directory open in the host agent.
- A platform-compatible `alt-subagent` wrapper binary supplied by the plugin.
- At least one target CLI installed and authenticated locally.

Supported target CLIs:

- Codex CLI for `--agent codex`.
- Antigravity CLI (`agy`) for `--agent gemini`.
- Claude Code CLI for `--agent claude`.

The plugin uses local target installations and local authentication state. It
does not bundle a separate target-agent runtime or account system.

## Components

The project contains:

- `.claude-plugin/plugin.json` for Claude Code plugin metadata.
- `.codex-plugin/plugin.json` for Codex plugin metadata.
- `skills/` for host-facing skills. The Codex plugin manifest points here so
  Codex can target Claude or Gemini, and Claude Code also discovers the same
  directory.
- `bin/alt-subagent` as the executable host agents call.
- `cmd/alt-subagent/` and internal Go packages for the wrapper implementation.
- `docs/` for foundation documents.

## Invocation Modes

### Blocking

Blocking invocation is the default. The host calls the wrapper and waits until
the target agent completes.

### Async

Async invocation is explicit. The wrapper starts a tracked local job, records
logs and status, and returns a job id. The host can later inspect the task
through `--status`, `--result`, or `--cancel`.

### Full Access

Full access is explicit. The caller must request it with `--full-access`.
Full access is appropriate only when the user intentionally trusts the
repository and machine context.

## Execution Defaults

The default execution uses the same checkout and working tree as the host. It
does not create a git worktree or sandboxed copy.

Default target invocations:

```text
codex exec --cd <repo> --sandbox workspace-write --ask-for-approval on-request ...
agy --print --sandbox --add-dir <repo> ...
claude --print --permission-mode auto --add-dir <repo> ...
```

Full-access target invocations:

```text
codex exec --dangerously-bypass-approvals-and-sandbox ...
agy --print --dangerously-skip-permissions ...
claude --print --dangerously-skip-permissions ...
```

The default Codex and Claude reasoning effort is `medium`; the wrapper exposes
`medium` and `high` through `--effort`.

## Output Requirements

The wrapper returns a concise result to the host. The result includes:

- Overall status.
- Human-readable summary.
- Changed files when known.
- Verification commands and outcomes when known.
- Target-agent final output or useful diagnostics.
- Failure reason and useful log excerpts when the target fails.
- Agent, working directory, access, effort, and job metadata when available.

The wrapper avoids dumping long raw logs into host context unless the task
failed and the logs are needed to continue.

## Safety Boundaries

The project treats same-checkout execution and no-sandbox execution as different
decisions.

Same-checkout execution is the default. No-sandbox or full-access execution is
not.

The wrapper keeps the user in control by making high-risk modes explicit and
reporting what the target agent did. It does not pretend that a target agent's
approval classifier is a security guarantee.

## Non-Goals

The project does not provide:

- A full agent job dashboard.
- A replacement for Codex, Claude Code, or Antigravity CLI.
- A replacement for host-agent permissions.
- A required worktree workflow.
- A general multi-agent planning framework.
