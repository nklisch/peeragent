# Vision

Alt Subagent gives coding assistants a low-friction way to delegate
implementation work to another local agent without leaving the current
repository workflow.

The host assistant remains the primary collaborator in the session. When an
implementation task benefits from a different autonomous coding agent, the host
invokes an Alt Subagent skill, passes arbitrary task text, waits by default, and
resumes with a compact result. The target agent works in the same checkout and
may edit files directly according to its local permission model.

## Primary Users

The primary user is a developer who works in Claude Code or Codex and also has
one or more alternate local agent CLIs installed:

- OpenAI Codex CLI
- Google Antigravity CLI (`agy`) for Gemini-backed agents
- Claude Code CLI

Claude and Codex are also users of this project. Skills give each host a clear
delegation contract: when to call another agent, what to send, how to interpret
results, and how to continue after the target agent returns.

## Problem

Different coding agents have different strengths. Switching between them
manually interrupts flow, loses context, and forces the human to act as the
integration layer. Existing bridge patterns often expose too many explicit
commands, job-management surfaces, or separate review flows. That makes
delegation feel like operating another tool rather than asking another
implementor to take a pass.

Alt Subagent solves the narrow problem of implementation delegation. It avoids
becoming a general multi-agent control panel.

## Product Shape

Alt Subagent is a plugin-ready repository containing:

- A bundled `alt-subagent` CLI wrapper.
- Claude-facing skills for delegating to Codex and Gemini.
- Codex-facing skills for delegating to Claude and Gemini.
- A shared JSON result contract for host agents.

The wrapper invokes the selected local CLI with predictable defaults, captures
the outcome, and returns a concise result to the host.

## Principles

- Same checkout by default.
- Explicit target agent selection.
- Direct implementation, not patch-only output.
- Safe defaults before full access.
- Blocking first, async as an explicit mode.
- Compact machine-readable results.
- Host assistant remains responsible for user communication.

## Success

Delegation feels natural when the host can say what needs implementing, wait for
another local agent to work, and then continue from a clear result. The user
should not need to manually copy prompts into another terminal, monitor two
agents, or reconcile ambiguous outputs.

Concrete success criteria:

- Claude can invoke `codex-implement` or `gemini-implement` skills.
- Codex can invoke `claude-implement` or `gemini-implement` skills.
- The wrapper can target `codex`, `gemini`, or `claude`.
- Target agents run in the current repository by default.
- Target agents can make direct edits and run relevant verification commands.
- Results include status, summary, verification, changed files when known,
  details, and metadata.

## Non-Goals

Alt Subagent is not a dashboard, a general multi-agent orchestrator, a mandatory
worktree manager, or a patch-only generator. It does not replace native
interfaces, host-agent permissions, or human review for high-risk changes.
