# Contract

## Skill Contract

The `codex-implement` skill accepts arbitrary task text from Claude or the user.

```text
/codex-implement <task text>
```

The skill calls the bundled CLI:

```text
codex-implement <task text>
```

The task text is treated as implementation intent, not as shell syntax. The wrapper is responsible for preserving the text safely when invoking Codex.

## CLI Synopsis

```text
codex-implement [options] <task text>
codex-implement [options] --prompt-file <path>
codex-implement --status <job-id>
codex-implement --result <job-id>
codex-implement --cancel <job-id>
```

## Core Options

`--async`

Start Codex in the background and return a job id.

`--full-access`

Run Codex with full local access. This is explicit because it weakens or removes normal permission boundaries.

`--worktree`

Run Codex in an isolated worktree when supported. This is explicit and not the default.

`--model <name>`

Pass a Codex model override.

`--effort <level>`

Set the Codex reasoning effort. Supported values are `medium` and `high`.
The default is `medium`, which is the standard setting for implementation work.
Use `high` for harder or more complex tasks. Lower and extra-high effort modes
are intentionally not exposed by the wrapper.

`--profile <name>`

Use a Codex configuration profile.

`--cwd <path>`

Set the working directory. Defaults to the current directory from Claude's Bash call.

`--json`

Emit machine-readable wrapper output.

`--prompt-file <path>`

Read task text from a file. This is useful for large prompts.

## Working Directory

The default working directory is the current directory of the Claude Code session. Codex operates in that same checkout. The wrapper does not create a worktree, clone, or sandbox copy unless explicitly requested.

The wrapper passes the resolved directory to Codex with `--cd` or the equivalent app-server field.

## Default Codex Invocation

The default invocation is conceptually:

```text
codex exec --cd <cwd> <constructed prompt>
```

The wrapper layers in permission and output options according to local Codex capabilities and user flags.

The default permission posture keeps Codex Auto-review useful when the local Codex configuration supports it. The wrapper does not default to unchecked full access.

## Result Shape

Human-readable output uses this structure:

```text
Codex Implement: <status>

Summary:
<short summary>

Changed Files:
- <path>

Verification:
- <command>: <passed|failed|not run>

Details:
<Codex final message or concise failure detail>

Metadata:
- cwd: <path>
- mode: blocking|async
- access: default|full-access|worktree
- effort: medium|high
- codex_session: <id if known>
- job_id: <id if async>
```

JSON output uses equivalent fields:

```json
{
  "status": "success",
  "summary": "Implemented the requested change.",
  "changed_files": [],
  "verification": [],
  "details": "",
  "metadata": {
    "cwd": "",
    "mode": "blocking",
    "access": "default",
    "effort": "medium",
    "codex_session": null,
    "job_id": null
  }
}
```

Valid statuses are:

- `success`
- `failed`
- `blocked`
- `cancelled`
- `running`

## Exit Codes

`0`

The command completed successfully. For async launch, this means the job started successfully, not that Codex completed the implementation.

`1`

Codex ran but failed, or the wrapper encountered an expected operational failure.

`2`

Invalid arguments or invalid mode combination.

`3`

Codex CLI is missing, unauthenticated, or incompatible with the requested mode.

`4`

Async job lookup failed.

## Failure Reporting

On failure, the wrapper returns:

- The failing phase.
- The Codex exit status when available.
- A concise stderr excerpt.
- The log path when available.
- A suggested next action for Claude.

The wrapper does not hide partial edits. If Codex changed files before failing, the result says so when the information is available.

## Async Job Contract

Async jobs have:

- A stable job id.
- A status file.
- A log file.
- The original task text or a safe reference to it.
- Start and completion timestamps.
- Codex session metadata when available.

`--status` reports whether the job is running, complete, failed, blocked, or cancelled.

`--result` returns the final result in the same shape as blocking mode.

`--cancel` attempts to stop the local Codex process and marks the job cancelled.

## Claude Continuation Contract

After the wrapper returns, Claude:

- Reads the status first.
- Uses changed files and verification information to decide the next step.
- Does not claim success when the result is failed or blocked.
- Does not rerun Codex automatically in a loop after repeated failures.
- Asks the user before escalating to full access when the original call did not request it.
