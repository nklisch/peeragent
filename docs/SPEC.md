# Specification

## Name

The repository, bundled wrapper, and plugin name are `peeragent`.

The host-facing skill is:

```text
/peer <task text>
```

The skill calls the wrapper with an explicit target when the user or host
selects one:

```text
peeragent --agent codex <task text>
peeragent --agent gemini <task text>
peeragent --agent claude <task text>
peeragent --agent zai <task text>
```

## Runtime Context

Alt Subagent runs on a developer machine with:

- A repository or working directory open in the host agent.
- A platform-compatible `peeragent` binary. The plugin ships prebuilt binaries
  committed at `plugin/bin/<goos>-<goarch>/peeragent` for linux amd64, linux
  arm64, darwin amd64, and darwin arm64. On other platforms, install manually
  from the GitHub releases page (https://github.com/nklisch/peeragent/releases)
  by downloading the matching asset and either setting `PEERAGENT_BIN` to its
  path or placing it at `<plugin>/bin/<goos>-<goarch>/peeragent`.
- At least one target CLI installed and authenticated locally.

Supported target CLIs:

- Codex CLI for `--agent codex`.
- Antigravity CLI (`agy`) for `--agent gemini`.
- Claude Code CLI for `--agent claude`.
- Pi CLI for `--agent zai`, fixed to Z.AI GLM 5.2.

The plugin uses local target installations and local authentication state. It
does not bundle a separate target-agent runtime or account system.

## Components

The project contains:

- `.claude-plugin/plugin.json` for Claude Code plugin metadata.
- `.codex-plugin/plugin.json` for Codex plugin metadata.
- `package.json` for the Pi package manifest, which loads the bundled `peer`
  skill from `./plugin/skills`.
- `skills/` for the host-facing skill. The Codex plugin manifest points here so
  Codex can target Claude, Gemini, or Z.AI GLM 5.2, and Claude Code also
  discovers the same directory.
- `bin/peeragent` as the executable shim host agents call.
- `plugin/bin/<target>/peeragent` — committed prebuilt binaries for each
  supported platform, included in the marketplace plugin artifact.
- `cmd/peeragent/` and internal Go packages for the wrapper implementation.
- `docs/` for foundation documents.

## Invocation Modes

### Blocking

Blocking invocation is the default. The host calls the wrapper and waits until
the target agent completes. It uses a tracked worker like async mode; every
launch has a job id in JSON and text, including resumed calls.

### Async

Async invocation is explicit at the CLI and preferred by the `peer` skill for
substantive work. The wrapper starts a tracked local job, records logs and
status, and returns a job id. The host should run `--wait <job-id>` through
native monitor or completion facilities when available; the attached waiter
prints the terminal result. `--status`, `--result`, and `--cancel` remain
explicit controls. `--wait-any <job-id>...` waits for the first terminal result
among a validated set; multi-id `--status` provides compact JSONL or a text
table. Each reference may include `@directory` for an unindexed job.

Unknown options and empty or missing values fail as usage errors before work
starts. A `--` separator permits literal task text starting with a dash.

Codex capacity stops return `capacity_exhausted` and retain the target session.
Async callers may opt into `--retry-capacity N` additional attempts with
`--capacity-backoff <duration>` (default ten minutes). Retries resume that
session under the original job id. A `max_output_tokens` stop reports
`output_limit`; the same budget permits one immediate, concise continuation
per job. Other failures are never retried.

`peeragent --jobs [--cwd DIR] [--all] [--json | --text]` lists every job in
that directory, including finished jobs and unindexed local jobs, newest first.
`--all` also includes directories in the per-user job index; removed directories
are skipped. It does not search arbitrary directories. JSON output is one object
per line, using the multi-job status fields: job id, agent, model, status,
`started_at` (creation time), `last_message_at` when available, and
`agent_session`. An empty JSON list produces no lines; `--text` prints a table.
`peeragent --version` prints the running binary's version. Status JSON carries
that version in `metadata.version` for one job and `version` in each multi-job
or listing row, even when the job was started by an older binary.

### Resume

`--resume <agent-session>` continues a target session in its indexed directory
unless `--cwd` overrides it. Each omitted model or effort inherits from the
session's most recent peeragent job; an explicit flag overrides that option.
If the previous job record is unavailable, omitted options keep current defaults
and launch/result metadata carries `resume_warning`. Resolved model and effort
are saved with the job and exposed in launch/result metadata. No model override
for Codex or Claude means the native CLI default.

### Job messaging

Jobs receive `PEERAGENT_JOB_ID` and `PEERAGENT_JOB_DIR` and a standard messaging
preamble, unless `--no-preamble` requests task text only. `peeragent tell <job-id>
[--cwd DIR] "<text>"` appends launcher replies to running jobs. `peeragent inbox`
prints and acknowledges unread replies; commands inside a job announce unread
counts on stderr. Resumes carry unread replies into the resumed prompt.

Use `peeragent say` only to ask a question you can't settle yourself, never for status or progress updates.
Your final handback carries the final status, so don't repeat it in a message.
After asking a question, watch your inbox for the answer: run `peeragent inbox` every few minutes, between steps,
until it arrives, and keep doing independent work meanwhile.

`peeragent say "<text>"` sends a mandatory labelled message through the recorded
launcher's Claude Code native inbox or a parent peer job's mailbox, separate
from typed input. It strips header brackets from the body, bounds text to
2,000 characters and three delivery attempts per five minutes per job, and logs
sent, queued or refused attempts. Refusals name the handback alternative. Without
a safe native channel, or on a transport failure, it queues the message in the
existing outbox and reports that receipt; there is no automatic retry or
terminal-input fallback. Status exposes mailbox counts, oldest unread age, and messaging
evidence source. Missing legacy flags are inferred from a saved preamble or
`say` entry, rather than being treated as disabled. Results include the
ordered conversation after the final message. See [Messaging](MESSAGING.md).

### Sandbox

Sandbox execution is the default target CLI mode. The caller may pass
`--sandbox` explicitly to choose the same mode used when no access flag is
supplied. Gemini combines the native terminal sandbox, `accept-edits`, and
`--dangerously-skip-permissions`: print mode cannot answer permission prompts,
so auto-approval is required to run tests autonomously. Antigravity's sandbox
contains terminal processes but not its direct file tools, which may write
outside the workspace when permissions are skipped. Pi has no
peeragent-specific sandbox/full-access toggle and runs with Pi's normal local
tool environment.

### Full Access

Full access is explicit. The caller must request it with `--full-access`.
Full access is appropriate only when the user intentionally trusts the
repository and machine context.

## Execution Defaults

The default execution uses the same checkout and working tree as the host. It
does not create a git worktree or sandboxed copy.

Target invocations (Codex shown with a selected model tier):

```text
codex exec --json --cd <repo> --sandbox workspace-write \
  --model <gpt-6.1-sol|gpt-6-<astra|luna|sol>|gpt-5.6-<luna|terra|sol>> \
  -c approval_policy="on-request" -c approvals_reviewer="auto_review" ...
agy --output-format json --model gemini-3.7-flash --effort high \
  --mode accept-edits --sandbox --dangerously-skip-permissions \
  --add-dir <repo> --print <prompt>
claude --print --output-format json --permission-mode auto --add-dir <repo> ...
pi --provider zai --model glm-5.2 --thinking <effort> --no-session -p ...
```

Full-access target invocations:

```text
codex exec --json --dangerously-bypass-approvals-and-sandbox ...
agy --dangerously-skip-permissions --print <prompt>
claude --print --dangerously-skip-permissions ...
```

Pi has no separate full-access argv; `--full-access` is recorded in metadata but
Z.AI still runs through the same Pi print-mode surface.

Codex reasoning effort defaults to `high`; the wrapper exposes `low`, `medium`,
`high`, and `xhigh` for Codex. Its `astra`, `luna`, `terra`, and `sol` aliases
normalize to `gpt-6-astra`, `gpt-6-luna`, `gpt-5.6-terra`, and
`gpt-6.1-sol` and pass through to the Codex CLI. Explicit `gpt-6-sol` and
GPT-5.6 Luna, Terra, and Sol IDs remain supported. GPT-6 is the recommended family for routine Codex
work: Luna at high is the routine fast path, Luna at xhigh handles larger
workloads, Terra is an optional middle bridge, Sol at low or medium is roughly
Opus-tier, and Sol at high or xhigh is roughly Fable-tier. GPT-6 Astra is the
flagship tier above Sol for the most demanding passes. Callers can jump
directly from Luna to Sol or Astra.

Gemini defaults to Gemini 3.7 Flash at `high` effort and accepts `low`,
`medium`, and `high`. `flash` selects Gemini 3.7 Flash and `pro` selects Gemini
3.1 Pro; explicit supported Flash family IDs for 3.5 through 3.7 are also
accepted. Pro supports `low` and `high`, not `medium`. Peeragent passes the
normalized model and effort to `agy`. Z.AI GLM
5.2 defaults to `high` and maps `medium`, `high`, and `xhigh` to Pi `--thinking`.
Claude reasoning effort defaults to `xhigh` and exposes `high` and `xhigh`.
Claude supports `--model fable`, `--model sonnet`, `--model opus`, and `--model
haiku`, which pass through to Claude Code. Z.AI is treated as fixed `glm-5.2`;
peeragent rejects every other Z.AI model name even if Pi lists it.

## Session Continuity

Blocking and async results include `metadata.agent_session` when the target
exposes a reliable session id. Hosts may pass that value back with
`--resume <agent-session>` to continue the same target-agent conversation.

Job controls locate newly launched jobs by id through a per-user working-directory
index, regardless of the caller's current directory. Resume defaults to the
last recorded directory for that agent/session pair. Explicit `--cwd` overrides
either lookup; older unindexed jobs remain accessible with `--cwd`.

Session resume is intended for continuity inside a single multi-pass workflow
such as peer review. A resume refuses with exit `2` if a running job for the
same agent/session has a live recorded process; the error names the job and
wait/cancel commands. `--allow-concurrent-session` permits intentional overlap. Resume should not
be used when the user wants an independent second opinion, because resumed sessions carry forward the prior critique and
can anchor later passes.

Codex exposes session ids through JSONL output and resumes with
`codex exec resume`. Claude exposes session ids through JSON output and resumes
with `--resume`. Antigravity's JSON print envelope exposes a `conversation_id`,
which peeragent captures and resumes with `--conversation`. Pi exposes
`--session` for a known session id, but the wrapper does not scrape logs to
capture new Pi session ids. Fresh Z.AI calls use `--no-session`.

## Output Requirements

The wrapper returns a concise result to the host. The result includes:

- Overall status.
- Human-readable summary.
- Changed files when known.
- Verification commands and outcomes when known.
- Target-agent final output or useful diagnostics.
- Failure reason and useful log excerpts when the target fails.
- Agent, working directory, access, effort, model, target session, and job
  metadata when available.

For Codex JSONL output, the default visible detail is the latest completed
`agent_message`, not the full stream of intermediate assistant messages. Hosts
can use `metadata.agent_session` for continuity and `metadata.log_path` for raw
target stdout/stderr inspection when output exists. Codex session ids and completed
messages are persisted while running; result includes untruncated output or a
`metadata.message_path` pointer after failures, cancellations and capacity retries.
Both status forms include the latest event time and a short one-line message
preview from a bounded tail of the live `agent.log`. Capacity results/status
explain the stop and how to resume before displaying the last agent message.
The known remote-compaction diagnostic prefix also qualifies for capacity.
Explicit terminal stream events start a five-second grace period before the
owned target tree is stopped if it has not exited. Progress messages alone do
not start shutdown.

The wrapper avoids dumping long raw logs into host context unless the task
failed and the logs are needed to continue.

Target launches strip inherited terminal-control addresses by default; the
explicit `--keep-terminal-env` override is recorded in job metadata. See the
[terminal environment policy](TERMINAL_ENV.md) for its complete variable list
and scope across targets, launch modes and platforms.

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
- A replacement for Codex, Claude Code, Antigravity CLI, or Pi.
- A replacement for host-agent permissions.
- A required worktree workflow.
- A general multi-agent planning framework.
