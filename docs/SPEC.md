# Specification

## Name

The skill and bundled CLI are both named `codex-implement`.

When packaged as a Claude Code plugin, the installed skill is invoked through Claude Code's plugin namespace, for example:

```text
/plugin-name:codex-implement <task text>
```

When installed as a standalone or project skill, the user-facing command name is:

```text
/codex-implement <task text>
```

## Runtime Context

Codex Implement runs on a developer machine with:

- Claude Code installed with plugin and skill support.
- Codex CLI installed and authenticated.
- A repository or working directory open in Claude Code.
- A platform-compatible `codex-implement` wrapper binary supplied by the plugin.

The plugin uses the local Codex installation and local Codex authentication state. It does not bundle a separate Codex runtime or account system.

## Components

The project contains:

- `.claude-plugin/plugin.json` for Claude Code plugin metadata.
- `skills/codex-implement/SKILL.md` for Claude's delegation instructions.
- `bin/codex-implement` as the executable Claude calls.
- `cmd/codex-implement/` and internal Go packages for the wrapper implementation.
- `docs/` for foundation documents.

## Invocation Modes

### Blocking

Blocking invocation is the default. Claude calls the wrapper and waits until Codex completes. This matches Claude Code's ordinary tool-use flow and keeps the handoff simple.

### Async

Async invocation is explicit. The wrapper starts a tracked Codex job, records logs and status locally, and returns a task id. Claude can later inspect the task through the wrapper.

Async mode is for long-running work. It is not the default architecture.

### Full Access

Full access is explicit. The caller must request it with a flag or wrapper option. Full access is appropriate only when the user intentionally trusts the repository and machine context.

## Codex Execution Defaults

The default Codex execution uses the same checkout and working tree as Claude. It does not create a git worktree or sandboxed copy.

The default Codex permissions are classifier-compatible:

- Codex can work in the current repository.
- Routine in-workspace edits and commands can proceed.
- Boundary-crossing requests remain reviewable by Codex Auto-review when configured.
- The wrapper does not default to `approval_policy = "never"` or unchecked full access.

This preserves the practical value of automatic approval review while still letting Codex edit the raw repository.

## Codex CLI Strategy

The first implementation path uses `codex exec` because it is simple, scriptable, and suitable for blocking calls.

The wrapper may use these Codex capabilities:

- `codex exec` for non-interactive execution.
- `--cd` to set the repository root.
- `--sandbox` or permission-profile options to control Codex permissions.
- `--ask-for-approval` or config overrides for approval behavior.
- `--output-last-message` to capture the final response.
- `--output-schema` when structured final output is reliable enough for the wrapper.
- `--json` for machine-readable event streams when useful.

Codex app-server is an extension point for richer progress events, resumable threads, and job management. It is not required for the default implementation.

## Output Requirements

The wrapper returns a concise result to Claude. The result includes:

- Overall status.
- Human-readable summary.
- Changed files when known.
- Verification commands and outcomes when known.
- Codex final message.
- Failure reason and useful log excerpts when Codex fails.
- Session or job identifiers when available.

The wrapper avoids dumping long raw logs into Claude's context unless the task failed and the logs are needed to continue.

## Safety Boundaries

The project treats same-checkout execution and no-sandbox execution as different decisions.

Same-checkout execution is the default. No-sandbox or full-access execution is not.

The wrapper keeps the user in control by making high-risk modes explicit, preserving classifier-compatible defaults, and reporting what Codex did. It does not pretend that classifier review is a security guarantee.

## Non-Goals

The project does not provide:

- A full Codex job dashboard.
- A replacement for Codex CLI.
- A replacement for Claude Code permissions.
- A required worktree workflow.
- A multi-agent planning framework.
- A second-opinion review system as the primary feature.
