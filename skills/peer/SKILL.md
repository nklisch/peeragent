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

## MCP-first behavior

If the host exposes the bundled `peeragent` MCP server, prefer its `delegate`
tool over shelling out to this skill's wrapper. Pass the task as `task`, select
`agent` when needed, and omit `cwd` unless the user explicitly asks for a
cross-repository checkout. For work likely to exceed the host tool timeout, set
`async: true`, poll `job_status`, and retrieve the completed result with
`job_result`. Ask for explicit user intent before using `full_access: true` or
`job_cancel`; both can change or terminate work.

If the MCP server is unavailable, use the bundled-wrapper procedure below. Do
not assume a globally installed `peeragent` command. Whether the MCP tool or
wrapper is used, never recursively delegate through peeragent's `peer` or
`peer-review` skill, MCP `delegate`, or wrapper from the delegated task. The
peer is the endpoint of this handoff.

## Default Behavior

Do not assume `peeragent` is on `PATH`. Resolve the bundled wrapper before
the first call and use that path for every invocation:

- If `PEERAGENT_BIN` names an executable, use it.
- Otherwise resolve from this skill file: go two directories up from the
  skill directory, then use `bin/peeragent`.
- In a development checkout, the same wrapper is `bin/peeragent`.
- Use bare `peeragent` only if the bundled path cannot be found and
  `command -v peeragent` succeeds.

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

Pass the user's task text to the wrapper:

```bash
<resolved-peeragent-bin> --agent <codex|claude|gemini|zai> [--model <name>] "$ARGUMENTS"
```

`--agent` defaults to `codex` if omitted. The wrapper runs in the current
repository, blocks until the peer finishes, and returns JSON. Read the
JSON before responding to the user.

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

Use GPT-5.6 for all Codex work. Luna is the routine fast path; Sol is the
flagship jump for difficult work. Terra is a useful bridge, but most calls can
jump directly from Luna to Sol.

| Desired pass | Codex recommendation | Claude equivalent |
| --- | --- | --- |
| Routine / fast | `--model luna --effort high` | Sonnet-class |
| Lots of work | `--model luna --effort xhigh` | Strong general pass |
| Optional bridge | `--model terra --effort high` or `xhigh` | Between general and flagship |
| Opus-tier | `--model sol --effort low` or `medium` | `--model opus --effort xhigh` |
| Fable-tier | `--model sol --effort high` or `xhigh` | `--model fable --effort high` or `xhigh` |

Claude also retains Opus, Sonnet, and Haiku aliases. Claude rejects `--effort medium`;
peeragent exposes only `high|xhigh` for Claude. Gemini ignores `--effort` and
`--model` beyond accepting `--model gemini-3.5` as a no-op for explicit
metadata. Z.AI maps `--effort medium|high|xhigh` to Pi `--thinking` and accepts
only `--model glm-5.2`; no other Z.AI model is surfaced.

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

Default is blocking. Wait for the command to return, then summarize the
result using the wrapper's `status`, `summary`, `changed_files`,
`verification`, and `details` fields.

The default `details` payload is compact. For Codex targets, peeragent surfaces
the final completed agent message instead of the full stream of interim assistant
messages. If a deeper look is needed, inspect `metadata.log_path` for raw target
stdout/stderr, or use `metadata.agent_session` with `--resume` for continuity.

Claude Fable and Opus runs, especially with `--effort xhigh`, may take 10
minutes or longer to finish. A slow or quiet flagship reply is not by itself
evidence of a hung process; keep waiting unless the wrapper exits, reports
failure, or there is concrete evidence the process is stuck.

Include a no-recursion instruction in every handoff. Tell the peer
explicitly **not** to reach for peeragent's own `peer` or `peer-review`
skills — and not to run the `peeragent` wrapper — to delegate the work back
out to another local coding agent. That would loop host → peer → host and
burn budget. The peer is the endpoint of this delegation. It may freely use
its own harness's internal sub-agents (Codex/Claude/Gemini built-in task or
sub-agent tooling, or Pi-native extensions if available) for parallel or
focused passes; the prohibition is only
on recursive peeragent calls. A short line in the prompt is enough, e.g.
"Do this work directly — do not call the peeragent peer/peer-review skills
to hand it off again; use your own harness's sub-agents if you need help."

## Result Handling

- `status: success` — report what changed and what verification ran.
- `status: blocked` — explain the blocker; continue from the host side.
- `status: failed` — surface the failure reason and useful log details.
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
- `status: running` — only with `--async`; report the job id and how to
  check it.
- `status: cancelled` — only after `--cancel`; report cleanly.

Do not claim success unless the wrapper reports `success`.

## Options

Use advanced modes only when the request calls for them:

- `--full-access` — run the peer CLI without sandboxing. Never implied;
  ask the user first if the wrapper reports full access is needed.
- `--worktree` — reserved; returns a clear failure today.
- `--profile <name>` — Codex profile override.
- `--model <name>` — Codex GPT-5.6 aliases (`luna`, `terra`, `sol`) or their
  canonical `gpt-5.6-*` IDs; Claude aliases (`fable`, `sonnet`, `opus`,
  `haiku`); explicit Gemini `gemini-3.5`; or explicit Z.AI `glm-5.2`. No other
  Z.AI models are accepted.
- `--resume <agent-session>` — continue a prior target-agent session when
  the previous result included `metadata.agent_session`. Use it for continuity
  inside one multi-pass workflow; omit it for an independent second opinion.
- `--cwd <path>` — repo directory the peer runs in.
- `--prompt-file <path>` — read large prompts from a file.
- `--async` — start the peer as a background job; the wrapper returns a
  `job_id` immediately with `status: running`.
- `--status <job-id>` — check an async job.
- `--result <job-id>` — fetch a finished async job's final result.
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
- No recursive peeragent. The handoff prompt must tell the peer not to
  call peeragent's `peer`/`peer-review` skills (or the `peeragent` wrapper)
  back out to another agent. The peer's own internal harness sub-agents are
  fine; recursive peeragent delegation is not.
- For review work, prefer `/peer-review` (the looping cross-model review
  skill) over a one-shot `peer` call.
