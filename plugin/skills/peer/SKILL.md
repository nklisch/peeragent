---
name: peer
description: >
  Delegate arbitrary task work — implementation, research, review, debugging,
  design, doc updates, anything — to another local coding agent through the
  bundled peeragent wrapper. Use when the host assistant wants a peer agent
  to take a focused pass on a task in the current repository. Targets: Codex
  (default), Claude Code, Gemini through Antigravity, or Z.AI GLM 5.2 through
  Pi. Free-form task text; the wrapper returns a JSON result the host summarizes
  for the user.
  Resolve the wrapper from the plugin location before invoking; do not
  assume `peeragent` is on PATH.
allowed-tools: Bash
metadata:
  short-description: Delegate a task to a peer coding agent
---

# Peer

Delegate a focused task pass to a different local coding agent while you
stay responsible for the user conversation. Use this any time a peer
agent would do the work better, faster, or with usefully different blind
spots — implementation, research, review, debugging, refactors, doc
updates, build fixes. Not just "implementation" despite the historical
name.

## Default Behavior

Do not assume `peeragent` is on `PATH`. Resolve the bundled wrapper before
the first call and use that path for every invocation:

- If `PEERAGENT_BIN` names an executable, use it.
- Otherwise resolve from this skill file: go two directories up from the
  skill directory, then use `bin/peeragent`.
- In a development checkout, the same wrapper is `bin/peeragent`.
- Use bare `peeragent` only if the bundled path cannot be found and
  `command -v peeragent` succeeds.

Claude Code cache wrappers follow the active install in its installed-plugins
record, including when an existing session keeps an older cache path on PATH.
The wrapper needs Python 3 or jq to read the record and otherwise keeps its
current install. Explicit binary and source checkout paths retain precedence.
Codex has no equivalent install-path record, so its wrapper selection is unchanged.

If a bare `peeragent` call fails with `command not found`, retry once with
the bundled plugin path before reporting failure.

For invocations that launch or control a peer CLI (`--agent ...`, `--async`,
or `--cancel`), run the wrapper with the host harness's outside-sandbox command
mode so it inherits the developer machine's normal network, auth, hook, and
process environment. In Codex, use the command tool's escalated/outside-sandbox
execution mode for that wrapper invocation. Pure local `--status` and
`--result` reads may stay sandboxed when the job directory is readable. This is
separate from peeragent's `--full-access` flag: do not pass `--full-access`
unless the user explicitly asked for that target permission mode.

Pass the user's task text to the wrapper. Prefer an asynchronous delegation for
substantive work:

```bash
<resolved-peeragent-bin> --async --agent <codex|claude|gemini|zai> [--model <name>] "$ARGUMENTS"
```

`--agent` defaults to `codex` if omitted. Read the launch JSON, retain its
top-level `job_id` (also retained as `metadata.job_id`) and repository
`metadata.cwd`, and follow the asynchronous lifecycle below.
Use a blocking call only for a genuinely short pass or when the host cannot
reliably complete the async lifecycle.

## Picking The Target

The right peer is the agent you are not, unless the user named one.

- **If you are Claude Code** → default to
  `--agent codex --model luna --effort high`. Use `--agent gemini` or
  `--agent zai` when the user asks for that model, when Codex isn't available,
  or when extra model diversity is useful.
- **If you are Codex** → default to `--agent claude`. Use
  `--agent gemini` or `--agent zai` when the user asks for that model, when
  Claude isn't available, or when extra model diversity is useful.
- **If the user named the target** ("ask Codex…", "have Gemini look
  at…", "ask GLM…", "use Z.AI…"), use that one.

Different blind spots are the point. Their misses are your catches and
vice versa.

## Effort And Model

The `sol` alias selects `gpt-6.1-sol`; `luna` selects `gpt-6-luna`, and
`terra` selects `gpt-6.1-terra`. Explicit `gpt-6-sol`, `gpt-5.6-luna`, `gpt-5.6-terra`, and
`gpt-5.6-sol` IDs remain supported.

Use GPT-6 for routine Codex work. Luna is the routine fast path; Sol is the
higher-capability GPT-6 tier. GPT-6 Astra is the flagship jump above Sol for the
most difficult work. Terra is a useful bridge, but most calls can jump
directly from Luna to Sol or Astra.

| Desired pass | Codex recommendation | Claude equivalent |
| --- | --- | --- |
| Routine / fast | `--model luna --effort high` | Sonnet-class |
| Lots of work | `--model luna --effort xhigh` | Strong general pass |
| Optional bridge | `--model terra --effort high` or `xhigh` | Between general and flagship |
| Opus-tier | `--model sol --effort low` or `medium` | `--model opus --effort xhigh` |
| Fable-tier | `--model sol --effort high` or `xhigh` | `--model fable --effort high` or `xhigh` |
| GPT-6 flagship | `--model astra --effort high` or `xhigh` | Above Fable-tier |

Claude also retains Opus, Sonnet, and Haiku aliases. Claude rejects `--effort medium`;
peeragent exposes only `high|xhigh` for Claude. Gemini defaults to Gemini 3.7
Flash at high effort and passes both model and effort to `agy`; Flash accepts
`low|medium|high`, while `--model pro` accepts `low|high`. Z.AI maps
`--effort medium|high|xhigh` to Pi `--thinking` and accepts only `--model
glm-5.2`; no other Z.AI model is surfaced.

Quickly test whether the Z.AI target is configured before delegating:

```bash
pi --list-models zai | grep -w 'glm-5.2'
pi --provider zai --model glm-5.2 --thinking high --no-session --no-tools -p 'Reply with OK.'
<resolved-peeragent-bin> --agent zai --text 'Reply with OK and do not edit files.'
```

If the Pi smoke test fails, configure Pi with `ZAI_API_KEY` or `/login` for the
ZAI provider and retry.

## Delegation Contract

The host delegates task intent; the peer performs the focused pass. The
task text is arbitrary natural language, not shell syntax. Don't split
the request into many wrapper calls unless the user explicitly asked for
separate passes.

Target agents start without inherited terminal-control addresses by default.
Pass `--keep-terminal-env` only when intentionally delegating work that needs the
launching terminal's control environment; job metadata records the override.
Ordinary settings and authentication remain available. The repository's
`docs/TERMINAL_ENV.md` is the canonical variable list and scope.

## Job Messages

For a running job, send a reply with `<resolved-peeragent-bin> tell <job-id>
[--cwd DIR] '<text>'` or `tell <job-id> --prompt-file <path>`. Finished jobs require `--resume`. Replies are a mailbox:
the job reads them with `peeragent inbox` after long steps and before handback.
Every command inside a job announces unread replies on stderr.
Use single quotes or a prompt file: backticks in double quotes execute in the shell before `tell` receives the text.

Both `tell` and `say` accept `--prompt-file <path>`. Use a file for long or
multiline messages. Like launch, positional text plus file input is accepted:
trim each source and join positional text first, then file content, separated
by a blank line. The file path is relative to the caller, independently of
`tell --cwd`; messaging does not read stdin.

A job can use `peeragent say "<text>"` to submit a message to its recorded
Claude Code native inbox, or its parent peer job's existing mailbox, with a
mandatory origin and reply header. This channel is separate from typed input; peeragent never submits text through a terminal pane.

Use `peeragent say` only to ask a question you can't settle yourself, never for status or progress updates.
Your final handback carries the final status, so don't repeat it in a message.
After asking a question, watch your inbox for the answer: run `peeragent inbox` every few minutes, between steps,
until it arrives, and keep doing independent work meanwhile.

Never use raw terminal, desktop or keyboard input to contact the launcher.
Outbound messages have a 2,000-character
limit and a three-per-five-minutes job limit; refusal directs the job to its
handback. Without a safe native channel, including Herdr-only or unsupported
launchers, the message is retained as `queued` in the job's `--result` conversation.
The receipt says queued; there is no automatic retry. Continue independent work
and include the unanswered question in the handback. A native submission has no
acceptance/read acknowledgement. `--status` shows mailbox
counts/time; `--result` includes the ordered conversation after the agent message.
Resumes carry unread replies into their prompt. `--no-preamble` suppresses the
standard execution/messaging instructions and sends task text only.

Labels and the environment strip help well-meaning agents avoid accidental
unlabelled input. A deliberately misbehaving job runs as the same user and can
read the launcher state or contact local sockets directly. The preamble
and `say` are the supported path, not an adversarial boundary.

Replies carried in a prompt are reserved by that attempt and hidden from its
inbox, unread notice and status count while it runs. Target output and a
non-capacity outcome mark them read. Startup failure, cancellation and capacity
refusal release them to unread; a dead worker's unresolved reservation is
released on the next mailbox read. Later replies remain independently readable.

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

## Async Lifecycle

Prefer `--async` for normal implementation, research, and review passes so a
long-running peer does not consume or exceed the launch tool call. For async
Codex work, add `--retry-capacity N` to opt into recovery from capacity and
output-limit and login stops. It is disabled by default and resumes the same session.
Capacity retries wait `10m` unless `--capacity-backoff` overrides it. Retry options require
`--async --agent codex` even with `--resume`; add `--async` to a resumed call.
For example:

```bash
<resolved-peeragent-bin> --async --agent codex --retry-capacity 12 --prompt-file <brief>
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
all additional attempts, including output-limit and login continuations. Other failures
are never retried.

Every launch carries a job id in JSON and text, including resumed calls. A
resume refuses with exit `2` while another job for that agent/session has a
live recorded process; follow its wait/cancel instructions. Use
`--allow-concurrent-session` only for intentional overlap. Capacity results
and status explain the stop and how to resume before the last agent message.

After the
launch returns a job id, start peeragent's attached waiter:

```bash
<resolved-peeragent-bin> --wait <job-id> [--cwd <same-cwd>]
```

`peeragent --jobs [--cwd DIR] [--all] [--json | --text]` lists every job in
that directory, including finished jobs and unindexed local jobs, newest first.
`--all` includes retained metadata for removed worktrees as well as live indexed
directories. Such rows carry `indexed_only: true` and a clear warning; they
preserve id, agent/model/effort, cwd, status, session, final summary and finish
time. They do not prove that a last-known running job still has a live process.
It does not search arbitrary directories. JSON output is one object
per line, using the multi-job status fields: job id, agent, model, status,
`started_at` (creation time), `last_message_at` when available, and
`agent_session`. An empty JSON list produces no lines; `--text` prints a table.
`peeragent --version` prints the running binary's version. Status JSON carries
that version in `metadata.version` for one job and `version` in each multi-job
or listing row, even when the job was started by an older binary.

For several outstanding jobs, attach one monitor to
`--wait-any <job-id> <other-job-id>`. It validates every id before waiting and
prints the first terminal result with its job id. An unknown id is usage exit
`2`; already terminal jobs return immediately, with ties in argument order.
Terminal outcomes retain `--wait`'s exit `0`; read the result status to learn
whether the target succeeded. Rearm with the remaining ids after completion.

`--status <job-id> <other-job-id>` prints one compact JSON record per job;
`--text` prints a table including model, effort, session, start time, capacity
backoff, worktree HEAD, last agent event time and a short message preview.
`agent.log` fills live with compact Codex events; status reads a bounded tail.
A single id keeps the usual result with live fields in metadata.
For either multi-job control, use `job-id@directory` for an unindexed job;
quote the whole reference for paths with spaces. Per-job directories win over
`--cwd`, which otherwise overrides indexed lookup.

When the host harness provides native background jobs, monitors, or completion
wake-ups, run that `--wait` command through the native facility and rely on its
completion notification. The waiter remains attached to the detached peer job
and prints the terminal result when it finishes, giving the harness a real
process to monitor. Do not hand-roll a sleep loop or manually poll while the
monitor is active.

If the harness has no monitor facility, run `--wait` in the foreground when its
tool timeout is sufficient. Otherwise check `--status <job-id>` only at
reasonable checkpoints and call `--result` once the job is terminal. Never
tight-poll. Do not give the user a final success or failure report until you
have read the terminal result, unless the user explicitly asked only to launch
the job.

For a short blocking call, wait for the command to return and apply the same
result handling directly.

The default `details` payload is compact. For Codex targets, peeragent surfaces
the final completed agent message instead of the full stream of interim assistant
messages. Read the complete handback from the top-level `final_message` string;
it is `null` while the job runs, after a capacity or output-limit stop, or when
the target produced no message. If a deeper look is needed, inspect `metadata.log_path` for raw target
stdout/stderr, or use `metadata.agent_session` with `--resume` for continuity.
Codex records session ids as they arrive, including while running and after
cancellation. `--result` preserves completed messages without truncation, or
provides `metadata.message_path` for a message from an earlier attempt. A
terminal event or explicitly final message starts a five-second grace period;
status reports “final message received, process still alive” until exit. After
the grace period peeragent ends only its own target process tree and uses the
terminal event's outcome. Unphased progress messages do not end a job.

Claude Fable and Opus runs, especially with `--effort xhigh`, may take 10
minutes or longer to finish. A slow or quiet flagship reply is not by itself
evidence of a hung process; keep waiting unless the wrapper exits, reports
failure, or there is concrete evidence the process is stuck.

Include a no-recursion instruction in every handoff. Tell the peer
explicitly **not** to reach for peeragent's `peer` skill — and not to run the
`peeragent` wrapper — to delegate the work back
out to another local coding agent. That would loop host → peer → host and
burn budget. The peer is the endpoint of this delegation. It may freely use
its own harness's internal sub-agents (Codex/Claude/Gemini built-in task or
sub-agent tooling, or Pi-native extensions if available) for parallel or
focused passes; the prohibition is only
on recursive peeragent calls. A short line in the prompt is enough, e.g.
"Do this work directly — do not call the peeragent peer skill to hand it off
again; use your own harness's sub-agents if you need help."

Job controls locate newly launched jobs by id from any directory; `--cwd`
remains an explicit override. Resume uses the directory last recorded for that
agent/session unless overridden. Older unindexed jobs/sessions still need their
original directory or `--cwd`. Index write failures appear as
`metadata.index_warning`; use `--cwd` when the lookup index is unavailable.

Unknown options (including `--list` and `--background`) and empty/missing option
values exit `2` before any job starts. Put `--` before literal task text that
starts with a dash. Validate session variables before constructing a resume call.

Job records live under `<git-dir>/peeragent/jobs/` in a Git checkout, separately
for each linked worktree. Outside Git they live under `.peeragent/jobs/`.
All controls, message commands, listings, resumes and indexed lookups also find
existing `.peeragent/jobs/` records without moving them.

## Result Handling

- `status: success` — report what changed and what verification ran.
- `status: blocked` — explain the blocker; continue from the host side.
- `status: failed` — surface `failure_reason` (the last provider error message)
  and useful log details.
  - If exit code `3` (or, in `--text` mode, a "no prebuilt binary for this
    platform" message), peeragent has no committed binary for the user's
    OS/arch. Prebuilt binaries cover linux/darwin on amd64/arm64. Tell the
    user: on those platforms, reinstall the plugin or download the matching
    archive from https://github.com/nklisch/peeragent/releases; on any other
    platform, install from source (requires Go) with
    `go install github.com/nklisch/peeragent/cmd/peeragent@latest` and set
    `PEERAGENT_BIN`. If the
    platform is misdetected, `PEERAGENT_TARGET_OVERRIDE=<goos>-<goarch>`
    selects a present binary. Do not retry in a loop.
- `status: capacity_exhausted` — Codex stopped for model capacity. Preserve
  `metadata.agent_session` for resumption. Async `--retry-capacity N` opts into
  N additional attempts with a default ten-minute backoff (override with
  `--capacity-backoff <duration>`). Retries resume that session under one job id;
  they stop without a session id or when the budget is spent. Other failures
  are never retried except the single output-limit and login continuations.
  `metadata.capacity_retries` counts all additional attempts.
- `status: usage_limited` — Codex hit its usage limit. Surface `summary` and
  `failure_reason`, which preserve the exact provider message and retry time
  when present. `--retry-capacity` never retries this status, including after
  a capacity retry; wait until usage is available before resuming manually.
- `status: output_limit` — Codex stopped at `max_output_tokens`. Keep the session
  and resume command. The shared async retry budget permits one immediate
  continuation; a repeated limit is terminal.
- `status: auth_required` — Codex stopped because its login was revoked or
  invalidated. Keep `metadata.agent_session`. The shared async retry budget
  permits one immediate resume using the current login. If it still stops,
  refresh the login and resume manually; peeragent never changes credentials.
- `status: interrupted` — Codex could not record rollout items and had no
  structured terminal outcome. Keep the session for manual resumption; there
  is no automatic retry. A structured terminal outcome wins over stderr.
- `status: failed` with `failure: policy_refusal` — OpenAI's safety filter
  stopped the turn. `metadata.stop_reason: provider_safety_filter` is the stable
  stop identifier; `metadata.stop_message` contains the exact provider message.
  Resuming this session is likely to fail the same way because its history
  remains. Start a fresh session for unrelated steps. Do not retry the filtered
  work automatically; return the reason to the host.
- `status: running` — keep monitoring the async job; report only an interim
  update if useful, then retrieve its terminal result before concluding.
- `status: cancelled` — only after `--cancel`; report cleanly.

Do not claim success unless the wrapper reports `success`.

Compact job metadata is retained in the per-user index for 30 days, up to
10,000 entries, with final summaries capped at 600 Unicode characters. It keeps
removed-worktree jobs discoverable through `--jobs --all`; `--status` and
`--result` can return their indexed summary even without a full archive.
`metadata.indexed_only: true` and `metadata.index_warning` identify that fallback;
no final message, logs, mailbox or details are reconstructed. `--wait` returns
such last-known history immediately, including when its status was running.
Resume retains the saved model/effort; supply `--cwd <replacement-worktree>`
when the original directory has been removed. Indexed-only jobs do not add old
mailboxes to a resumed turn or count as evidence of another live writer.

Full terminal result JSON still has its separate 30-day archive and takes
precedence over the compact fallback. Listing or resuming promotes older
archives into metadata when needed. Prompts and raw log files are not copied;
repository job folders keep their existing lifetime. A legacy job with neither
metadata nor a retained archive still needs its original files.

## Options

Use advanced modes only when the request calls for them:

- `--full-access` — run the peer CLI without sandboxing. Never implied;
  ask the user first if the wrapper reports full access is needed.
- `--worktree` — reserved; returns a clear failure today.
- `--profile <name>` — Codex profile override.
- `--model <name>` — Codex aliases (`astra`, `luna`, `terra`, `sol`) or their
  canonical `gpt-6.1-<sol|terra>`, `gpt-6-<astra|luna|sol>`, and
  `gpt-5.6-<luna|terra|sol>` IDs;
  Claude aliases (`fable`, `sonnet`, `opus`, `haiku`); Gemini `flash`, `pro`, or a supported explicit family ID; or Z.AI
  `glm-5.2`. No other Z.AI models are accepted.
- `--resume <agent-session>` — continue a prior target-agent session when
  the previous result included `metadata.agent_session`. Use it for continuity
  inside one multi-pass workflow; omit it for an independent second opinion.
  Omitted `--model` and `--effort` keep the session's most recent peeragent job's
  settings; explicit flags override each option independently. If that job
  record is unavailable, omitted options use current defaults and
  `metadata.resume_warning` explains the fallback. Launch/result metadata shows
  the resolved settings; an absent Codex/Claude model means the native CLI
  default. The indexed session directory is reused unless `--cwd` overrides it.
- `--cwd <path>` — override the indexed job/session or default task directory.
- `--prompt-file <path>` — read task or `tell`/`say` message text from a file.
- `--async` — preferred for substantive work; starts a background job and
  returns a `job_id` immediately with `status: running`; add opt-in
  `--retry-capacity N` for Codex capacity recovery.
- `--retry-capacity <N>` — opt into at most N shared capacity/output-limit/login retries (async Codex only).
- `--capacity-backoff <duration>` — positive retry delay (default `10m`).
- `--jobs [--cwd DIR] [--all]` — list local jobs, or all indexed directories;
  JSONL by default, `--text` for a table.
- `--version` — print the running binary version.
- `--status <job-id>...` — check one job or get compact JSONL / a text table
  for several.
- `--result <job-id>` — fetch an async job's current or final result.
- `--wait <job-id>` — stay attached until an async job has a terminal result;
  JSON stdout contains exactly one document, the terminal result;
  run this through native harness monitors when available.
- `--wait-any <job-id>...` — wait for the first terminal job, validating all ids
  first. Accepts `job-id@directory` references, as does multi-job status.
- `--cancel <job-id>` — best-effort cancel.
- `--text` / `--json` — output format; JSON is default.

## Guardrails

- Keep the handoff concise. The peer receives the task, not a full
  transcript recap unless context matters.
- Do not run the same peer repeatedly in a loop after `failed` or
  `blocked` results — diagnose first. On exit code `3`, tell the user to
  install a prebuilt binary (linux/darwin amd64/arm64) or build from source
  with Go; do not retry.
- Preserve the host's responsibility for explaining outcomes to the
  user. The peer is the worker; you are the narrator.
- Treat the working tree as shared space — the peer may edit files you
  are also editing. Re-read before re-edit.
- No recursive peeragent. The handoff prompt must tell the peer not to call
  peeragent's `peer` skill (or the `peeragent` wrapper) back out to another
  agent. The peer's own internal harness sub-agents are fine; recursive
  peeragent delegation is not.
