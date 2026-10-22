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

- `codex exec` for Codex, including GPT-6 Astra, Luna, and Sol plus GPT-5.6
  Luna, Terra, and Sol selection.
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
`dist/peeragent` build, a temporary source build when Go is available in a source
checkout, and a committed platform binary at `bin/<goos>-<goarch>/peeragent`.
The source path runs the compiled binary directly so its exit codes survive,
then removes its private build directory. An already-launched Unix async child
continues from its loaded executable after cleanup.
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

## Job Messaging

Job mailboxes use the lifecycle lock for appends, snapshots and read cursors.
Inbox delivery holds the lock through stdout writes and acknowledgement, so a
failed write leaves replies unread. Outbound delivery reserves a pending outbox
entry under the lock, invokes Herdr outside it, then appends sent/refused status
with the same message ID. Readers fold that journal into one entry per message;
completion never rescans mailbox history. Command-scoped snapshots share each
job's single history read with its unread notice, status and result output.
This enforces the per-job rate limit across concurrent senders without holding
the lifecycle lock during a subprocess call. A crashed sender's pending entry
reserves its slot until the five-minute window expires.

The launcher captures its Herdr address and binary path in a separate mode-0600
file before starting the tracked worker. Public results never include these
addresses. A shared execution context supplies the job id/directory at the target
process boundary, alongside terminal environment filtering; adapters do not own
mailbox state. Herdr receives a mandatory generated origin/reply header through
its documented `agent prompt <pane> <text>` command, with sanitized body brackets.
The transport is bounded by a ten-second subprocess timeout. Windows uses the
same messaging service and records an explicit unsupported-channel refusal.

Session discovery is shared by the concurrent-resume check and unread-inbox
carryover, avoiding two worktree scans. Jobs are deduplicated by recorded job
ID across directory aliases. The job persists mailbox references, and the worker
reserves unread replies for each attempt. Reserved entries are excluded from
inbox output and unread counts; sparse read state lets later arrivals be read
independently. Only a target that emitted output and did not refuse for capacity
accepts its reservation. Startup failure, cancellation and capacity refusal
release it. The worker writes a terminal attempt outcome before settling its
source mailboxes. Readers recover accepted entries from that record, or release
abandoned reservations when the recorded worker process is gone. No process is
signalled during recovery.
[Messaging](MESSAGING.md) defines the user-visible commands and limits.

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

The shared process runner filters inherited terminal-control addresses at the
moment each target process starts. Each adapter passes the request's deliberate
`--keep-terminal-env` override; saved execution specs preserve it through async
launches and retries. Result metadata records the override. The caller's own
environment and unrelated target settings are preserved. Windows batch argument
transport remains intact. [Terminal environment policy](TERMINAL_ENV.md) contains
the complete variable list and protection scope.

## Blocking Flow

```text
1. Host invokes the `peer` skill with task text.
2. The skill calls `peeragent --agent <target>`.
3. The wrapper starts a tracked worker, as in async mode, and waits for it.
   Blocking text output prints the job id to stderr before waiting. The caller
   retains the native worker handle and collects exit alongside result polling.
   Exit without a terminal result becomes a failed job with worker status/logs,
   preserving already-recorded terminal/cancellation results without a deadline.
4. The target edits, runs commands, and returns a final message.
5. The wrapper formats the terminal result, including metadata.job_id.
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
   facilities when available, or `--wait-any <id>...` for several jobs. The attached waiter prints the terminal result;
   `--status` and `--result` remain fallbacks.
```

Async job truth is local to the repository under `.peeragent/jobs/`. A private
per-user index maps job ids and agent/session pairs to absolute working
directories. Separate atomic files avoid contention between launches. The
CLI consults it before applying the current-directory default; application job
controls use the same lookup when no directory is supplied. Explicit `--cwd`
bypasses lookup. Index write failures warn without preventing delegation.

Codex normalizes explicit capacity error events into a distinct terminal result
while preserving the session id. The known remote-compaction error prefix is
unwrapped before matching the diagnostic; structured terminal outcomes remain
authoritative over diagnostic history. The async child optionally retries that result
with the same session and execution settings, a continuation prompt, and a
bounded retry count and configurable backoff. Each attempt retains its raw log.
The job stays running until the final attempt, and terminal cancellation wins
against later completion. The Codex adapter observes stdout as it arrives,
persisting the session id and latest completed message under the same lock as
cancellation. Explicit terminal events and messages marked with a final phase
start a five-second grace period. A `turn.completed` candidate is revoked by
subsequent turn, item, agent or tool activity; later completion restarts grace.
Status exposes the current candidate while the child is still alive.

The runner keeps stdin copying and stdout/stderr draining inside its cancellable
lifecycle, independently of leader exit. It passes a file-backed stdin to
`Cmd`, so `Cmd.Wait` has no hidden input-copy goroutine to block on. All copying
finishes before the explicit reap. Linux uses
`waitid` with `WNOWAIT`; macOS uses a process-exit kqueue notification. Both
leave the leader unreaped, reserving the group number through its last possible
signal. Group signalling is revoked before the explicit reap. After completion
grace or cancellation, owned-tree cleanup closes the runner's stdin writer and output readers so
retained descriptors cannot stall cleanup. Windows retains its Job Object
and process handle. Ordinary draining has no pipe timeout. Unphased progress
messages never initiate shutdown. Async Unix workers forward cancellation
through their context to the separately owned Codex process group.

Cancellation writes cancelled `job.json` and `result.json` state before
signalling. On unix, it sends SIGTERM to the recorded process group, waits up
to 5 seconds, then sends SIGKILL if the group is still present. Child finish
reloads current state before writing and leaves cancelled terminal state
untouched.

Multi-job wait validates and resolves all references before waiting, then polls
resolved locations without rereading the index. Multi-job status reads each job
once and HEAD once per distinct directory; single-job output retains its result
contract. Explicit `job-id@directory` references override a shared `--cwd`.
Completed messages live in `message.txt`, keeping status reads small even when
an agent returns a long report. Results either include the message or expose its
path, including when a later capacity attempt fails without another message.
Live compact progress records append to `agent.log`; status reads its bounded
8 KiB tail for the latest event timestamp and short message preview. Raw target
logs are written after each attempt, keeping full output separate.

Resumed launches hold a short per-session filesystem lock across the live-job
check and worker launch/PID persistence. The check scans the requested worktree
and directories known to the per-user index, ignoring terminal jobs and exited
recorded processes. It refuses with usage exit 2 and wait/cancel instructions to
prevent interleaved session work; `--allow-concurrent-session` bypasses the check
and lock. Blocking and async CLI calls share this boundary.

## Extension Points

The architecture supports these extensions without changing the user-facing
skill or CLI result semantics:

- Native host-agent adapters when a host exposes reliable completion events.
- Structured output parsing when target CLIs provide reliable schemas.
- Optional worktree mode.
- Richer async status and cancellation.
- Target-specific config profiles.
