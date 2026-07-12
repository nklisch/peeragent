# Architecture

## Overview

Alt Subagent is a plugin-packaged wrapper that lets one coding assistant invoke
another local coding agent or model harness for implementation, research, or
review work.

```text
Host assistant session
  -> host skill or MCP client
    -> CLI adapter or stdio MCP adapter
      -> shared peeragent application services
        -> target CLI / async job store
          -> current repository working tree
```

Supported target CLIs:

- `codex exec` for Codex, including GPT-5.6 Luna, Terra, and Sol selection.
- `agy --print` for Gemini through Antigravity CLI.
- `claude --print` for Claude Code, including the Fable alias.
- `pi --provider zai --model glm-5.2 -p` for Z.AI GLM 5.2 through Pi.

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
.mcp.claude.json
.mcp.codex.json
skills/
  peer/
    SKILL.md
  peer-review/
    SKILL.md
bin/
  peeragent
cmd/
  peeragent/
internal/
  app/       # shared delegation and job-control use cases
  claude/
  codex/
  executil/
  gemini/
  input/
  jobs/
  mcp/       # stdio protocol adapter
  prompt/
  result/
  zai/
docs/
```

The plugin-level contract is the skill set, bundled MCP configuration, and the
`peeragent` executable. The executable has two inbound adapters: the existing
CLI and a local stdio MCP server. Both call the same application services and
return the same domain result semantics. Claude Code and Codex manifests point
to host-specific MCP configuration files so each can resolve the packaged
binary with its own plugin-root variable without a brittle cross-host shim.

Installing and enabling either bundled plugin makes the `peeragent` MCP server
available without a separate global MCP entry. Claude Code reads
`.mcp.claude.json` through `${CLAUDE_PLUGIN_ROOT}`; Codex reads
`.mcp.codex.json` through `${PLUGIN_ROOT}`. Each config contains one local
stdio server and starts `bin/peeragent mcp`. The configs are deliberately
separate because host-root interpolation is not portable between ecosystems.

Host approval policy remains authoritative. `delegate` can change the checkout
and `job_cancel` is destructive; `job_status` and `job_result` are read-only.
The skills prefer these MCP tools when the server is available and fall back to
the bundled wrapper for older hosts or standalone skill use. The host, not the
MCP adapter, owns iterative peer-review orchestration.

## Skill Role

Each skill tells the host:

- When implementation, research, or review delegation is appropriate.
- Which target agent it invokes.
- How to pass arbitrary task text to the wrapper.
- Which flags are available.
- That blocking mode is the default.
- That MCP `delegate` with `async: true` is preferred for work likely to exceed
  the host tool timeout, followed by `job_status` and `job_result`.
- That the wrapper result must be read before responding.
- That the host remains responsible for continuing the user conversation.

The skills are deliberately thin. They do not ask the host to manually inspect
the repository before every delegation, re-plan the target's work, or operate a
large command suite.

## Wrapper Role

`bin/peeragent` is the executable entrypoint. It resolves the compiled Go
binary using the following order: an explicit `PEERAGENT_BIN` override, a local
`dist/peeragent` build, `go run cmd/peeragent` when Go is available in a source
checkout, and a committed platform binary at `bin/<goos>-<goarch>/peeragent`.
If none of those resolve, the shim exits with code `3` and directs the user to
install from the GitHub releases page.

The wrapper's CLI adapter:

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

## MCP Adapter Role

`peeragent mcp` is a stdio transport and protocol adapter. It owns MCP
initialization, tool schemas, request decoding, protocol error mapping, and
stdout purity. It delegates execution and job operations to shared application
services also used by the CLI; it does not invoke the CLI parser in-process or
spawn peeragent recursively.

The adapter exposes blocking delegation and the async lifecycle: launch, status,
result, and cancellation. Tool input schemas derive from the same authoritative
request contract used by CLI validation, while tool results preserve the
existing peeragent result fields. Protocol errors and valid peeragent failure
results remain distinct.

Because stdout carries protocol frames, diagnostics and target process output
must never be written there by server infrastructure. Diagnostics go to stderr;
target output remains captured by the existing execution and logging paths.

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
labels it in the result. Some targets do not have a separate full-access argv:
Gemini's default is `agy --print` scoped with `--add-dir`, and Z.AI runs through
Pi print mode with Pi's normal local tool environment.

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
2. The wrapper resolves the request, writes `job.json` and `prompt.txt`, and
   starts a detached child wrapper as `peeragent --job-run <id> --cwd <cwd>`.
3. On unix, the child starts in a new session so its PID is also the process
   group id recorded in the `pid` sidecar.
4. The child loads the execution spec and prompt from the job directory, runs
   the target CLI, writes `result.json`, updates `job.json`, and removes `pid`.
5. The wrapper returns a job id and log location.
6. The host checks status or result through `--status` or `--result`.
```

Async jobs are local to the repository under `.peeragent/jobs/`.

Cancellation writes cancelled `job.json` and `result.json` state before
signalling. On unix, it sends SIGTERM to the recorded process group, waits up
to 5 seconds, then sends SIGKILL if the group is still present. Child finish
reloads current state before writing and leaves cancelled terminal state
untouched.

## Extension Points

The architecture supports these extensions without changing the user-facing
skills or MCP tool semantics:

- Additional MCP transports behind an explicit security and lifecycle design.
- Structured output parsing when target CLIs provide reliable schemas.
- Optional worktree mode.
- Optional review-after-implementation hooks.
- Richer async status and cancellation.
- Target-specific config profiles.
