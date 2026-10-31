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
peeragent messages <job-id> [--cwd DIR] [--json | --text]
```

Task text is resolved from positional arguments, then `--prompt-file`, then
non-interactive stdin. When the wrapper is reading actual process stdin, it
reads stdin only when no positional task text, prompt file, or job-control flag
is present; this avoids accidentally blocking on host-provided stdin when the
task is already explicit. In-memory stdin readers used by tests or embedders
are always read when present.

`tell` and `say` also accept `--prompt-file <path>`. Like launch, they trim
positional text and file content, omit empty parts, and combine positional text
first, then file content, with a blank line between them. They accept either
source alone or both together; messaging commands do not read stdin. File paths
are relative to the calling process, independently of `tell --cwd`. `inbox`
takes no message source.

Unknown option-shaped arguments are usage errors (exit `2`) naming the valid
options; `--` makes every following argument literal task text. Every valued
option requires a non-empty value and cannot consume the next option as its
value. This validation happens before reading prompts or launching work.

Task text consisting of a job ID, or starting with a lowercase command-like word
followed by a job ID, is a usage error naming the unknown command and real
commands. This prevents a mistyped job-control command from launching an editing
agent. Explicit `--` task input and `--prompt-file` disambiguate intentional text;
ordinary tasks and stdin task input keep their existing resolution order.

Codex receives the constructed prompt through stdin with `-` as its prompt
argument, for both new and resumed sessions. This also applies to file-based
tasks and avoids command-line size limits when Windows resolves `codex` to an
npm `.cmd` shim.

Options:

- `--agent <codex|gemini|claude|zai>`: Select the target agent. Defaults to
  `codex`. The `zai` target is fixed to Z.AI GLM 5.2 through Pi.
- `--async`: Start the target in the background and return a job id.
- `--retry-capacity <N>`: For async Codex delegation, resume capacity, output-limit
  and authentication stops up to N additional times (non-negative integer; default zero). A captured or
  supplied session id is required; otherwise the stop is terminal.
  Requires `--async --agent codex`, including with `--resume`. Output-limit
  and login recovery each allow one immediate continuation within this shared budget.
- `--capacity-backoff <duration>`: Positive delay between capacity retries,
  default `10m`. Requires `--async --agent codex`, including with `--resume`.
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
- `--model <astra|luna|terra|sol|fable|sonnet|opus|haiku|flash|pro|glm-5.2>`:
  Select a GPT-6.1, GPT-6, or GPT-5.6 Codex tier, Claude alias, Gemini family, or
  explicitly record the fixed Z.AI GLM 5.2 target. Codex also accepts canonical
  `gpt-6.1-<sol|terra>`, `gpt-6-<astra|luna|sol>`, and
  `gpt-5.6-<luna|terra|sol>` IDs.
  Gemini defaults to `gemini-3.7-flash`; `flash`, `pro`, and supported explicit
  Gemini family IDs normalize to an `agy --model` value. Flash accepts
  `low|medium|high`, while Pro accepts `low|high`. Z.AI accepts only
  `glm-5.2`; no other Z.AI models are surfaced.
- `--profile <name>`: Pass a Codex configuration profile.
- `--keep-terminal-env`: Keep inherited terminal-control variables for a deliberate
  need. The default strips them from every target process, including resumes and
  capacity retries. The override is recorded in job/result metadata. See
  [the terminal environment policy](TERMINAL_ENV.md) for the complete variable list.
- `--allow-concurrent-session`: Permit concurrent jobs in one agent session.
  Without it, a resume with a live running job is usage exit `2`, with the job
  id, wait/cancel commands and override in the error. Terminal jobs and exited
  recorded processes do not block. The short check/start lock covers concurrent
  resumed launches; the override also bypasses an interrupted launcher's lock.
- `--resume <agent-session>`: Resume a prior target-agent session when the
  target supports it. Use this for continuity inside one review loop; omit it
  for an independent second opinion. Omitted `--model` and `--effort` inherit
  independently from the session's most recent peeragent job; explicit flags win.
  Missing job records keep current defaults for omitted options and add
  `metadata.resume_warning`, without refusing the launch. Launch and result
  metadata contain the resolved model and effort. An absent model for Codex or
  Claude means its native CLI default, including when inherited.
- `--cwd <path>`: Set the repository root, overriding the indexed job or session
  directory. Paths resolve to absolute paths before target invocation. If a job
  read finds no local job there, an indexed job location is used and
  `metadata.lookup_warning` names both folders.
- `--prompt-file <path>`: Read task text from a file.
- `--json`: Emit JSON output. This is the default.
- `--text`: Emit human-readable text output; this flag takes no value.
- `--jobs [--cwd DIR] [--all]`: List jobs, with the scope and format below.
- `--version`: Print the running binary version.
- `--status <job-id>`: Inspect an async job.
- `--result <job-id>`: Fetch an async job's current or terminal result.
- `--wait <job-id>`: Stay attached until an async job has a terminal result,
  making it suitable for native host process monitors.
- `--cancel <job-id>`: Cancel an async job by marking terminal state. On Unix,
  it sends SIGTERM to the async process group, then sends SIGKILL after a
  5-second grace period when the group is still running.

## Job Messages

```text
peeragent say ["<text>"] [--prompt-file <path>]
peeragent tell <job-id> [--cwd DIR] ['<text>'] [--prompt-file <path>]
peeragent inbox
peeragent messages <job-id> [--cwd DIR] [--json | --text]
```

`PEERAGENT_JOB_ID` and `PEERAGENT_JOB_DIR` identify the current job. Every target
receives them; inherited terminal addresses remain stripped by default. `say`
and `inbox` use that identity. `tell` uses the normal job index/explicit `--cwd`.
Messages use literal argv, never shell evaluation. Use `--` before literal text
beginning with a dash. Commands inside a job first announce unread entries on
stderr, preserving machine-readable stdout. `inbox` emits unread `{time, from,
text}` entries as JSONL and marks them read under the same lock as `tell`.
Finished jobs reject `tell` and suggest `--resume`. `say` uses a captured Claude
Code native inbox or a parent peer job's mailbox, preserving its fixed origin
label. It never submits terminal input. Unsupported or ambiguous launchers and
failed transports retain the question as `queued` in the job's result
conversation, with an explicit receipt and no automatic retry. Native socket
submission has no acceptance/read acknowledgement.

`messages` prints both mailbox directions, timestamped and ordered oldest first,
without acknowledging or consuming inbox replies. Default output is text;
`--json` prints JSON Lines, one object per message. Outbound outcome updates are
collapsed by message ID to their latest state. A removed worktree may retain
conversation history in its full terminal archive; compact indexed metadata has
no mailbox history. `say` accepts at most 2,000 Unicode characters and three
native delivery attempts per five minutes. Rate-limit refusal names the exact
UTC time the next send is allowed; the limit itself is unchanged.

The standard prompt includes these instructions for checking and sending messages:

Use `peeragent say` only to ask a question you can't settle yourself, never for status or progress updates.
Your final handback carries the final status, so don't repeat it in a message.
After asking a question, watch your inbox for the answer: run `peeragent inbox` every few minutes, between steps,
until it arrives, and keep doing independent work meanwhile.

`--no-preamble` persists in the execution spec and suppresses the whole standard
execution/messaging preamble. Resumed prompts include unread entries from prior
jobs for that agent/session; capacity continuations include their own unread
inbox. These entries are reserved by their attempt and hidden from inbox output,
unread notices and status counts while the target runs. Target output and a
non-capacity outcome mark them read; startup failure, cancellation, interruption and capacity/login
refusal release them to unread. Later arrivals remain independently readable.
A dead worker's unresolved reservations are released on the next mailbox read;
a terminal acceptance record instead commits them read. Directory aliases are
deduplicated by job ID.

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

New jobs record `messaging_enabled` in `job.json`. For older jobs without that
field, a saved messaging preamble or an outbound `say` entry proves messaging
support. Controls report `messaging_enabled: true` and `messaging_source` as
`inferred from prompt` or `inferred from outbox`, and show that source in text
and delivery confirmations. Explicit recorded flags remain authoritative.
Only jobs with neither evidence source get the pre-messaging warning; delivery
still succeeds and replies are carried on `--resume`. `--no-preamble` jobs get
a specific missing-preamble warning. Saving a legacy job's lifecycle preserves
an absent marker so later controls can still infer support from evidence.

Single-job status adds `metadata.inbox_unread`, `metadata.outbox_sent`, and
optional `metadata.last_message_at`; multi-job records carry the same fields
at the top level. Times are UTC RFC3339 with fractional seconds. Results add
`conversation` after `details`: entries ordered by `time`, with `direction`
(`inbox`/`outbox`), `from`, `text`, and outbound `status`/optional `reason`.
Text results put the conversation after the agent message. See
[Messaging](MESSAGING.md) for the fixed header, limits and platform behavior.

## Result JSON

The default result is JSON:

```json
{
  "job_id": "job-id",
  "status": "success",
  "summary": "Claude implementation completed",
  "final_message": "Claude implementation completed",
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
    "job_id": "job-id",
    "log_path": "/repo/.peeragent/runs/..."
  }
}
```

`final_message` is the job's complete final agent message as a plain string,
never trimmed, so scripts can read the handback without parsing `details`.
`summary` keeps its 600-character preview and `details` keeps its
`stdout:`/`stderr:` shape. `final_message` is `null` while a job runs (until
the target reports its final message), after a failed or resumable stop, and when the target produced no agent message, such as a
failure that printed only diagnostics. A failed job keeps its last agent
message in `details` and leaves `final_message` null. Its summary names the stop
cause rather than being overwritten by an interim message. `--result`, `--wait` and blocking calls fill it in; launch and
`--status` results carry `null`. Text output already shows the message under
details, so it adds no separate section.

When available, `metadata.agent_session` contains the target-agent session id
that can be passed back with `--resume`. Every CLI launch, including blocking
and resumed calls, carries top-level `job_id` and matching `metadata.job_id`, and persists tracked job state.
Single-job status and results mirror the same ID, including older stored results.
`--wait` JSON stdout contains exactly one document: the terminal result. It never
prints intermediate status; diagnostics and unread-message notices use stderr.
Text results print `Job: <job-id>` on its own line, with a `peeragent --wait`
command while running; blocking text calls also print the ID to stderr before waiting.

`status` values:

- `success`: Target completed successfully.
- `failed`: Target or wrapper failed.
- `not_found`: No job under the searched folder and no indexed location; lookup exit `4`.
- `blocked`: Reserved for explicit target-reported blockers.
- `running`: Async job started or is still running, including capacity backoff.
- `capacity_exhausted`: Codex reported “Selected model is at capacity”; terminal
  after the optional retry budget, or immediately when no retry/session is available.
  `metadata.agent_session` carries the resumable conversation when known.
  `metadata.capacity_retries` records additional attempts actually made.
- `usage_limited`: Codex hit its usage limit; terminal, with no automatic retry.
  The summary and `failure_reason` preserve the exact provider message and retry
  time when present, including when the limit occurs after a capacity retry.
- `output_limit`: Codex stopped at `max_output_tokens`; terminal unless the shared
  retry budget permits its one immediate continuation. Session and resume command
  are reported as described below.
- `auth_required`: Codex reported a revoked or invalidated login. Terminal
  unless the shared async retry budget permits its one immediate continuation
  using the current login. The conversation remains resumable when known.
- `interrupted`: The recorded worker is gone, belongs to a prior boot, or its
  PID belongs to a different process; or Codex could not record rollout items
  because its thread was not found, with no structured terminal outcome. Preserve the session and
  resume manually; this status is never retried automatically.
- `cancelled`: Async job was cancelled.

Failed results expose `failure_reason`: the last provider error event's message,
or the wrapper failure summary when no provider message is available. A bare
`turn.failed` retains the preceding error and classification. Single-job status,
job listings, archives and the compact index retain this reason.

A structured Codex cybersecurity refusal remains `failed`, with
`failure: policy_refusal`. Its summary explicitly names OpenAI's safety filter
and warns that resuming this session is likely to fail the same way; use a fresh
session for unrelated steps. `metadata.stop_reason` is the stable value
`provider_safety_filter`, and `metadata.stop_message` preserves the exact provider
message. Worker-loss recovery uses `metadata.stop_reason: worker_interrupted`. Single-job status and job listings also expose
this classification. An explicit structured terminal outcome takes precedence
over stderr diagnostics, including rollout-recording errors.

## Text Result

Text output starts with:

```text
Alt Subagent: <status>
```

It includes summary, changed files, verification, details, and metadata sections
when available.

## Working Directory

The default working directory for new tasks is the current host directory.
New job records live in the checkout's Git directory (`git rev-parse --git-dir`),
under `peeragent/jobs/`. Linked worktrees have separate job folders. Outside Git,
records live under `.peeragent/jobs/`. Every lookup also reads existing
`.peeragent/jobs/` records, including session scans, messaging and listings;
updates to an existing job stay in its original folder. The working directory
recorded in metadata and the index remains the target checkout, not its Git directory.

Job controls without `--cwd` first look up the job id in the per-user index.
Resume without `--cwd` first looks up the agent/session pair recorded at the
last result, including blocking and failed results. Missing entries fall back
to the current directory, preserving access to older repository-local jobs.
Index read errors explain the `--cwd` override; index write errors report
`metadata.index_warning` and allow execution to continue. An explicit `--cwd`
is searched first; if the job is absent and indexed elsewhere, the indexed folder
is used with `metadata.lookup_warning` naming both paths. Final result JSON is also archived under `index/results/` for
30 days. `--result`, `--wait`, single-job `--status`, and `--cancel` use this copy
when repository job metadata is absent, including after the worktree is removed.
Archives expire after 30 days; every archive write prunes older snapshots.
Prompt files and raw log files are not copied, and repository job folders keep
their existing lifetime. Jobs with neither a retained snapshot nor indexed metadata need their original
files. When full result archives are unavailable, `--status` and `--result`
return the compact indexed summary with `metadata.indexed_only: true` and an
explicit `metadata.index_warning`; full message/details are absent.
`metadata.started_at` and `metadata.finished_at` carry retained times.
`--wait` returns indexed-only history immediately rather than polling a lost
worktree indefinitely. To resume in a replacement worktree, pass its existing
path with `--cwd`; saved model and effort still apply. Indexed-only records are
excluded from live-writer checks and old-mailbox carryover, so reads and resume
do not recreate removed worktrees. Corrupt or unreadable local state is still
an error; the archive never masks it. After index resolution, the selected
folder must match the archived result's directory, comparing
case-insensitively on Windows and default macOS filesystems, and exactly on
Linux. A successful status lookup reports exit code zero regardless of the
archived target exit code; an absent job still reports lookup exit code four.
The target operates in that same checkout. The wrapper does not create a worktree,
clone, or sandbox copy unless explicitly requested.

## Target Invocation

Codex (when a Codex model is selected):

```text
codex exec --json --cd <repo> --sandbox workspace-write \
  --model <gpt-6.1-<sol|terra>|gpt-6-<astra|luna|sol>|gpt-5.6-<luna|terra|sol>> \
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

For Codex, the short aliases `astra`, `luna`, `terra`, and `sol` normalize to
and pass through as `gpt-6-astra`, `gpt-6-luna`, `gpt-6.1-terra`, and
`gpt-6.1-sol`. Explicit `gpt-6-sol` and GPT-5.6 Luna, Terra, and Sol IDs
are also accepted.
For Claude, the wrapper passes `--model <alias>` to
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
`metadata.log_path` when that field is present. Lines beginning with the Codex
diagnostic prefix “Orphan custom tool call:” are filtered from the visible stderr
summary, while raw logs retain them. Other diagnostics mentioning that phrase
remain visible.

## Multiple Jobs

`--wait-any <job-id>...` validates every reference before waiting. Unknown ids
are usage exit `2`; storage/decoding errors remain infrastructure failures.
Already terminal jobs return immediately, with ties in argument order. The
selected job's ordinary result includes `metadata.job_id` and retains `--wait`
exit behavior: terminal target outcomes exit `0`, with the target's outcome in
`status` and `metadata.exit_code`.

`peeragent --jobs [--cwd DIR] [--all] [--json | --text]` lists every job in
that directory, including finished jobs and unindexed local jobs, newest first.
`--all` includes retained metadata from removed worktrees as well as live
indexed directories. Removed-worktree rows carry `indexed_only: true` and an
`index_warning` explaining that local files/logs are unavailable. They retain
id, agent, model, effort, cwd, last known status, `agent_session`, final summary,
creation time and `finished_at` when finished. Last-known running status is not
proof of a live process. It does not search arbitrary directories. JSON output is one object
per line, using the multi-job status fields: job id, agent, model, status,
`started_at` (creation time), `last_message_at` when available, and
`agent_session`. An empty JSON list produces no lines; `--text` prints a table.
`peeragent --version` prints the running binary's version. Status JSON carries
that version in `metadata.version` for one job and `version` in each multi-job
or listing row, even when the job was started by an older binary.

Codex `max_output_tokens` stops return the terminal status `output_limit`,
with the captured `agent_session` and a resume command in the summary. Under
`--async --agent codex --retry-capacity N`, an output-limit stop consumes one
of the same N additional attempts as capacity recovery. It resumes the same
session immediately with a short continuation asking for a concise handback;
`--capacity-backoff` applies only to capacity stops. Each job gets at most one
automatic output-limit continuation, even if budget remains. Without a session,
with N=0, after that continuation, or after the shared budget is spent, the stop
is terminal. `metadata.capacity_retries` retains its existing name and counts
all additional attempts, including output-limit and login continuations. Other failures
are never retried.

`--status <job-id>...` with multiple ids emits one compact JSON object per line,
with `version`, `agent`, `messaging_enabled`, `job_id`, `status`, `model`, `effort`, `agent_session`, `started_at` (launch
time), `capacity_backoff`, `final_received`, `cwd`, and optional `head_commit`,
`last_event_at`, `last_message` and `summary`. Live timestamps use UTC RFC3339
with fractional seconds; message previews are one short line. Single-id status
adds `metadata.last_event_at` and `metadata.last_message`. Capacity status/results
put “stopped: model at capacity; resumable with --resume <session>” in the summary,
ahead of the last agent message (omitting the resume hint if no session is known).
`--text` renders a table. HEAD is read once per worktree, without walking changed
files; missing git/HEAD omits the commit. Single-id output remains a Result.
Missing or unreadable batch-status jobs produce an error before printing rows.

Both controls accept `job-id@directory` references. A per-job directory wins
over `--cwd`; without one, explicit `--cwd` is searched before indexed lookup.
A missing job at that location can use its indexed folder with a lookup warning. An
unindexed id falls back to the current directory. Quote references containing
spaces. Relative directories resolve against the caller's directory.

## Exit Codes

- `0`: Completed successfully. For async launch, this means the job started.
- `1`: Target failed or exited non-zero.
- `2`: Wrapper usage error or unsupported mode.
- `3`: Wrapper binary unavailable for this platform.
- `4`: Async job lookup failed.

## Job State

Blocking and async CLI launches use the same tracked worker. Their state is
stored under:

Blocking callers retain the native worker handle and collect its exit while
watching the result file. A worker exit without a terminal result records
`failed`, the worker's exit status in `metadata.exit_code`, and its `agent.log`
path in `metadata.log_path`. Even an exit status of zero without a terminal
result is a failed job; the blocking CLI returns exit `1`. No wall-clock
failure deadline is used. A terminal result or cancellation already recorded
keeps precedence. Normal terminal results are read without being rewritten.
Blocking callers read persisted completion without reconciling worker loss;
their owned handle supplies the definitive exit code and log. External
observers retain stale-worker recovery. Collected worker exits remove the PID
sidecar. Unix liveness checks also ignore
exited, uncollected workers (Linux process state; macOS native exit events).
Windows retains the worker's native handle until its wait completes.

```text
<git-dir>/peeragent/jobs/<job-id>/
  job.json       lifecycle + ExecSpec + worker boot/PID/start identity + mailbox references + live observations
  prompt.txt     resolved task text, parent-written, child-read
  launcher.json  optional private native launcher channel (legacy Herdr fields readable), mode 0600
  inbox.jsonl    launcher replies {time, from, text}, mode 0600
  inbox-state.json  read prefix + independently read/reserved entries, mode 0600
  inbox-attempts/   per-attempt worker PID and running/accepted/released outcome, private
  outbox.jsonl   timestamped outbound messages/outcomes, mode 0600
  pid            child PID/PGID for cancel, present while running
  agent.log      live compact Codex progress JSONL + background wrapper diagnostics
  target.log     raw target stdout+stderr written after the attempt exits
  message.txt    latest completed agent message, written as received (untruncated)
  result.json    final result, written by child, cancellation, or lost-worker recovery
  lock.guard     persistent kernel-lock inode; lock contains the transient owner identity
```

The private per-user lookup index is stored under
`<os.UserCacheDir()>/peeragent/index/{jobs,sessions}/`, or
`$PEERAGENT_STATE_DIR/index/{jobs,sessions}/` when overridden. Keys are hashed;
session values are absolute working directories. Job values are compact JSON
metadata; legacy path-only job entries are still accepted. Index directories use `0700` and
files `0600` on Unix. Independent atomic entries support concurrent launches
without one shared read/modify/write file. The index can be deleted; explicit
`--cwd` remains usable. Job entries are written before launch and session
entries when a target session is observed. Short per-session launch locks live
in `index/session-launch-locks/` and hold a launcher boot/PID/start identity for diagnostics. The
job index preserves metadata independently of the original worktree. Snapshots
are updated at launch, when worker identity or a session is learned, and on completion. Summaries
are capped at 600 Unicode characters. Launch/completion prune entries older
than 30 days and retain at most 10,000, newest updates first; live progress does
not scan the index to prune it. Full terminal snapshots in `index/results/`
keep their separate 30-day lifetime and existing contents. Listing or resuming
promotes older terminal archives into metadata, skipping complete snapshots
already represented there. Older snapshots without explicit times use the job
ID timestamp for creation and the archive file modification time for completion.
Archive failures warn through `metadata.index_warning` and leave the
repository completion intact.
On Windows, transient sharing or access-denied errors during index replacement
are retried up to ten attempts with at most 450ms of delay. Other errors return
immediately; persistent write failures still warn without blocking delegation.

`job.json` persists the capacity retry limit and backoff in `spec`, and updates
`spec.resume` before retrying. During backoff the job stays running. A retry
uses a continuation prompt and the same session, model, access, effort and job
id. Every attempt gets a separate raw log; the final result points to its own
attempt's log. Only explicit Codex capacity, output-limit, and revoked-login diagnostics
trigger the corresponding retries, never ordinary failures or a final agent
message discussing them. Login stops consume at most one immediate continuation
within the same N-attempt budget, using the current login without changing
credentials. A repeated auth stop is terminal even if budget remains. The last structured
`turn.failed` or `turn.completed` event determines the outcome, overriding earlier
diagnostic history. Stderr recovery diagnostics are considered only when no structured
terminal event exists; historical or quoted mentions do not qualify. The known
“Error running remote compact task:” diagnostic prefix is unwrapped before
checking for capacity, including on `turn.failed`. The existing
cancel/finish lock protects terminal state from later retry completion.

`--cancel` writes `job.json` and `result.json` as cancelled before signalling.
On Unix, it signals the process group recorded by `pid` only after verifying
its saved process start time and boot identity. Missing, mismatched, or legacy
identities without a start time skip signalling while retaining cancelled state.
Verification repeats before escalation; a narrow exit/reuse race remains between
the verification and signal kernel operations.

Worker records use `/proc/sys/kernel/random/boot_id` where available and a PID
plus process start time on other supported platforms. Status, result, wait,
wait-any and resume reconcile a provably lost running worker to terminal
`interrupted` under the same writer lock as completion. A stored terminal result
wins. The session, full last message and progress preview survive; unavailable
probes alone never prove death, and the launch's pre-PID window remains running.
Compact indexed metadata retains worker identity for reads after worktree removal.
Job and session locks use kernel locks released at process death. Owner files
left by provably dead holders are recovered under that lock; unknown/live owners
retain the bounded lock wait and manual removal or concurrent-session override.

## Guarantees

The wrapper:

- Preserves arbitrary task text as data.
- Does not run full access unless explicitly requested.
- Does not silently switch target agents.
- Does not retry failures except the documented capacity, output-limit and login
  continuations when async Codex retries are explicitly enabled.
- Reports the target exit status when available.
- Stores raw target stdout/stderr at `metadata.log_path` when output exists.
- Reports compact stdout/stderr details, preferring the final peer message when
  the target exposes one.

The wrapper does not guarantee that target-agent output is complete,
well-structured, or free of partial edits before failure.

## Live Codex Completion

Codex JSONL is observed as it arrives, without a line-size cap. Session ids are
persisted in `spec.resume` immediately and remain visible to status/result after
failure or cancellation. Completed agent messages are saved without truncation
in `message.txt`; results include them or provide `metadata.message_path`, even
when a subsequent capacity attempt has no message. This file holds the latest
completed message, which may be a progress update if interrupted before a final
reply. Raw per-attempt logs remain separate.

`agent.log` appends a compact JSON record for each structured Codex event as it
arrives: `at` (UTC timestamp), `event`, and `last_message` (the latest short
preview). Tool activity retains the previous message preview. Status reads a
bounded 8 KiB tail, skipping partial writes, instead of loading entire logs or
reports. This keeps live progress visible without duplicating full reports.
Retries retain the preceding attempt's last message until a new message arrives.

Before a resume starts, peeragent scans jobs in the requested worktree and
worktrees located by the job/session index, checking nonterminal jobs for the
same agent/session and a matching live worker identity. A per-session launch lock makes the
check and worker launch atomic relative to other resumed launches. This prevents
lost or interleaved work in one session. It is not an authority over unknown
worktrees: after clearing the index, use the original worktree with `--cwd`.
It never signals processes during the check. Native liveness checks are separate
on Unix and Windows; the intentional override is `--allow-concurrent-session`.

Terminal `turn.failed`, `task_complete` (including `event_msg` envelopes),
and messages marked `final`/`final_answer` start a five-second grace period.
`turn.completed` starts a provisional grace: a following `turn.started`,
`item.started`, or other agent/tool activity cancels it and clears
`metadata.final_received`. A later completion starts fresh grace. Explicit
session completion is not downgraded by a later turn marker. Unphased
`item.completed` agent messages may be progress and do not start shutdown.
During grace status says “final message received, process still alive” and sets
`metadata.final_received`.

After confirmed completion grace, peeragent stops only its owned process group
(Job Object on Windows), closes its own stdin writer and output readers, joins all input/output copies,
reaps the child and derives
its outcome from the last terminal event. Linux and macOS observe leader exit
without reaping, keeping the group number reserved until its last possible
signal. Group signalling is revoked before reap, so cleanup cannot target a
reused group number. Before completion or cancellation, input/output copying has no
timeout, including when a launcher exits ahead of its worker. Pending stdin
copying remains cancellable and cannot move the runner into a blocking reap. Explicit
cancellation still wins through the existing terminal lock. A JSONL line without
a trailing newline is observed when its stdout stream reaches EOF.

`job.json` also records `capacity_backoff_until` and the count of retries started.
The job remains `running` during backoff; compact multi-job status exposes it.


## Active installed wrapper

The POSIX wrapper keeps `PEERAGENT_BIN`, a local `dist/peeragent`, and a Go source
checkout ahead of installed binaries. A Claude Code cache wrapper reads
`${CLAUDE_CONFIG_DIR:-$HOME/.claude}/plugins/installed_plugins.json` for its
marketplace's active peeragent install and execs that install's wrapper when it
differs. An internal `PEERAGENT_ACTIVE_INSTALL` marker allows only one forwarding
step and is removed before running a binary, so child commands can resolve
updates independently. Existing sessions with an older cache directory on PATH
pick up active updates after that older wrapper contains this forwarding logic.

The wrapper parses the record with Python 3, or jq when Python 3 is unavailable.
Absent or unreadable records, unavailable JSON readers, and missing active
wrappers leave the current install usable. Codex has no equivalent install-path
record; its wrappers keep their existing selection. Separate harness caches
never redirect through each other's records.
