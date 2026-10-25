# Peeragent

Peeragent is a Claude Code and Codex plugin that lets a host assistant
delegate arbitrary task work to a peer local coding agent — Codex, Claude
Code, Gemini through Google Antigravity, or Z.AI GLM 5.2 through Pi.

Use it when you are working in Claude Code or Codex and want another local
agent/model to take a focused task pass — implementation, research,
review, debugging, refactors, docs, build fixes, anything. The host
assistant keeps the conversation with you; the peer agent inspects or
edits the repo, runs verification when it can, and returns a small result
for the host to summarize.

## What It Can Call

Peeragent wraps local CLIs you already have installed:

- `codex`: OpenAI Codex CLI through `codex exec`
- `gemini`: Gemini through Google Antigravity CLI, `agy --print`
- `claude`: Claude Code CLI through `claude --print`
- `zai`: Z.AI GLM 5.2 through Pi, `pi --provider zai --model glm-5.2 -p`

It does not include accounts, API keys, Codex, Claude Code, Antigravity, Pi,
or Z.AI access. Install and authenticate/configure the target CLIs you want to
use before delegating work.

## Install From The Marketplace

Claude Code:

```sh
claude plugin marketplace add nklisch/peeragent
claude plugin install peeragent@peeragent
```

Codex:

```sh
codex plugin marketplace add nklisch/peeragent
codex plugin add peeragent@peeragent
```

Pi:

```sh
pi install git:github.com/nklisch/peeragent@v0.15.1
```

The Pi package loads the `peer` skill from `plugin/skills`, so its wrapper
resolution lands on the bundled `plugin/bin/peeragent` shim and committed
platform binaries.

On the four supported platforms (linux amd64/arm64, darwin amd64/arm64),
the plugin runs immediately with no download and no Go toolchain required —
prebuilt binaries are committed in `plugin/bin/<goos>-<goarch>/peeragent`.

On any other platform, install from source (requires Go 1.25 or newer):
`go install github.com/nklisch/peeragent/cmd/peeragent@latest` (or pin
`@v<plugin-version>` to match your installed plugin), then set `PEERAGENT_BIN`
to the installed binary (typically `$(go env GOPATH)/bin/peeragent`).
Prebuilt release archives for the four supported platforms are also published at
https://github.com/nklisch/peeragent/releases for manual install. If your
platform is misdetected, set `PEERAGENT_TARGET_OVERRIDE=<goos>-<goarch>` to
select a present binary.

On Windows, agent CLIs installed through npm can be launched through their
`.cmd` shims, including from paths with spaces and parentheses. The shared
runner preserves quoted arguments and shell characters when a shim forwards
`%*` to a native executable. Arguments travel through process-local environment
variables and expand after shell parsing, preserving multiline tasks without
interpreting their contents as commands. This requires the shim to preserve
delayed expansion at its forwarding command, as npm shims do. Native launches
retain their normal argument handling. Codex prompts are sent through stdin
for both new and resumed sessions, so large tasks (including `--prompt-file`
input) bypass Windows command-line limits. Other targets still pass prompts as
arguments: a quoted argument exceeding cmd.exe's 8191-character limit is
rejected before launch; use a native agent executable for those tasks.

Claude Code's POSIX cache wrapper follows the active install named in
`plugins/installed_plugins.json`, so a session's older PATH entry can pick up an
updated plugin without restarting. This requires the older wrapper to contain
the forwarding logic, and Python 3 or jq to read the record. Missing records or
readers keep the current install running. `PEERAGENT_BIN` and source checkout
builds retain precedence. Codex currently has no equivalent install-path record;
its wrapper keeps the existing selection.

## Using It

The plugin exposes one skill in Claude Code, Codex, and Pi:

- `/peer`: delegate a focused task pass to a peer local coding agent

Example prompts:

```text
/peer --agent codex --model luna --effort high Fix the failing parser test and run the relevant test package.
/peer --agent claude --model fable --effort xhigh Refactor the result formatter and update its tests.
/peer --agent gemini Inspect the CLI docs and patch stale usage text.
/peer --agent zai --effort xhigh Ask GLM 5.2 to audit the retry edge cases.
```

The host assistant remains responsible for reading the wrapper result and
explaining the outcome to you. For substantive work, the skill prefers
`--async`, starts `--wait <job-id>` through native background monitors or
completion wake-ups when available, and reads the terminal result.

## Why Peeragent Uses Skills And A CLI

Peeragent deliberately uses a host skill that invokes the bundled CLI instead of
registering an MCP server. The skill prefers `--async` for substantive work and
runs `--wait <job-id>` through native host monitoring or completion wake-ups
when available. Short tasks can remain blocking; `--status`, `--result`, and
`--wait` provide explicit job controls.

MCP is not registered because it provides request/response tools, not a portable
completion wake-up for detached work. Blocking calls compete with host MCP
timeouts; async MCP calls return before completion and depend on the model
remembering to poll. The CLI instead provides an attached `--wait` process that
a host's native monitor can observe. Native host agent channels can provide lifecycle
notifications, but that is a host capability rather than something a generic
peeragent MCP server can reproduce.

## Agent Equivalence And Defaults

Peeragent does not automatically replace a host assistant's normal sub-agent
pattern. If you want that behavior, add a project instruction to `CLAUDE.md` or
`AGENTS.md` telling the host when to delegate through `/peer`.

The `sol` alias selects `gpt-6.1-sol`; `luna` selects `gpt-6-luna`, and
`terra` selects `gpt-5.6-terra`. Explicit `gpt-6-sol`, `gpt-5.6-luna`, and
`gpt-5.6-sol` IDs remain supported.

Use GPT-6 for routine Codex work and jump to GPT-6 Astra for the most
demanding passes. Luna is the fast default, Sol is the higher-capability GPT-6 tier,
Astra is the GPT-6 flagship above Sol, and Terra is an optional bridge when
Luna is not enough but Sol is more than the task needs:

| Desired delegated pass | Recommended Codex target | Rough Claude tier |
| --- | --- | --- |
| Fast or routine | `--agent codex --model luna --effort high` | Sonnet-class |
| Lots of routine work | `--agent codex --model luna --effort xhigh` | Strong general-purpose pass |
| Middle bridge | `--agent codex --model terra --effort high` (or `xhigh`) | Between the general and flagship tiers |
| Opus-tier | `--agent codex --model sol --effort low` (or `medium`) | `--agent claude --model opus --effort xhigh` |
| Fable-tier | `--agent codex --model sol --effort high` (or `xhigh`) | `--agent claude --model fable --effort high` (or `xhigh`) |
| GPT-6 flagship | `--agent codex --model astra --effort high` (or `xhigh`) | Above Fable-tier |

Most callers can jump directly from Luna to Sol rather than routing through
Terra. Gemini through Antigravity defaults to Gemini 3.7 Flash at high effort.
Peeragent passes both `--model` and `--effort` to current `agy`; `flash` selects
3.7 Flash, while `pro` selects 3.1 Pro. Z.AI through Pi is fixed to `glm-5.2`;
`--model glm-5.2` is accepted for explicit metadata, and no other Z.AI models
are surfaced by peeragent.

Claude Code project snippet:

```markdown
## Peer Delegation

When you would normally use an implementation, research, or review sub-agent,
prefer `/peer` for concrete code changes, bug fixes, refactors, tests, docs
updates, build fixes, research passes, and review passes in this repository.

- Use `/peer --agent codex --model luna --effort high` for routine work and
  Luna at `xhigh` when there is lots of work. Jump to Sol at `low|medium` for an
  Opus-tier pass or `high|xhigh` for a Fable-tier pass; Terra is an optional
  bridge. Use GPT-6 Astra at `high|xhigh` for the most demanding passes.
- Use `/peer --agent claude --model fable` for the strongest Claude pass.
- Use `/peer --agent gemini` for a Gemini 3.7 Flash pass through Antigravity.
- Use `/peer --agent zai` for a Z.AI GLM 5.2 pass through Pi.
- Use `--effort xhigh` when the work is dense or the stakes are high.
- Research-only and review-only delegation are allowed.
- Do not use peeragent for planning-only orchestration work.
```

Codex project snippet:

```markdown
## Peer Delegation

When you would normally use an implementation, research, or review sub-agent,
prefer `/peer` for concrete code changes, bug fixes, refactors, tests, docs
updates, build fixes, research passes, and review passes in this repository.

- Use `/peer --agent claude --model fable --effort xhigh` for the strongest
  Claude pass; Opus, Sonnet, and Haiku remain available for lower tiers.
- Use `/peer --agent gemini` for a Gemini 3.7 Flash pass through Antigravity.
- Use `/peer --agent zai` for a Z.AI GLM 5.2 pass through Pi.
- For Codex, use GPT-6 Luna at `high` for routine work or `xhigh` for lots of
  work. Jump directly to Sol at `low|medium` for an Opus-tier pass or
  `high|xhigh` for a Fable-tier pass. Terra is an optional middle bridge. Use
  GPT-6 Astra at `high|xhigh` for the most demanding passes.
- Do not use peeragent for planning-only orchestration work.
```

## Direct CLI Usage

Clone and build for local development:

```sh
git clone https://github.com/nklisch/peeragent.git
cd peeragent
make build
```

Blocking mode is the default:

```sh
bin/peeragent --agent codex --model luna --effort high "Implement the requested change and run relevant tests."
bin/peeragent --agent gemini "Implement the requested change and run relevant tests."
bin/peeragent --agent claude "Implement the requested change and run relevant tests."
bin/peeragent --agent zai "Implement the requested change and run relevant tests."
```

Read task text from a file:

```sh
bin/peeragent --agent codex --model luna --prompt-file task.md
```

Run against another checkout:

```sh
bin/peeragent --cwd /path/to/repo --agent claude "Update the CLI help text."
```

Ask for human-readable output:

```sh
bin/peeragent --text --agent gemini "Fix the failing parser test."
```

Continue a prior target-agent session:

```sh
bin/peeragent --agent claude --resume <agent-session> "Check whether the revision addressed your prior concerns."
```

Every launch has a `metadata.job_id`, including blocking and resumed calls.
Text output prints the job id; blocking text calls also print it to stderr before waiting.
Blocking calls retain and collect their worker's exit. If it exits before writing
a terminal result, the job fails with the worker's exit status and a log path;
the caller returns failure and that exited worker does not block a later resume.
When available, JSON output includes `metadata.agent_session`. Use that value
for continuity inside one review loop. Start a fresh call without `--resume`
when you want an independent second opinion.

The default result is intentionally compact: for Codex JSONL output, peeragent
surfaces only the final completed agent message in `details`. Scripts should
read the complete, untrimmed handback from the top-level `final_message` string
(`null` while running or when the target produced none). If you need to
inspect the raw target stdout/stderr, open `metadata.log_path`; for continuity,
use `metadata.agent_session` to resume the target session.

Target agents start without the launching terminal's control environment by
default, preventing use of inherited pane/socket addresses to inject terminal
input. Use `--keep-terminal-env` for a deliberate need; the job records the
choice. See [the terminal environment policy](docs/TERMINAL_ENV.md) for the
complete variable list and scope. Authentication and ordinary shell settings
remain available.

## Models, Effort, Profiles, And Access

`--resume <agent-session>` keeps the model and effort from that session's most
recent peeragent job. `--model` and `--effort` each override only their own value.
The session's indexed directory is reused unless `--cwd` overrides it. If the
previous job record is unavailable, omitted options use today's defaults and
`metadata.resume_warning` explains the fallback. Launch and result metadata show
the resolved options; an absent Codex/Claude model keeps the native CLI default.

Codex, Gemini, Claude, and Z.AI GLM 5.2 support `--effort`. Codex defaults to
`high` and accepts `low`, `medium`, `high`, and `xhigh`. Gemini defaults to
`high` and accepts `low`, `medium`, and `high`. Z.AI defaults to `high` and
accepts `medium`, `high`, and `xhigh`. Claude defaults to `xhigh` and accepts
only `high` or `xhigh` through peeragent:

```sh
bin/peeragent --agent codex --model luna --effort high "Implement the routine change."
bin/peeragent --agent codex --model luna --effort xhigh "Work through the large routine change."
bin/peeragent --agent codex --model sol --effort medium "Run an Opus-tier review."
bin/peeragent --agent codex --model sol --effort xhigh "Run a Fable-tier migration."
bin/peeragent --agent codex --model terra --effort high "Use the optional middle tier."
bin/peeragent --agent codex --model astra --effort xhigh "Run the GPT-6 flagship pass."
bin/peeragent --agent claude --model fable --effort xhigh "Untangle the hardest integration test."
bin/peeragent --agent claude --model opus --effort xhigh "Run an Opus pass."
bin/peeragent --agent gemini --model flash --effort high "Run a Gemini 3.7 Flash pass."
bin/peeragent --agent zai --effort xhigh "Review the cross-module migration for hidden regressions."
```

Codex accepts the short aliases `astra`, `luna`, `terra`, and `sol` and passes
`astra` as `gpt-6-astra`, `luna` as `gpt-6-luna`, `sol` as `gpt-6.1-sol`,
and `terra` as `gpt-5.6-terra` to the Codex CLI. Explicit `gpt-6-sol` and
GPT-5.6 IDs are also supported. Claude
supports `--model
fable`, `--model sonnet`, `--model opus`, and `--model haiku`. Gemini accepts
`flash` (the default Gemini 3.7 Flash), `pro` (Gemini 3.1 Pro), and explicit
supported family IDs for 3.7, 3.6, and 3.5 Flash or 3.1 Pro. Flash accepts
`low|medium|high`; Pro accepts `low|high`. Peeragent passes the normalized model
and effort to `agy`. Z.AI accepts only `--model glm-5.2`;
peeragent intentionally does not surface the other Z.AI models that Pi may
list.

```sh
bin/peeragent --agent gemini --model flash --effort high "Implement the requested change."
bin/peeragent --agent zai --model glm-5.2 "Implement the requested change."
```

Codex also supports profiles:

```sh
bin/peeragent --agent codex --model luna --profile peeragent "Use this Codex profile."
```

Quick Z.AI GLM 5.2 configuration checks:

```sh
pi --list-models zai | grep -w 'glm-5.2'
pi --provider zai --model glm-5.2 --thinking high --no-session --no-tools -p 'Reply with OK.'
bin/peeragent --agent zai --text 'Reply with OK and do not edit files.'
```

If the Pi smoke test fails, configure Pi with `ZAI_API_KEY` or `/login` for the
ZAI provider, then retry. Peeragent's Z.AI target always uses `glm-5.2`; it does
not expose the other Z.AI models Pi may know about.

Default execution uses each target CLI's autonomous mode in the current
checkout:

```text
codex exec --json --cd <repo> --sandbox workspace-write ...
agy --output-format json --model gemini-3.7-flash --effort high \
  --mode accept-edits --sandbox --dangerously-skip-permissions \
  --add-dir <repo> --print <prompt>
claude --print --output-format json --permission-mode auto --add-dir <repo> ...
pi --provider zai --model glm-5.2 --thinking <effort> --no-session -p ...
```

You can pass `--sandbox` explicitly to select that same default mode where the
target CLI has one. Gemini needs `--dangerously-skip-permissions` in print mode
so it can run shell commands and tests without an unavailable interactive
prompt. Peeragent combines that auto-approval with Antigravity's terminal
sandbox and `accept-edits` mode. The sandbox contains spawned terminal
processes, but it does not confine Antigravity's direct file tools;
`--dangerously-skip-permissions` can approve writes outside the workspace. Treat
a Gemini delegation as a trusted local agent even without peeragent's
`--full-access`; that flag additionally removes terminal isolation. Pi exposes
no separate peeragent sandbox flag, so its delegation requires the same trusted
repository and machine context.

Use full access only for a trusted repo and an explicit reason:

```sh
bin/peeragent --agent claude --full-access "Run the trusted local migration."
```

`--worktree` is reserved for future isolated worktree execution. Today it
returns a clear JSON failure instead of silently changing how work is done.

Options are strict: unknown options such as `--list` and `--background`, and
missing or empty option values, exit with usage code `2` before starting work.
The source-checkout shim preserves those exit codes too.
Use `--` to pass task text that starts with a dash:

```sh
bin/peeragent -- --review-this-task
```

## Job messages

The launcher can reply to a running job, and the job can ask its launcher a
question without receiving terminal-control addresses:

```sh
peeragent tell <job-id> [--cwd DIR] '<text>'  # launcher → job mailbox
peeragent tell <job-id> --prompt-file reply.md # read a longer reply from a file
peeragent inbox                            # job → print unread replies and mark read
peeragent say "<text>"                      # job → native launcher inbox, or queued for review
peeragent say --prompt-file question.md       # file input also works for say
```

Outbound messages always begin with:

```text
⟦peer · <agent> job <job-id> · <worktree basename>@<branch> · reply: peeragent tell <job-id> "…"⟧ <text>
```

Body header brackets are replaced. Limits are 2,000 characters and three messages
per five minutes per job. Without a safe native launcher inbox,
`say` retains the question as `queued` in the job's `--result` conversation and
prints an explicit queued receipt. It never writes to terminal input. Claude
Code launchers use their recorded local socket; nested jobs use the parent job's
mailbox. Herdr-only and unsupported launchers queue. Transport failures also queue, without automatic retries.
Native submission does not acknowledge acceptance or reading.

Single-quote reply text or use `--prompt-file`: backticks inside double quotes
execute in the shell before `tell` can read the message.

Both `tell` and `say` accept `--prompt-file <path>` alone or with positional text.
As with launch, positional text comes first, followed by trimmed file content,
separated by a blank line. Paths are relative to the calling process.

`tell` confirms persisted delivery, for example:
`delivered to job <id> (<agent>, <status>); unread 3, oldest 2h14m; the job reads its inbox between long steps`.
Delivery does not mean the job has read the reply. The count and oldest unread
age come from the delivery lock; the job may read the reply immediately afterward.
Status and job listings expose `oldest_unread_at` and `oldest_unread_age` alongside
`inbox_unread` (in metadata for single-job results). Text includes the age and
context when replies remain unread. Ages use whole seconds below a minute,
then whole minutes; future timestamps display `0s`. Missing timestamps display
`unknown`; an empty inbox has no oldest-unread fields. Entries reserved for a
running prompt are excluded until their reservation is released.

New jobs record `messaging_enabled` in `job.json`, and launch JSON reports that same saved state.
`tell` to a job without messaging prints `queued for the next --resume (N pending)`
when its agent session is known. Replies stay in the job folder and are prepended to the next
resume prompt, clearly labelled as messages sent while the job ran. If no session is known,
`tell` exits non-zero and says the message was not queued; retry after the session is recorded.

For older jobs without that field, a saved messaging preamble or an outbound `say` entry proves messaging
support. Controls report `messaging_enabled: true` and `messaging_source` as
`inferred from prompt` or `inferred from outbox`, and show that source in text
and delivery confirmations. Explicit recorded flags remain authoritative.
Only jobs with neither evidence source get the pre-messaging warning; replies
are queued for `--resume` when a session is known. `--no-preamble` jobs get
a specific missing-preamble warning. Saving a legacy job's lifecycle preserves
an absent marker so later controls can still infer support from evidence.

Prompts instruct every target to check `peeragent inbox` after long steps and
before handback:

Use `peeragent say` only to ask a question you can't settle yourself, never for status or progress updates.
Your final handback carries the final status, so don't repeat it in a message.
After asking a question, watch your inbox for the answer: run `peeragent inbox` every few minutes, between steps,
until it arrives, and keep doing independent work meanwhile.

`--no-preamble` sends task text only. Each command inside a job announces unread
replies on stderr. Resumes include unread messages in the new
prompt and reserve them, hiding them from the running target's inbox and unread
count. Target output and an accepted turn mark them read; startup failure,
capacity/login refusal or interruption releases them. A dead worker's unresolved reservation recovers
on the next mailbox read. Later replies remain independently readable. Status shows `inbox_unread`,
`outbox_sent`, and `last_message_at`; results include the ordered `conversation`
after the agent message. See [messaging details](docs/MESSAGING.md).

## Heavy integration

When the optional `heavy` client is installed, delegated workers inherit
`HEAVY_PARENT_SESSIONS`: the existing colon-separated lineage plus the launcher's
owner from `heavy whoami`, falling back to `HEAVY_SESSION`, then `CODEX_THREAD_ID`.
This lets heavyd authorize ancestor sessions to manage delegated heavy work.

At handback, peeragent records queued or running heavy work in `pending_heavy_steps`
(`job`, `state`, `command`) and warns in text output. Ownership matches process ancestry
or the target's agent session, never just the directory. Linux also recognizes orphaned
clients that remain in the worker's process session. It reads `heavy ls --json`, falling
back to `$HEAVYD_DASHBOARD` (default `http://127.0.0.1:8473/api/overview`). Missing tools,
unsupported subcommands, or unavailable dashboards silently skip this optional check.
The warning records the handback snapshot; it does not wait for or cancel heavy steps.

Result and wait summaries contain the final handback, trimmed to 600 characters with
an ellipsis when needed. Details and text output keep the complete message.

## Async Jobs

For longer work, start a background job. For Codex, add opt-in
`--retry-capacity N` beside `--async` to resume after capacity stops (default
backoff `10m`; retries are disabled unless requested):

```sh
bin/peeragent --agent gemini --async "Refactor the result formatter and run tests."
```

Wait for the terminal result, preferably through the host harness's native
background monitor:

```sh
bin/peeragent --wait <job-id>
```

Wait for whichever of several jobs finishes first, using the same native monitor:

```sh
bin/peeragent --wait-any <job-id> <other-job-id>
```

All ids are checked before waiting; an unknown id exits `2`. Already finished
jobs return immediately. If several are terminal when checked, argument order
breaks the tie. Like `--wait`, terminal results exit `0` even for failed or
cancelled targets; inspect `status` and `metadata.exit_code` for their outcomes.

Check status without waiting:

```sh
bin/peeragent --status <job-id>
```

`peeragent --jobs [--cwd DIR] [--all] [--json | --text]` lists every job in
that directory, including finished jobs and unindexed local jobs, newest first.
`--all` also includes retained metadata from removed worktrees, marked as
`indexed_only` with an explicit warning. It does not search arbitrary directories. JSON output is one object
per line, using the multi-job status fields: job id, agent, model, status,
`started_at` (creation time), `last_message_at` when available, and
`agent_session`. An empty JSON list produces no lines; `--text` prints a table.
`peeragent --version` prints the running binary's version. Status JSON carries
that version in `metadata.version` for one job and `version` in each multi-job
or listing row, even when the job was started by an older binary.

For several jobs, `--status <job-id> <other-job-id>` prints one compact JSON
record per line. Add `--text` for a table of status, model, effort, session,
start time, capacity backoff, current worktree HEAD when git is available,
and the last agent event time and a short single-line message preview.
Single-id status keeps its result format and adds `metadata.last_event_at`
and `metadata.last_message` when progress is available. `started_at` is the job's
launch time. HEAD is read once per worktree without inspecting changed files.

Both multi-job controls accept `<job-id>@<directory>` for unindexed jobs,
including paths with spaces when quoted. Each per-job directory wins over
`--cwd`; other ids use explicit `--cwd`, then the index, then the current
directory. For example:

```sh
bin/peeragent --wait-any <indexed-job-id> '<old-job-id>@/path with spaces/repo'
```

Fetch the current or final result:

```sh
bin/peeragent --result <job-id>
```

Codex session ids and the latest completed agent message are saved while the
job runs, so `--status` exposes `metadata.agent_session` before cancellation.
`--result` includes the message without truncation or points to
`metadata.message_path` (`message.txt`), including after failure or cancellation
and across capacity retries. `agent.log` fills live with timestamped Codex event
names and message previews; status reads only its last 8 KiB. Full reports stay
in `message.txt`, and raw attempt logs remain available through
`metadata.log_path` when written.

A Codex terminal event (`turn.completed`, `turn.failed`, `task_complete`, or a
message marked with a final phase) starts a five-second completion grace period.
During that period, status says “final message received, process still alive”.
If Codex stays alive, peeragent ends its own child and owned process group
(Windows Job Object on Windows), preserving the terminal event's outcome.
A `turn.completed` event is provisional: new turn, item, agent or tool activity
cancels its timer, and later completion starts fresh grace. Ordinary completed
messages without a final phase can be progress updates and do not start this
timer. Launcher exit alone does not limit how long worker output can drain.
An unterminated JSONL line is processed when stdout reaches EOF.

Cancel the job:

```sh
bin/peeragent --cancel <job-id>
```

Job state for both blocking and async launches lives under `.peeragent/jobs/`
in the target repository. It is
local runtime state and ignored by git. Job controls find newly launched jobs by
id from any directory. `--resume <agent-session>` uses the directory recorded
when that session was last observed. An explicit `--cwd <path>` always
wins. Older jobs and sessions without an index entry still need their original
directory or `--cwd`.

The lookup index lives at `<user-cache>/peeragent/index/` (Go's `os.UserCacheDir`:
`$XDG_CACHE_HOME` or `~/.cache` on Linux, `~/Library/Caches` on macOS, and
`%LocalAppData%` on Windows). `PEERAGENT_STATE_DIR` overrides the `peeragent`
state root. Its job entries retain compact metadata; session entries contain working
directories, keyed by agent and session id; short launch-lock files contain their holder PID. Directories use mode `0700` and files `0600` on Unix. Clearing
it does not remove jobs, but requires `--cwd` for lookup again. An index write
failure is reported in `metadata.index_warning`; the job still runs and can be
controlled with `--cwd`.

A terminal result is also copied to `index/results/` so `--result`, `--wait`,
`--status`, and `--cancel` can return it by id after its worktree is removed.
These snapshots contain only result JSON; prompt files and raw log files are
not copied. Snapshots expire after 30 days, and archiving a new result removes
expired snapshots. Repository job folders keep their existing lifetime.
Snapshots are a fallback for missing repository state; corrupt or unreadable
local state still reports its error. Clearing the index also clears these copies.
Jobs with neither retained metadata nor a full result snapshot need their
repository-local files.

Job metadata survives worktree removal in the per-user index for 30 days, up to
10,000 entries, with summaries capped at 600 Unicode characters. `--jobs --all`
keeps those jobs discoverable, including their session, model, effort and finish
time. When no full result archive remains, `--status` and `--result` return the
indexed summary and mark `metadata.indexed_only: true`. Resume with
`--resume <agent-session> --cwd <existing-replacement-worktree>` to keep the
saved model/effort without recreating the removed directory or its old mailbox.
Full result archives retain their separate 30-day lifetime and take precedence;
older archives are promoted into compact metadata during listing or resumption.

Resuming a session with a live running job exits `2` before launch, naming that
job and its `--wait` and `--cancel` commands. This prevents two agents from
interleaving edits and session history. Terminal jobs and jobs whose recorded
process is gone do not block resumption. Use `--allow-concurrent-session` to
permit intentional overlap or bypass a stale launch lock. The check covers the
current directory and worktrees known to the per-user index; after clearing the
index, keep `--cwd` pointed at the worktree holding the running job.

Codex's explicit “Selected model is at capacity” stop, including during remote
context compaction, returns the terminal status `capacity_exhausted`, with `metadata.agent_session` when available.
Results, wait summaries and text output say `stopped: capacity_exhausted`, include
`--resume <session>` when known, and suggest `--retry-capacity N` for automatic recovery.
Scripts should read the JSON `status` field and match `capacity_exhausted`, rather than
matching human-readable summaries or the agent's own log text. Agent logs are preserved verbatim.
Blocking calls exit `1`. By default no retry occurs. To opt into up to twelve
additional attempts with a ten-minute delay:

```sh
bin/peeragent --async --retry-capacity 12 "Implement the requested change."
```

Codex `max_output_tokens` stops return the terminal status `output_limit`,
with the captured `agent_session` and a resume command in the summary. Under
`--async --agent codex --retry-capacity N`, an output-limit stop consumes one
of the same N additional attempts as capacity recovery. It resumes the same
session immediately with a short continuation asking for a concise handback;
`--capacity-backoff` applies only to capacity stops. Each job gets at most one
automatic output-limit continuation, even if budget remains. Without a session,
with N=0, after that continuation, or after the shared budget is spent, the stop
is terminal. `metadata.capacity_retries` retains its existing name and counts
all additional attempts, including output-limit and login continuations. Other
failures are never retried.

A revoked or invalidated Codex login returns `auth_required`, preserving
`metadata.agent_session` and a resume hint. With the shared async retry budget,
peeragent resumes once immediately using the current login; it does not change
credentials. If that attempt still needs authentication, sign in and resume
manually. This continuation shares the N-attempt budget and is counted in
`metadata.capacity_retries`.

Codex's “failed to record rollout items: thread ... not found” diagnostic returns
`interrupted` only when no structured terminal outcome exists. The session is
preserved for manual resumption; interruptions are not retried automatically.
An explicit structured outcome always wins over stderr history. A Codex
cybersecurity policy refusal remains `failed`, with `failure: "policy_refusal"`
and a summary naming OpenAI's safety filter, including status and job listings.
`metadata.stop_reason: "provider_safety_filter"` identifies the stop, and
`metadata.stop_message` carries the exact provider diagnostic. The summary warns
that resuming the same history is likely to fail again; start a fresh session
for unrelated steps. Such refusals are never retried automatically.

`--capacity-backoff <duration>` changes the delay, for example `--capacity-backoff
5m`. Retry options require `--async --agent codex`, including when using
`--resume`; add `--async` to a resumed call to enable them. Each retry resumes
the same
session with a continuation prompt and preserves execution settings and the
job id; it never starts a replacement conversation. Without a session id, or
after the retry budget is spent, the capacity result is terminal. Jobs remain
`running` during the delay. `metadata.capacity_retries` records retries made.
Other failures are never retried. Each attempt keeps a raw log (`target.log`,
then `target-retry-N.log`); noisy Codex “Orphan custom tool call” lines are
omitted only from the visible stderr summary, remaining in the raw log.

Single-job JSON output mirrors `metadata.job_id` as top-level `job_id` for launch,
status and results. The repository directory remains `metadata.cwd`. `--wait`
writes exactly one JSON document to stdout, its terminal result; diagnostics and
unread-message notices use stderr. Multi-job status and job listings use JSONL.


## Repository Shape

This repo is shaped as a Claude Code marketplace, a Codex marketplace, and a
Pi package. The root is the development source; `plugin/` is the committed
install package that marketplaces point at.

```text
package.json                            # Pi package manifest, loads ./plugin/skills
.claude-plugin/marketplace.json        # Claude marketplace entry, source ./plugin
.agents/plugins/marketplace.json       # Codex marketplace entry, source ./plugin
plugin/.claude-plugin/plugin.json      # Claude plugin manifest
plugin/.codex-plugin/plugin.json       # Codex plugin manifest
plugin/plugin.json                     # Antigravity plugin manifest
plugin/skills/peer/SKILL.md
plugin/bin/peeragent
```

The root also keeps the same manifests, skill, and shim for local development.
Run `scripts/package-plugin.sh` after changing plugin metadata, the skill, or
the shim so `plugin/` stays in sync.

## Releasing

Marketplace installs use the committed `plugin/bin/<goos>-<goarch>/peeragent`
binaries directly. Release artifacts are those same four platform binaries
published as downloadable archives for manual install.

Build release archives locally:

```sh
make release VERSION=0.15.1
```

That writes:

```text
dist/release/peeragent_0.15.1_linux_amd64.tar.gz
dist/release/peeragent_0.15.1_linux_arm64.tar.gz
dist/release/peeragent_0.15.1_darwin_amd64.tar.gz
dist/release/peeragent_0.15.1_darwin_arm64.tar.gz
dist/release/checksums.txt
```

Publish a GitHub release from a machine with `gh` authenticated:

```sh
make publish-release VERSION=0.15.1
```

The GitHub Actions workflow in `.github/workflows/release.yml` also publishes
these assets whenever a `v*` tag is pushed, or when run manually with a version.

## Development

The supported minimum Go version for source builds and development is Go 1.25.

Run the test suite:

```sh
make test
```

Run the full validation script:

```sh
scripts/validate.sh
```

The validation script runs tests, builds the binary, builds release archives,
checks plugin and marketplace metadata, verifies README examples, and runs a
small shim smoke test.

## Troubleshooting

If delegation fails immediately, confirm the target CLI is installed and signed
in:

```sh
codex --version
agy --version
claude --version
pi --version
```

If peeragent reports it has no prebuilt binary for this platform (exit code 3),
note that the prebuilt platforms are linux/darwin on amd64/arm64. On those,
reinstall the plugin or download the matching archive from
https://github.com/nklisch/peeragent/releases. On any other platform, install
from source (requires Go 1.25 or newer): `go install github.com/nklisch/peeragent/cmd/peeragent@latest`
then set `PEERAGENT_BIN` to the installed binary. For source checkouts,
`make build` also produces a local `dist/peeragent` binary.

Job ids from indexed launches work from any directory. If lookup fails, check
whether the per-user index was cleared; use `--cwd` for an unindexed job whose
repository state remains. Removed worktrees require retained metadata or a result archive; a job with
neither still needs its original files. Metadata is retained for 30 days, up
to 10,000 entries. Use `--jobs --all` to recover a retained session ID.

### Gemini permissions and autonomy

Peeragent runs Gemini with `--dangerously-skip-permissions` because Antigravity
print mode cannot pause for tool approval. Without auto-approval, ordinary test
and build commands are soft-denied and autonomous coding is not viable. The
default also passes `--sandbox`, which contains terminal processes. Direct file
tools are not contained by that terminal sandbox and can write outside the
workspace when permissions are skipped, so use Gemini only with repositories
and task prompts you trust. `--full-access` removes the remaining terminal
sandbox.

Peeragent's current `agy` integration expects Antigravity CLI 1.1.12 or newer
because that release fixed `--mode` handling in headless runs.

### Gemini auth times out every call

Antigravity (`agy`) stores its OAuth token in the system keyring (Keychain on
macOS, Credential Manager on Windows, libsecret on Linux). On headless Linux
without a desktop session, no Secret Service is running by default, so `agy`
falls through to a one-time browser flow that times out in non-interactive
mode and never persists a token. Symptoms: the wrapper returns immediately
with output containing `Authentication required` and `authentication timed
out` even after a successful interactive login.

Fix on Linux:

1. Install and start a libsecret provider:

   ```sh
   sudo dnf install gnome-keyring         # or: sudo apt install gnome-keyring
   eval "$(gnome-keyring-daemon --start --components=secrets,ssh)"
   ```

2. Use an empty-password default keyring so it auto-unlocks without prompts
   (Seahorse: delete the default keyring and create a new one with no
   password, then mark it default).

3. Run `agy` once interactively, complete the browser login, type a prompt
   to confirm, and exit cleanly. The token lands in libsecret under
   `service=gemini` and persists across runs.

4. Keep the daemon alive across logins by adding to your shell rc:

   ```bash
   if ! pgrep -f 'gnome-keyring-daemon.*--components=secrets' >/dev/null 2>&1; then
     eval "$(gnome-keyring-daemon --start --components=secrets,ssh 2>/dev/null)"
   fi
   ```

If the keyring is impractical (CI or locked-down hosts), Antigravity CLI 1.1.13
and newer can use `GEMINI_API_KEY` directly. Set `modelProvider` to `"gemini"`
in `~/.gemini/antigravity-cli/settings.json`, export `GEMINI_API_KEY`, and
optionally set `GOOGLE_GEMINI_BASE_URL` for a custom endpoint. This uses the
Gemini API route rather than an Antigravity subscription session.
