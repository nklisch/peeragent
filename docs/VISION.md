# Vision

Alt Subagent gives coding assistants a low-friction way to delegate
implementation, research, or review work to another local agent or model
harness without leaving the current repository workflow.

The host assistant remains the primary collaborator in the session. When a task
benefits from a different autonomous coding agent or model family, the host
invokes the `peer` skill, passes arbitrary task text, and resumes with a compact
result. Substantive work launches asynchronously and uses native host completion
facilities to observe the attached `--wait` process when available; short
passes may remain blocking. The target agent works in the same checkout and
may edit files directly according to its local permission model.

## Primary Users

The primary user is a developer who works in Claude Code or Codex and also has
one or more alternate local agent CLIs installed:

- OpenAI Codex CLI
- Google Antigravity CLI (`agy`) for Gemini-backed agents
- Claude Code CLI
- Pi CLI configured for Z.AI GLM 5.2

Claude, Codex, and Pi are also users of this project. Skills give each host a
clear delegation contract: when to call another agent, what to send, how to
interpret results, and how to continue after the target agent returns.

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
- A `/peer` skill for focused delegation, including implementation, research,
  and review tasks.
- A shared request and result contract across the CLI and host skill.

The wrapper invokes the selected local CLI with predictable defaults, captures
the outcome, and returns a concise result to the host.

## Principles

- Same checkout by default.
- Explicit target agent selection.
- Direct task execution, not patch-only output.
- Safe defaults before full access.
- Async delegation for substantive work; blocking for short passes.
- Compact machine-readable results.
- Host assistant remains responsible for user communication.

## Success

Delegation feels natural when the host can say what needs implementing,
researching, or reviewing, wait for another local agent to work, and then
continue from a clear result. The user should not need to manually copy prompts
into another terminal, monitor two
agents, or reconcile ambiguous outputs.

Concrete success criteria:

- Claude, Codex, and Pi can invoke the `peer` skill.
- The wrapper can target `codex`, `gemini`, `claude`, or `zai`.
- The `zai` target uses Pi with only Z.AI `glm-5.2` surfaced.
- Target agents run in the current repository by default.
- Target agents can make direct edits and run relevant verification commands.
- Results include status, summary, verification, changed files when known,
  details, and metadata.

## Non-Goals

Alt Subagent is not a dashboard, a general multi-agent orchestrator, an MCP
server, a mandatory worktree manager, or a patch-only generator. MCP is not a
fit for the completion boundary because detached tools have no portable wake-up
mechanism and blocking tools compete with host timeouts. It does not replace
native interfaces, host-agent permissions, or human review for high-risk
changes.
