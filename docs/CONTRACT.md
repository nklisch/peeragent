# Contract

## Skill Interface

The `peer` skill accepts arbitrary task text from the host or user.

The plugin exposes the same skill to Claude Code, Codex, and Pi hosts:

```text
/peer <task text>
```

The task text is treated as delegated task intent, not as shell syntax. The
wrapper is responsible for preserving the text safely when invoking the target
agent.

## Delegation Lifecycle

The skill invokes the wrapper directly and reads its terminal result before
concluding. It prefers `--async` for substantive work, then runs `--wait
<job-id>` through native host monitors or completion wake-ups when available;
short tasks may remain blocking.
The plugin does not register an MCP server because detached MCP calls have no
portable completion notification of their own. CLI job ids keep status and
result retrieval explicit.

## Wrapper Interface

```text
peeragent [options] <task text>
peeragent [options] --prompt-file <path>
peeragent --status <job-id>
peeragent --result <job-id>
peeragent --wait <job-id>
peeragent --cancel <job-id>
```

Task text is resolved from positional arguments, then `--prompt-file`, then
non-interactive stdin. When the wrapper is reading actual process stdin, it
reads stdin only when no positional task text, prompt file, or job-control flag
is present; this avoids accidentally blocking on host-provided stdin when the
task is already explicit. In-memory stdin readers used by tests or embedders
are always read when present.

Options:

- `--agent <codex|gemini|claude|zai>`: Select the target agent. Defaults to
  `codex`. The `zai` target is fixed to Z.AI GLM 5.2 through Pi.
- `--async`: Start the target in the background and return a job id.
- `--sandbox`: Use the default target mode. For Gemini this enables terminal
  containment while tool permissions remain auto-approved for headless autonomy.
- `--full-access`: Remove target sandboxing. Gemini still auto-approves tools in
  either mode; full access additionally removes its terminal sandbox.
- `--worktree`: Reserved for future isolated worktree execution.
- `--effort <low|medium|high|xhigh>`: Set target reasoning effort. Codex
  defaults to `high` and accepts all four levels. Gemini defaults to `high` and
  accepts `low`, `medium`, or `high`. Z.AI defaults to `high` and accepts
  `medium`, `high`, or `xhigh`; Claude defaults to `xhigh` and accepts `high` or
  `xhigh`. Gemini passes effort to `agy`; Z.AI maps it to Pi `--thinking`.
- `--model <luna|terra|sol|fable|sonnet|opus|haiku|flash|pro|glm-5.2>`:
  Select a GPT-5.6 Codex tier, Claude alias, Gemini family, or explicitly record
  the fixed Z.AI GLM 5.2 target. Codex also accepts canonical `gpt-5.6-*` IDs.
  Gemini defaults to `gemini-3.7-flash`; `flash`, `pro`, and supported explicit
  Gemini family IDs normalize to an `agy --model` value. Flash accepts
  `low|medium|high`, while Pro accepts `low|high`. Z.AI accepts only
  `glm-5.2`; no other Z.AI models are surfaced.
- `--profile <name>`: Pass a Codex configuration profile.
- `--resume <agent-session>`: Resume a prior target-agent session when the
  target supports it. Use this for continuity inside one review loop; omit it
  for an independent second opinion.
- `--cwd <path>`: Set the repository root.
- `--prompt-file <path>`: Read task text from a file.
- `--json`: Emit JSON output. This is the default.
- `--text`: Emit human-readable text output.
- `--status <job-id>`: Inspect an async job.
- `--result <job-id>`: Fetch an async job's current or terminal result.
- `--wait <job-id>`: Stay attached until an async job has a terminal result,
  making it suitable for native host process monitors.
- `--cancel <job-id>`: Cancel an async job by marking terminal state. On Unix,
  it sends SIGTERM to the async process group, then sends SIGKILL after a
  5-second grace period when the group is still running.

## Result JSON

The default result is JSON:

```json
{
  "status": "success",
  "summary": "Claude implementation completed",
  "changed_files": [],
  "verification": [],
  "details": "stdout:\n...",
  "metadata": {
    "cwd": "/repo",
    "agent": "claude",
    "access": "default",
    "profile": "",
    "effort": "xhigh",
    "model": "sonnet",
    "agent_session": "session-id",
    "exit_code": 0,
    "job_id": "",
    "log_path": "/repo/.peeragent/runs/..."
  }
}
```

When available, `metadata.agent_session` contains the target-agent session id
that can be passed back with `--resume`.

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

Codex (when a recommended GPT-5.6 model is selected):

```text
codex exec --json --cd <repo> --sandbox workspace-write \
  --model gpt-5.6-<luna|terra|sol> \
  -c approval_policy="on-request" -c approvals_reviewer="auto_review" ...
```

Default Gemini:

```text
agy --output-format json --model gemini-3.7-flash --effort high \
  --mode accept-edits --sandbox --dangerously-skip-permissions \
  --add-dir <repo> --print-timeout 15m --print <prompt>
```

`--print` is a string-valued flag in current `agy`, so peeragent emits it after
all other options. Putting it first causes the next option to be consumed as the
prompt and silently drops the intended model, sandbox, and permission settings.

Default Claude:

```text
claude --print --output-format json --add-dir <repo> --permission-mode auto ...
```

Default Z.AI GLM 5.2 through Pi:

```text
pi --provider zai --model glm-5.2 --thinking <effort> --no-session -p ...
```

For Codex, the short aliases `luna`, `terra`, and `sol` normalize to and pass
through as `gpt-5.6-luna`, `gpt-5.6-terra`, and `gpt-5.6-sol`. The canonical
IDs are also accepted. For Claude, the wrapper passes `--model <alias>` to
Claude Code; accepted aliases are `fable`, `sonnet`, `opus`, and `haiku`.
For Gemini, `flash` normalizes to `gemini-3.7-flash`, `pro` normalizes to
`gemini-3.1-pro`, and the wrapper passes the model and effort to `agy`. When
`--model glm-5.2` is provided for Z.AI, the wrapper records the fixed target and
passes that model to Pi; other Z.AI model names are rejected.

Full access maps to each target's explicit bypass flag where one exists. Gemini
already auto-approves tools in its default print-mode invocation because no
interactive approver exists; full access removes the terminal sandbox. That
sandbox does not contain direct file tools, so Gemini requires trusted local
context in both modes. Pi has no separate peeragent sandbox/full-access toggle;
its target uses Pi print mode with the normal local Pi tool environment.

Resume maps to each target's native resume surface:

```text
codex exec resume <session-id> ...
agy --conversation <conversation-id> --print <prompt>
claude --print --resume <session-id> ...
pi --provider zai --model glm-5.2 --session <session-id> -p ...
```

Codex, Claude, and Gemini sessions are captured from machine-readable target
output. Gemini resumes the captured `conversation_id` through `--conversation`.
Pi can resume a caller-supplied session id, but the wrapper does not scrape logs
to infer new Pi session ids. Fresh Z.AI calls use `--no-session` by default.

Default result output is compact. For Codex JSONL output, peeragent records the
latest completed `agent_message` as the visible stdout detail instead of joining
intermediate assistant messages. Hosts that need more context can resume with
`metadata.agent_session` or inspect the raw target stdout/stderr stored at
`metadata.log_path` when that field is present.

## Exit Codes

- `0`: Completed successfully. For async launch, this means the job started.
- `1`: Target failed or exited non-zero.
- `2`: Wrapper usage error or unsupported mode.
- `3`: Wrapper binary unavailable for this platform.
- `4`: Async job lookup failed.

## Async State

Async state is stored under:

```text
.peeragent/jobs/<job-id>/
  job.json       lifecycle + ExecSpec; written on child finish or by --cancel
  prompt.txt     resolved task text, parent-written, child-read
  pid            child PID/PGID for cancel, present while running
  agent.log      background wrapper stdout+stderr
  target.log     raw target stdout+stderr when available
  result.json    final result, written by child OR by --cancel
```

`--cancel` writes `job.json` and `result.json` as cancelled before signalling.
On Unix, it signals the process group recorded by `pid`. If the `pid` sidecar is
missing, cancellation still records the terminal state and skips signalling.

## Guarantees

The wrapper:

- Preserves arbitrary task text as data.
- Does not run full access unless explicitly requested.
- Does not silently switch target agents.
- Does not rerun a failed target automatically in a loop.
- Reports the target exit status when available.
- Stores raw target stdout/stderr at `metadata.log_path` when output exists.
- Reports compact stdout/stderr details, preferring the final peer message when
  the target exposes one.

The wrapper does not guarantee that target-agent output is complete,
well-structured, or free of partial edits before failure.
