# Architecture

## Overview

Alt Subagent is a plugin-packaged wrapper that lets one coding assistant invoke
another local coding agent or model harness for implementation, research, or
review work.

```text
Host assistant session
  -> host skill
    -> peeragent CLI adapter
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
skills/
  peer/
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
  prompt/
  result/
  zai/
docs/
```

The plugin-level contract is the `peer` skill and bundled `peeragent`
executable. Claude Code, Codex, and Pi discover the skill, which resolves and
invokes the plugin-relative wrapper. The wrapper calls shared application services and
returns one domain result contract for blocking delegation and explicit async
job controls.

The plugin intentionally does not register an MCP server. A generic MCP tool
call has no portable completion notification once delegation is detached:
blocking calls compete with host tool timeouts, while async MCP calls depend on
the model polling status and later fetching the result. The skill-driven CLI
path can instead use the host's native background monitors or completion
wake-ups and retrieve a durable peeragent job result. Those lifecycle signals
are host facilities rather than a protocol feature peeragent can emulate.

## Skill Role

The `peer` skill tells the host:

- When implementation, research, or review delegation is appropriate.
- Which target agent it invokes.
- How to pass arbitrary task text to the wrapper.
- Which flags are available.
- That `--async` is preferred for substantive work, with native host monitors
  or completion wake-ups used when available.
- That the terminal job result must be read before the host concludes.
- That the host remains responsible for continuing the user conversation.

The skill is deliberately thin. It does not ask the host to manually inspect
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

Default execution starts in the current checkout with the strongest autonomous
mode each target can provide without removing its available sandbox.

Full-access execution is explicit. The wrapper exposes it as `--full-access` and
labels it in the result. Gemini print mode cannot answer permission prompts, so
both Gemini paths use `--dangerously-skip-permissions` for autonomous tool use.
The default also enables agy's terminal sandbox; full access removes that
terminal containment. Antigravity's sandbox does not confine direct file tools,
so Gemini is a trusted-local-agent target in either mode. Z.AI has no separate
full-access argv and runs through Pi print mode with Pi's normal local tool
environment.

Worktree execution is recognized but not implemented yet.

## Blocking Flow

```text
1. Host invokes the `peer` skill with task text.
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
6. The host starts `peeragent --wait <id>` through native monitor/completion
   facilities when available. The attached waiter prints the terminal result;
   `--status` and `--result` remain fallbacks.
```

Async jobs are local to the repository under `.peeragent/jobs/`.

Cancellation writes cancelled `job.json` and `result.json` state before
signalling. On unix, it sends SIGTERM to the recorded process group, waits up
to 5 seconds, then sends SIGKILL if the group is still present. Child finish
reloads current state before writing and leaves cancelled terminal state
untouched.

## Extension Points

The architecture supports these extensions without changing the user-facing
skill or CLI result semantics:

- Native host-agent adapters when a host exposes reliable completion events.
- Structured output parsing when target CLIs provide reliable schemas.
- Optional worktree mode.
- Richer async status and cancellation.
- Target-specific config profiles.
