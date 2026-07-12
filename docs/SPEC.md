# Specification

## Name

The repository, bundled wrapper, and plugin name are `peeragent`.

The host-facing skills are:

```text
/peer <task text>
/peer-review [review context]
```

MCP-capable hosts may instead start `peeragent mcp` as a local stdio server and
invoke its delegation and async-job tools. The MCP adapter and CLI share the
same request validation, execution, job store, and result contracts.

Both skills call the same wrapper with an explicit target when the user or host
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
- `skills/` for host-facing skills. The Codex plugin manifest points here so
  Codex can target Claude, Gemini, or Z.AI GLM 5.2, and Claude Code also
  discovers the same directory.
- `bin/peeragent` as the executable shim host agents call.
- `plugin/bin/<target>/peeragent` — committed prebuilt binaries for each
  supported platform, included in the marketplace plugin artifact.
- `cmd/peeragent/` and internal Go packages for the wrapper implementation.
- An MCP stdio adapter that exposes shared application services without writing
  non-protocol output to stdout.
- `docs/` for foundation documents.

## Invocation Modes

### MCP stdio

`peeragent mcp` runs a Model Context Protocol server over stdin/stdout. It
supports protocol initialization and tool discovery, blocking delegation,
async delegation launch, job status, result retrieval, and cancellation. It
does not listen on a network socket, forward arbitrary MCP server configuration
to target agents, or encode the iterative `/peer-review` workflow.

### Blocking

Blocking invocation is the default. The host calls the wrapper and waits until
the target agent completes.

### Async

Async invocation is explicit. The wrapper starts a tracked local job, records
logs and status, and returns a job id. The host can later inspect the task
through `--status`, `--result`, or `--cancel`.

### Sandbox

Sandbox execution is the default bounded target CLI mode. The caller may pass
`--sandbox` explicitly to choose the same mode that is used when no access flag
is supplied. Each target maps this to its own bounded mode (see below). Gemini
through Antigravity is the exception: `agy` has no usable sandbox flag in print
mode (it loops until timeout), so its bounded default is `agy --print` scoped
only by `--add-dir`. Pi is also an exception: it has no peeragent-specific
sandbox/full-access toggle, so the Z.AI target runs in Pi print mode with Pi's
normal local tool environment.

### Full Access

Full access is explicit. The caller must request it with `--full-access`.
Full access is appropriate only when the user intentionally trusts the
repository and machine context.

## Execution Defaults

The default execution uses the same checkout and working tree as the host. It
does not create a git worktree or sandboxed copy.

Target invocations (Codex shown with a selected GPT-5.6 tier):

```text
codex exec --json --cd <repo> --sandbox workspace-write \
  --model gpt-5.6-<luna|terra|sol> \
  -c approval_policy="on-request" -c approvals_reviewer="auto_review" ...
agy --print --add-dir <repo> ...
claude --print --output-format json --permission-mode auto --add-dir <repo> ...
pi --provider zai --model glm-5.2 --thinking <effort> --no-session -p ...
```

Full-access target invocations:

```text
codex exec --json --dangerously-bypass-approvals-and-sandbox ...
agy --print --dangerously-skip-permissions ...
claude --print --dangerously-skip-permissions ...
```

Pi has no separate full-access argv; `--full-access` is recorded in metadata but
Z.AI still runs through the same Pi print-mode surface.

Codex reasoning effort defaults to `high`; the wrapper exposes `low`, `medium`,
`high`, and `xhigh` for Codex. Its `luna`, `terra`, and `sol` aliases normalize
to the corresponding `gpt-5.6-*` IDs and pass through to the Codex CLI. GPT-5.6
is the recommended family for all Codex work: Luna at high is the routine fast
path, Luna at xhigh handles larger workloads, Terra is an optional middle
bridge, Sol at low or medium is roughly Opus-tier, and Sol at high or xhigh is
roughly Fable-tier. Callers can jump directly from Luna to Sol.

Z.AI GLM 5.2 defaults to `high` and maps `medium`, `high`, and `xhigh` to Pi
`--thinking`. Claude reasoning effort defaults to `xhigh` and exposes `high`
and `xhigh`. Claude supports `--model fable`, `--model sonnet`, `--model opus`,
and `--model haiku`, which pass through to Claude Code. Gemini is treated as
fixed Gemini 3.5; `--model gemini-3.5` is accepted for explicit metadata, but
the wrapper does not pass a model flag to `agy` because `agy --print` does not
expose a non-interactive model option. Z.AI is treated as fixed `glm-5.2`;
peeragent rejects every other Z.AI model name even if Pi lists it.

## Session Continuity

Blocking and async results include `metadata.agent_session` when the target
exposes a reliable session id. Hosts may pass that value back with
`--resume <agent-session>` to continue the same target-agent conversation.

Session resume is intended for continuity inside a single multi-pass workflow
such as peer review. It should not be used when the user wants an independent
second opinion, because resumed sessions carry forward the prior critique and
can anchor later passes.

Codex exposes session ids through JSONL output and resumes with
`codex exec resume`. Claude exposes session ids through JSON output and resumes
with `--resume`. Antigravity exposes `--conversation` for a known conversation
id, and Pi exposes `--session` for a known session id, but the wrapper does not
scrape logs to capture new Gemini or Pi session ids. Fresh Z.AI calls use
`--no-session`.

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
target stdout/stderr inspection when output exists.

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

- A remote or Streamable HTTP MCP service.
- An arbitrary MCP proxy for target agents.
- A first-class MCP tool for iterative peer-review orchestration.
- A full agent job dashboard.
- A replacement for Codex, Claude Code, Antigravity CLI, or Pi.
- A replacement for host-agent permissions.
- A required worktree workflow.
- A general multi-agent planning framework.
