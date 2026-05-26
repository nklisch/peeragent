# Vision

## Purpose

Codex Implement gives Claude Code a low-friction way to delegate implementation work to OpenAI Codex without leaving the current repository workflow.

Claude remains the primary collaborator in the session. When an implementation task benefits from an independent autonomous coding agent, Claude calls the `codex-implement` skill, passes arbitrary task text, waits for Codex by default, and resumes with a compact result. Codex works in the same checkout Claude is already using and may edit files directly.

## Users

The primary user is a developer working in Claude Code who also has Codex CLI installed and authenticated locally. The user wants Claude to stay in control of the conversation while Codex performs focused implementation passes in the background of the same development environment.

Claude is also a user of this project. The skill gives Claude a clear delegation contract: when to call Codex, what to send, how to interpret results, and how to continue after Codex returns.

## Problem

Claude Code and Codex each have different strengths. Switching between them manually interrupts flow, loses context, and forces the human to act as the integration layer. Existing bridge patterns often expose too many explicit commands, job-management surfaces, or separate review flows. That makes delegation feel like operating another tool rather than asking another implementor to take a pass.

Codex Implement solves the narrow problem of implementation delegation. It avoids becoming a general Codex control panel.

## Product Definition

Codex Implement is a Claude Code plugin containing a `codex-implement` skill and a bundled `codex-implement` CLI wrapper. Claude invokes the skill with arbitrary implementation text. The skill runs the wrapper. The wrapper invokes Codex CLI with predictable defaults, captures the outcome, and returns a concise result to Claude.

The default execution mode is:

- Same repository checkout.
- Same working tree.
- Blocking call.
- Codex may edit files directly.
- No worktree or isolation unless explicitly requested.
- Classifier-compatible Codex permissions by default.
- Explicit full-access mode for trusted environments.
- Optional async mode for long-running tasks.

## What Good Looks Like

Delegation feels natural when Claude can say what needs implementing, wait for Codex to work, and then continue from a clear result. The user should not need to manually copy prompts into another terminal, monitor two agents, or reconcile ambiguous outputs.

The first useful version is successful when:

- Claude can invoke `codex-implement` with arbitrary text.
- Codex runs in the current repository by default.
- Codex can make direct edits and run relevant verification commands.
- Classifier-backed safety is preserved by default instead of being bypassed accidentally.
- Claude receives a compact result containing status, summary, changed files, verification, and failure details when applicable.
- Async operation is available without becoming the default mental model.

## Non-Goals

Codex Implement is not a Codex dashboard, a general multi-agent orchestrator, a mandatory worktree manager, or a patch-only generator. It does not replace Claude Code subagents, Claude Code hooks, or Codex's native interfaces.

The project does not hide risk by silently granting full access. Full access is an explicit caller decision.

