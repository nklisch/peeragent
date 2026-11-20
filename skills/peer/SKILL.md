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

Delegate a focused task pass to a different local coding agent while you stay
responsible for the user conversation. Use this any time a peer agent would do
the work better, faster, or with usefully different blind spots —
implementation, research, review, debugging, refactors, doc updates, build
fixes.

## Default Behavior

Do not assume `peeragent` is on `PATH`. Resolve the bundled wrapper before the
first call and use that path for every invocation:

- If `PEERAGENT_BIN` names an executable, use it.
- Otherwise resolve from this skill file: go two directories up from the skill
  directory, then use `bin/peeragent`.
- In a development checkout, the same wrapper is `bin/peeragent`.
- Use bare `peeragent` only if the bundled path cannot be found and
  `command -v peeragent` succeeds.

When the machine's own instructions say that bare `peeragent` runs a locally
deployed build (for example, an owner who builds peeragent's main and installs
it on `PATH`), use bare `peeragent` instead so merged fixes reach that build
without a plugin update. If a bare `peeragent` call fails with
`command not found`, retry once with the bundled plugin path before reporting
failure.

For invocations that launch or control a peer CLI (`--agent ...`, `--async`, or
`--cancel`), run the wrapper with the host harness's outside-sandbox command
mode so it inherits the developer machine's normal network, auth, hook, and
process environment. In Codex, use the command tool's escalated/outside-sandbox
execution mode. Pure local `--status` and `--result` reads may stay sandboxed
when the job directory is readable. Do not pass peeragent's `--full-access`
unless the user explicitly asked for that target permission mode.

Prefer an asynchronous delegation for substantive work:

```bash
<resolved-peeragent-bin> --async --agent <codex|claude|gemini|zai> [--model <name>] "$ARGUMENTS"
```

`--agent` defaults to `codex` if omitted. Read the launch JSON, retain its
top-level `job_id` (also `metadata.job_id`) and `metadata.cwd`, and follow the
async lifecycle below. Use a blocking call only for a genuinely short pass or
when the host cannot reliably complete the async lifecycle.

## Picking The Target

The right peer is the agent you are not, unless the user named one.

- **If you are Claude Code** → default to
  `--agent codex --model luna --effort high`. Use `--agent gemini` or
  `--agent zai` when the user asks for that model, when Codex isn't available,
  or when extra model diversity is useful.
- **If you are Codex** → default to `--agent claude`. Use `--agent gemini` or
  `--agent zai` when the user asks for that model, when Claude isn't available,
  or when extra model diversity is useful.
- **If you are Antigravity (`agy` / Gemini)** → default to
  `--agent codex --model luna --effort high` or `--agent claude`. Use
  `--agent gemini` when the user asks for another Gemini model/session or when
  Codex and Claude aren't available.
- **If the user named the target** ("ask Codex…", "have Gemini look at…",
  "ask Claude…", "ask GLM…", "use Z.AI…"), use that one.

Different blind spots are the point. Their misses are your catches and vice
versa.

## Effort And Model

### Codex

The `sol` alias selects `gpt-6.2-sol`; `luna` selects `gpt-6-luna`; `terra`
selects `gpt-6.1-terra`; `astra` selects `gpt-6.2-astra`. Explicit
`gpt-6.1-sol`, `gpt-6-<astra|sol>`, and `gpt-5.6-<luna|terra|sol>` IDs remain
supported.

| Desired pass | Codex recommendation | Claude equivalent |
| --- | --- | --- |
| Routine / fast | `--model luna --effort high` | Sonnet-class |
| Lots of work | `--model luna --effort xhigh` | Strong general pass |
| Optional bridge | `--model terra --effort high` or `xhigh` | Between general and flagship |
| Opus-tier | `--model sol --effort low` or `medium` | `--model opus --effort xhigh` |
| Fable-tier | `--model sol --effort high` or `xhigh` | `--model fable --effort high` or `xhigh` |
| GPT-6.2 flagship | `--model astra --effort high` or `xhigh` | Above Fable-tier |

Codex defaults to `xhigh` for Luna and Terra, and `high` for Sol, Astra, and
the native default model. Explicit `--effort` overrides the default; resumes
retain the previous job's effort unless overridden.

### Claude

Claude supports `fable`, `sonnet`, `opus`, and `haiku`. Claude defaults to
`xhigh` and exposes only `high|xhigh` (`medium` and `low` are rejected).

### Gemini (`agy`)

Gemini defaults to `gemini-4-neon` at `high` effort and accepts only two models:

- `neon` (`gemini-4-neon`): `low|medium|high|xhigh` (default `high`)
- `carbon` (`gemini-4-carbon`): `low|medium|high` (default `high`)

No other Gemini models are accepted. `agy` runs with `--output-format
stream-json`, so session IDs and live progress record immediately while running
and survive cancellation just like Codex.

### Z.AI (`pi`)

Z.AI maps `--effort medium|high|xhigh` to Pi `--thinking` and accepts only
`--model glm-5.2`. See
[references/lifecycle-and-options.md](./references/lifecycle-and-options.md) for
the Z.AI configuration smoke test.

## Delegation Contract

The host delegates task intent; the peer performs the focused pass. The task
text is arbitrary natural language, not shell syntax. Don't split one request
into many wrapper calls unless the user explicitly asked for separate passes.

Target agents start without inherited terminal-control addresses by default.
Pass `--keep-terminal-env` only when intentionally delegating work that needs
the launching terminal's control environment (see `docs/TERMINAL_ENV.md`).

Include a no-recursion instruction in every handoff. Tell the peer explicitly
**not** to reach for peeragent's `peer` skill — and not to run the `peeragent`
wrapper — to delegate the work back out to another local coding agent. The peer
may freely use its own harness's internal sub-agents; only recursive peeragent
delegation is prohibited. Example:
`"Do this work directly — do not call the peeragent peer skill to hand it off again; use your own harness's sub-agents if you need help."`

## Job Messages

For a running job, send a reply with:

```bash
<resolved-peeragent-bin> tell <job-id> [--cwd DIR] '<text>'
<resolved-peeragent-bin> tell <job-id> --prompt-file <path>
```

Finished jobs require `--resume`. Use single quotes or `--prompt-file` so
backticks are not expanded by the shell.

Inside a job, `peeragent inbox` reads and acknowledges replies, and every
command announces unread replies on stderr. If a tool display truncates inbox
output, reread with `peeragent messages "$PEERAGENT_JOB_ID" --json`.

A job can use `peeragent say "<text>"` (or `--prompt-file <path>`) only to ask
a question it cannot settle itself — never for status updates. Limits: 2,000
characters, 3 messages per 5 minutes. It uses a recorded Claude Code native
inbox or parent peer job mailbox and never writes to terminal input; without a
native channel it queues the question for `--result`.

See [references/lifecycle-and-options.md](./references/lifecycle-and-options.md)
for reservation rules, `oldest_unread_age`, and legacy job inference details.

## Async Lifecycle

Prefer `--async` for substantive work. For async Codex work, add
`--retry-capacity N` to opt into recovery from capacity, connection,
output-limit, and login stops (default `10m` capacity backoff; override with
`--capacity-backoff <duration>`). Retry options require `--async --agent codex`
even with `--resume`:

```bash
<resolved-peeragent-bin> --async --agent codex --retry-capacity 12 --prompt-file <brief>
```

After launch returns a `job_id`, start peeragent's attached waiter:

```bash
<resolved-peeragent-bin> --wait <job-id> [--cwd <same-cwd>]
```

For several outstanding jobs, attach one waiter to
`--wait-any <job-id> <other-job-id>`. It validates all IDs first, prints the
first terminal result (ties in argument order), and exits `0` on terminal
outcomes; rearm with the remaining IDs.

**Use native harness monitors / background wake-ups for `--wait`:**

- **Antigravity (`agy`)**: Run `--wait <job-id>` via `run_command` with a short
  wait (`WaitMsBeforeAsync: 500`) so it backgrounds immediately, then end your
  turn and rely on the system's reactive completion wakeup. Do not poll or
  sleep.
- **Claude Code / Codex**: Run `--wait <job-id>` through the harness's native
  background task or monitor facility and wait for its completion notification.
- **No native monitor**: Run `--wait` in the foreground when tool timeouts
  suffice, or check `--status <job-id>` at reasonable checkpoints and call
  `--result` once terminal. Never tight-poll.

Other controls:

- `--jobs [--cwd DIR] [--all] [--json | --text]` — list jobs newest first.
- `--status <job-id>...` — one job returns the full status object; several
  return JSONL or a `--text` table with live `last_event_at` and message
  preview (`agent.log` streams live events for both Codex and Gemini/`agy`).
- `--result <job-id>` — fetch current or final result.
- `--cancel <job-id>` — best-effort cancel.
- `--version` — print running binary version.

Job IDs resolve from any directory through the per-user index; `--cwd` is an
explicit override, and multi-job controls accept `job-id@directory`. Read the
complete handback from top-level `final_message` (`null` while running or on
non-success stops); inspect `metadata.log_path` for raw stdout/stderr, or
continue with `--resume <metadata.agent_session>`.

Claude Fable/Opus (`--effort xhigh`) and Gemini 4 Neon (`--effort xhigh`) runs
may take 10 minutes or longer. A quiet run is not by itself evidence of a hang;
keep waiting unless the wrapper exits or fails.

Unknown options and missing/empty option values exit `2` before starting work.
Put `--` before literal task text starting with a dash.

## Result Handling

- `status: success` — report what changed and what verification ran.
- `status: blocked` — explain the blocker; continue from the host side.
- `status: failed` — surface `failure_reason` and useful log details.
  - If exit code `3` (no prebuilt binary for platform), tell the user prebuilt
    binaries cover linux/darwin on amd64/arm64; reinstall or download from
    https://github.com/nklisch/peeragent/releases, or build with
    `go install github.com/nklisch/peeragent/cmd/peeragent@latest` and set
    `PEERAGENT_BIN` (or `PEERAGENT_TARGET_OVERRIDE=<goos>-<goarch>`). Do not
    retry in a loop.
  - With `failure: policy_refusal` (`metadata.stop_reason:
    provider_safety_filter`), surface `metadata.stop_message` and start a fresh
    session for unrelated steps rather than resuming.
- `status: capacity_exhausted` — preserve `metadata.agent_session` for resume
  (or use `--retry-capacity N` for async Codex).
- `status: usage_limited` — surface `summary` and `failure_reason`; never
  auto-retried. Wait for quota reset before resuming manually.
- `status: output_limit` — preserve session and resume command (async Codex
  with `--retry-capacity` gets one automatic continuation).
- `status: auth_required` — refresh target login and resume with
  `metadata.agent_session`.
- `status: interrupted` — preserve session for manual resume; no auto-retry.
- `status: running` — keep monitoring until terminal before concluding.
- `status: cancelled` — report cleanly after `--cancel`.

Do not claim success unless the wrapper reports `success`.

## Options

- `--agent <codex|claude|gemini|zai>` — target CLI (default `codex`).
- `--model <name>` — Codex (`astra|luna|terra|sol` or explicit IDs), Claude
  (`fable|sonnet|opus|haiku`), Gemini (`neon|carbon`, i.e.
  `gemini-4-neon|gemini-4-carbon` only), Z.AI (`glm-5.2`).
- `--effort <low|medium|high|xhigh>` — target reasoning effort.
- `--resume <agent-session>` — continue a prior session (inherits last job's
  model/effort and directory unless overridden; use `--allow-concurrent-session`
  only for intentional overlap).
- `--async`, `--retry-capacity <N>`, `--capacity-backoff <duration>` — async
  execution and Codex capacity recovery.
- `--cwd <path>`, `--prompt-file <path>`, `--profile <name>`, `--full-access`,
  `--keep-terminal-env`, `--no-preamble`, `--worktree` (reserved),
  `--json` / `--text`.

Full option semantics, archive/index retention rules, and mailbox lifecycle
details live in
[references/lifecycle-and-options.md](./references/lifecycle-and-options.md).

## Guardrails

- Keep the handoff concise. Send the task, not a full transcript recap unless
  context matters.
- Do not retry the same peer repeatedly in a loop after `failed` or `blocked`
  results — diagnose first. Never retry exit code `3`.
- Preserve the host's responsibility for explaining outcomes to the user. The
  peer is the worker; you are the narrator.
- Treat the working tree as shared space — the peer may edit files you are also
  editing. Re-read before re-edit.
- No recursive peeragent. Always instruct the peer not to invoke the `peer`
  skill or `peeragent` wrapper; its own internal harness sub-agents are fine.
