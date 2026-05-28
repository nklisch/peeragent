---
name: peer
description: >
  Delegate arbitrary task work — implementation, research, review, debugging,
  design, doc updates, anything — to another local coding agent through the
  bundled peeragent wrapper. Use when the host assistant wants a peer agent
  to take a focused pass on a task in the current repository. Targets: Codex
  (default), Claude Code, Gemini through Antigravity. Free-form task text;
  the wrapper returns a JSON result the host summarizes for the user.
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
<resolved-peeragent-bin> --agent <codex|claude|gemini> "$ARGUMENTS"
```

`--agent` defaults to `codex` if omitted. The wrapper runs in the current
repository, blocks until the peer finishes, and returns JSON. Read the
JSON before responding to the user.

## Picking The Target

The right peer is the agent you are not, unless the user named one.

- **If you are Claude Code** → default to `--agent codex`. Use
  `--agent gemini` when the user asks for Gemini or when Codex isn't
  available.
- **If you are Codex** → default to `--agent claude`. Use
  `--agent gemini` when the user asks for Gemini or when Claude isn't
  available.
- **If the user named the target** ("ask Codex…", "have Gemini look
  at…"), use that one.

Different blind spots are the point. Their misses are your catches and
vice versa.

## Effort And Model

Match depth to the task. Routine work takes the defaults; bump up only
when the work is dense or the stakes are high.

| Target | Default | Lightweight | Deeper pass |
| --- | --- | --- | --- |
| `--agent codex` | `--effort high` | `--effort medium` | `--effort xhigh` |
| `--agent claude` | `--model sonnet --effort xhigh` | `--model haiku --effort high` | `--model opus --effort xhigh` |
| `--agent gemini` | fixed Gemini 3.5 (no flag needed) | — | — |

Claude rejects `--effort medium`. Gemini ignores `--effort` and `--model`
beyond accepting `--model gemini-3.5` as a no-op for explicit metadata.

## Delegation Contract

The host delegates task intent; the peer performs the focused pass. The
task text is arbitrary natural language, not shell syntax. Don't split
the request into many wrapper calls unless the user explicitly asked for
separate passes.

Default is blocking. Wait for the command to return, then summarize the
result using the wrapper's `status`, `summary`, `changed_files`,
`verification`, and `details` fields.

## Result Handling

- `status: success` — report what changed and what verification ran.
- `status: blocked` — explain the blocker; continue from the host side.
- `status: failed` — surface the failure reason and useful log details.
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
  `blocked` results — diagnose first.
- Preserve the host's responsibility for explaining outcomes to the
  user. The peer is the worker; you are the narrator.
- Treat the working tree as shared space — the peer may edit files you
  are also editing. Re-read before re-edit.
- For review work, prefer `/peer-review` (the looping cross-model review
  skill) over a one-shot `peer` call.
