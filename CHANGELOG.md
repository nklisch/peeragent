# Changelog

## v0.11.3

### Fixes
- Infer messaging support for jobs without a recorded flag from a saved messaging preamble or an outbound `say`;
  report the source instead of incorrectly calling v0.11.0/v0.11.1 jobs pre-messaging. Preserve missing legacy flags
  during lifecycle saves, and keep explicit recorded flags authoritative.
- Avoid a Windows test deadlock by discarding captured stdout to the platform null device. The old helper wrote a
  command's output before draining its pipe; v0.11.2's 4,105-byte help text could fill the Windows pipe. A bounded
  large-output regression covers this independent of host pipe capacity.

### Features
- Status, job listings and delivery confirmations show the oldest unread reply's age, with context that jobs read
  their inbox between long steps. JSON includes `oldest_unread_at` and `oldest_unread_age`; read entries and prompt
  reservations are excluded. Delivery remains distinct from acknowledgement.

## v0.11.2

### Features
- `tell` confirms delivery with the target job, agent, status, and locked unread count.
- New jobs record messaging support. `tell` and status warn when an older job has no recorded messaging preamble;
  replies still deliver and are carried on `--resume`.
- `--jobs [--cwd DIR] [--all]` lists jobs newest first, including agent/model, lifecycle, creation time, last message
  time and live session. The default scope is the selected directory; `--all` also includes indexed directories.
- Persist `agent_session` directly in `job.json` as soon as a target reports it.
- `--version` prints the running binary version; single and multiple status JSON carry it too.

### Fixes
- Classify Codex `max_output_tokens` stops as `output_limit`, retaining the session and resume command. Opt-in
  `--retry-capacity N` shares its additional-attempt budget with one immediate output-limit continuation per job;
  capacity stops retain their configured backoff. Repeated limits, missing sessions and spent budgets remain terminal.

## v0.11.1

### Fixes
- Windows: wait for the job lock instead of failing with "Access is denied" when another command has just released
  it. The released lock file can stay pending deletion for a moment, which made a `tell` that arrived while the job
  read its inbox fail. A lock that times out now reports the last error it saw.
- Windows: retry every atomic file replace (job state, inbox read state, the job index) for up to 450 ms while another
  process briefly holds the destination open, as the job index already did.

## v0.11.0

### Features
- Add `peeragent say "<text>"` for labelled messages to the recorded launching Herdr pane. The fixed origin/reply
  header cannot be omitted through this command; body header brackets are neutralized. Messages are limited to
  2,000 characters and three per five minutes per job, with timestamped sent/refused outbox entries.
- Add `peeragent tell <job-id> [--cwd DIR] "<text>"` and `peeragent inbox` for replies to running jobs. Commands
  inside jobs announce unread messages on stderr; inbox delivery and acknowledgement share a lock with writers.
- Give every target its job identity without terminal-control variables. All target prompts explain messaging;
  `--no-preamble` sends just the task. Resumes and capacity continuations carry unread replies into their prompts.
- Single/multiple status show unread/sent counts and last message time. Results include the ordered conversation
  after the agent message. Launcher addresses live in a separate user-readable-only file.
- Windows supports inbox/tell and reports `no launcher channel on this platform` for say.

### Fixes
- Keep carried replies unread through target startup failure, cancellation and capacity refusal; acknowledge only
  accepted turns. Reserve carried replies by attempt so the running inbox cannot deliver them again, while later
  replies remain readable. Recover abandoned reservations on the next read after a worker crash. Deduplicate alias-discovered jobs by recorded job ID so resumes include each reply once.
- Flatten Unicode separators, explicit direction formatting and common header-bracket look-alikes. Share one
  mailbox snapshot per job per command, and append outbound outcomes without rescanning history.
- Clarify that labelled messaging guides well-meaning same-user agents; it is not an adversarial boundary.

## v0.10.1

### Security
- Remove inherited terminal-control variables from every target agent by default, preventing use of the launching
  terminal's inherited control addresses to inject input that appears to be from the user. Applies to blocking,
  async, resumed and capacity-retried calls across Codex, Claude, Gemini and GLM.
- Add the deliberate `--keep-terminal-env` override and record it in job/result metadata. Authentication and ordinary
  shell settings remain available. The complete variable list is in [the terminal environment policy](docs/TERMINAL_ENV.md).

### Fixes
- Async CLI tests cancel and wait for their own workers to exit before removing temporary job files, preventing
  Windows cleanup failures caused by workers still holding `agent.log` open.

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
- Single and multi-job status show the last Codex event time and a short agent-message preview from a bounded log tail.
  `agent.log` fills while the job runs; full reports remain in `message.txt`.
- Every CLI launch, including blocking and resumed calls, carries its job id in JSON and text.
- Resumes refuse with exit 2 when the same agent session has a live running job, naming the job and wait/cancel commands.
  `--allow-concurrent-session` permits intentional overlap; terminal jobs and exited processes do not block.

### Fixes
- An async Codex job that logged its final message and `task_complete`, but never exited, stays `running` no longer.
  A terminal event starts a short completion grace, after which only the job's own process tree is stopped. Any new
  agent activity during the grace cancels it, so a session that continues after `turn.completed` isn't cut off.
- Cleanup keeps ownership of the job's process group until its last signal, so it can never signal a process-group
  number the system has reused.
- Pipe draining has no timeout before completion, so a launcher that starts the agent in the background and exits
  no longer cuts the agent off.
- Cancellation reaches the whole job even when a large prompt is still being written to a descendant that holds stdin.
- Blocking launches retain and collect their worker's exit. A worker that exits without a terminal result now fails
  with its exit status and log path instead of hanging the caller. Exited, uncollected Unix workers no longer block resumes.
- Recognize model-capacity failures during remote context compaction, retaining terminal-event precedence and opt-in retries.
- Capacity results and status explain the stop and show the resume command before the last agent message.
- Retry-option errors and help explicitly require `--async --agent codex`, including with `--resume`.

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
