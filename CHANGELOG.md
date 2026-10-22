# Changelog

## v0.10.0

### Features
- `--wait-any <job-id>...` blocks until the first of several jobs finishes and prints that job's terminal result,
  with `--wait`'s exit semantics. `job-id@directory` names an unindexed job. An unknown id is a usage error before
  any wait.
- `--status` accepts several job ids and prints one compact record per job: status, model, effort, session,
  start time, capacity backoff, and head commit. JSON by default, `--text` for a table. Single-job output is
  unchanged.
- A Codex job's session id is recorded as soon as Codex reports it, so `--status`, `--result` and a later `--resume`
  have it while the job runs and after a cancel.
- Terminal results always carry the agent's final message untruncated, or a pointer to the file holding it, across
  running, failed, cancelled and capacity-retried jobs.
- `--retry-capacity` is described beside `--async` in help, the README and the skills.

### Fixes
- An async Codex job that logged its final message and `task_complete`, but never exited, stays `running` no longer.
  A terminal event starts a short completion grace, after which only the job's own process tree is stopped. Any new
  agent activity during the grace cancels it, so a session that continues after `turn.completed` isn't cut off.
- Cleanup keeps ownership of the job's process group until its last signal, so it can never signal a process-group
  number the system has reused.
- Pipe draining has no timeout before completion, so a launcher that starts the agent in the background and exits
  no longer cuts the agent off.
- Cancellation reaches the whole job even when a large prompt is still being written to a descendant that holds stdin.

## v0.9.1

### Fixes
- Windows: concurrent writes to the same job or session index entry retry
  briefly when Windows denies the replacement, instead of warning and leaving
  the entry missing.
- Windows: path-handling tests expect the absolute working directory each
  platform produces, so the Windows test suite passes again.

## v0.9.0

### Features
- `--retry-capacity N` (async Codex jobs) resumes the same session after a
  "model is at capacity" stop, under one job id, waiting `--capacity-backoff`
  (ten minutes by default) between attempts. A capacity stop now ends with its
  own terminal status, `capacity_exhausted`, carrying the session to resume.
- `--wait`, `--status`, `--result`, `--cancel` and `--resume` find a job or
  session from any directory through a private per-user index; `--cwd` still
  wins when given.

### Fixes
- Unknown options and empty option values are usage errors (exit 2) before any
  job starts; previously `--list` or `--background` started a real job. Task
  text that begins with a dash still works after `--`.
- Only a capacity stop is retried: the job's terminal event decides, so an
  ordinary failure after an earlier capacity warning is never retried.
- Codex's "Orphan custom tool call" diagnostics are filtered from the visible
  error output and kept in the raw log.
- The source shim keeps the program's exit code from any working directory.
- Windows: v0.8.2 and v0.8.3's shim launch fixes are included.

## v0.8.1

### Features
- Add GPT-6.1 Sol support for Codex. The `sol` alias now selects
  `gpt-6.1-sol`; explicit GPT-6 and GPT-5.6 Sol IDs remain supported.

## v0.8.0

### Features
- Add GPT-6 Sol and Luna model IDs for Codex. The `sol` and `luna` aliases
  now select GPT-6; explicit GPT-5.6 model IDs remain supported.

## v0.7.0

### Features
- Add GPT-6 Astra support for Codex: `astra` alias and `gpt-6-astra` canonical
  model id, positioned as the flagship tier above GPT-5.6 Sol.

### Fixes
- Version the Antigravity plugin manifest (`plugin/plugin.json`) with releases:
  `scripts/bump.sh` now bumps it alongside the other manifests and
  `scripts/validate.sh` fails when its version drifts from the release
  version.

## v0.6.0

### Features
- Default Gemini delegation to Gemini 3.7 Flash at high effort, with selectable Flash/Pro families and native Antigravity model and effort flags.
- Capture Antigravity structured output and conversation IDs for reliable results and resume support.
- Run Gemini autonomously with edit auto-approval, tool auto-approval, and terminal sandboxing; `--full-access` additionally removes terminal isolation.
- Add `--wait <job-id>` so native host background monitors can stay attached to an async peeragent job and receive its terminal result.

### Removed
- Remove the bundled MCP adapter and plugin registration. Detached MCP work has no portable completion wake-up, while blocking MCP calls compete with host timeouts; the plugin now uses the bundled CLI completion boundary.
- Remove the opinionated `peer-review` orchestration skill. The general `peer` skill remains the single way to give hosts access to other local agents.

## v0.5.1

### Fixes
- Repair Codex plugin MCP packaging so the bundled server uses the standard `.mcp.json` contract and starts from the plugin root.
- Validate the packaged Codex MCP configuration with the real plugin-relative working directory.

## v0.5.0

### Features
- Add a stdio MCP server with typed `delegate`, `job_status`, `job_result`, and `job_cancel` tools backed by shared application services.
- Bundle MCP server configuration and guidance for Claude Code and Codex plugins.
- Add GPT-5.6 Luna, Terra, and Sol model aliases for Codex and Fable model support for Claude.
- Ship the first-class Pi package alongside Claude Code and Codex marketplace packaging.
- Harden asynchronous jobs with durable prompt/PID state, guarded terminal transitions, process-group cancellation, and robust child cleanup.

### Security
- Reject malformed and path-traversing async job IDs at input and storage boundaries.

### Fixes
- Prevent inherited Go test-helper state from launching peeragent recursively.
- Avoid Gemini sandbox false-positive authentication and timeout failures.
- Preserve cleanup when detached-process PID persistence or process release fails.

### Refactor
- Centralize agent identities, aliases, prompt names, and display names in one typed registry.
- Centralize persisted lifecycle statuses and terminal-state membership.
- Share target-adapter runner test support without weakening target-specific assertions.

### Documentation
- Document MCP setup, Pi packaging, supported models, async job controls, and the current repository layout.
