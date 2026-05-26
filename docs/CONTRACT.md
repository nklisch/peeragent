# Contract

## Skill Interface

Skills accept arbitrary task text from the host or user.

Claude-facing skills:

```text
/codex-implement <task text>
/gemini-implement <task text>
```

Codex-facing skills:

```text
claude-implement
gemini-implement
```

The task text is treated as implementation intent, not as shell syntax. The
wrapper is responsible for preserving the text safely when invoking the target
agent.

## Wrapper Interface

```text
alt-subagent [options] <task text>
alt-subagent [options] --prompt-file <path>
alt-subagent --status <job-id>
alt-subagent --result <job-id>
alt-subagent --cancel <job-id>
```

Options:

- `--agent <codex|gemini|claude>`: Select the target agent. Defaults to
  `codex`.
- `--async`: Start the target in the background and return a job id.
- `--full-access`: Run the target with full local access.
- `--worktree`: Reserved for future isolated worktree execution.
- `--effort <medium|high>`: Set Codex or Claude reasoning effort.
- `--profile <name>`: Pass a Codex configuration profile.
- `--cwd <path>`: Set the repository root.
- `--prompt-file <path>`: Read task text from a file.
- `--json`: Emit JSON output. This is the default.
- `--text`: Emit human-readable text output.
- `--status <job-id>`: Inspect an async job.
- `--result <job-id>`: Fetch an async job result.
- `--cancel <job-id>`: Cancel an async job.

## Result JSON

The default result is JSON:

```json
{
  "status": "success",
  "summary": "Codex implementation completed",
  "changed_files": [],
  "verification": [],
  "details": "stdout:\n...",
  "metadata": {
    "cwd": "/repo",
    "agent": "codex",
    "access": "default",
    "profile": "",
    "effort": "medium",
    "exit_code": 0,
    "job_id": ""
  }
}
```

`status` values:

- `success`: Target completed successfully.
- `failed`: Target or wrapper failed.
- `blocked`: Reserved for explicit target-reported blockers.
- `running`: Async job started or is still running.
- `cancelled`: Async job was cancelled.

## Text Result

Text output starts with:

```text
Alt Subagent: <status>
```

It includes summary, changed files, verification, details, and metadata sections
when available.

## Working Directory

The default working directory is the current directory of the host session. The
target operates in that same checkout. The wrapper does not create a worktree,
clone, or sandbox copy unless explicitly requested.

## Target Invocation

Default Codex:

```text
codex exec --cd <repo> --sandbox workspace-write --ask-for-approval on-request ...
```

Default Gemini:

```text
agy --print --sandbox --add-dir <repo> --print-timeout 15m ...
```

Default Claude:

```text
claude --print --output-format text --add-dir <repo> --permission-mode auto ...
```

Full access maps to each target's explicit bypass flag.

## Exit Codes

- `0`: Completed successfully. For async launch, this means the job started.
- `1`: Target failed or exited non-zero.
- `2`: Wrapper usage error or unsupported mode.
- `4`: Async job lookup failed.

## Async State

Async state is stored under:

```text
.alt-subagent/jobs/<job-id>/
  job.json
  agent.log
  result.json
```

`--cancel` attempts to stop the local child wrapper process and marks the job
cancelled.

## Guarantees

The wrapper:

- Preserves arbitrary task text as data.
- Does not run full access unless explicitly requested.
- Does not silently switch target agents.
- Does not rerun a failed target automatically in a loop.
- Reports the target exit status when available.
- Reports stdout/stderr excerpts in `details`.

The wrapper does not guarantee that target-agent output is complete,
well-structured, or free of partial edits before failure.
