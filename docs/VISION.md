# Vision

Alt Subagent gives coding assistants a low-friction way to delegate
implementation, research, or review work to another local agent or model
harness without leaving the current repository workflow.

The host assistant remains the primary collaborator in the session. When a task
benefits from a different autonomous coding agent or model family, the host
invokes an Alt Subagent skill, passes arbitrary task text, waits by default, and
resumes with a compact result. The target agent works in the same checkout and
may edit files directly according to its local permission model.

## Primary Users

The primary user is a developer who works in Claude Code or Codex and also has
one or more alternate local agent CLIs installed:

- OpenAI Codex CLI
- Google Antigravity CLI (`agy`) for Gemini-backed agents
- Claude Code CLI
- Pi CLI configured for Z.AI GLM 5.2

Claude and Codex are also users of this project. Skills give each host a clear
delegation contract: when to call another agent, what to send, how to interpret
results, and how to continue after the target agent returns. MCP-capable coding
hosts can use the same delegation contract through peeragent's local stdio MCP
server without requiring a host-specific skill.

## Problem

Different coding agents have different strengths. Switching between them
manually interrupts flow, loses context, and forces the human to act as the
integration layer. Existing bridge patterns often expose too many explicit
commands, job-management surfaces, or separate review flows. That makes
delegation feel like operating another tool rather than asking another agent to
take a pass.

Alt Subagent solves the narrow problem of focused task delegation. It avoids
becoming a general multi-agent control panel.

## Product Shape

Alt Subagent is a plugin-ready repository containing:

- A bundled `peeragent` CLI wrapper.
- A local stdio MCP server exposing delegation and async job controls.
- A `/peer` skill for focused delegation.
- A `/peer-review` skill for iterative cross-model review.
- Shared request and result contracts across CLI, MCP, and host skills.

The wrapper invokes the selected local CLI with predictable defaults, captures
the outcome, and returns a concise result to the host.

## Principles

- Same checkout by default.
- Explicit target agent selection.
- Direct task execution, not patch-only output.
- Safe defaults before full access.
- Blocking first, async as an explicit mode.
- Compact machine-readable results.
- Host assistant remains responsible for user communication.

## Success

Delegation feels natural when the host can say what needs implementing,
researching, or reviewing, wait for another local agent to work, and then
continue from a clear result. The user should not need to manually copy prompts
into another terminal, monitor two
agents, or reconcile ambiguous outputs.

Concrete success criteria:

- Claude and Codex can invoke `/peer` or `/peer-review`.
- MCP-capable hosts can discover and invoke peeragent delegation and async job
  tools through a local stdio server.
- The wrapper can target `codex`, `gemini`, `claude`, or `zai`.
- The `zai` target uses Pi with only Z.AI `glm-5.2` surfaced.
- Target agents run in the current repository by default.
- Target agents can make direct edits and run relevant verification commands.
- Results include status, summary, verification, changed files when known,
  details, and metadata.

## Non-Goals

Alt Subagent is not a dashboard, a general multi-agent orchestrator, a remote
MCP service, an arbitrary MCP proxy, a mandatory worktree manager, or a
patch-only generator. It does not replace native interfaces, host-agent
permissions, or human review for high-risk changes.
