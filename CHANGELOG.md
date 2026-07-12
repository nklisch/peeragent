# Changelog

## v0.5.0

### Features
- Add a stdio MCP server with typed `delegate`, `job_status`, `job_result`, and `job_cancel` tools backed by shared application services.
- Bundle MCP server configuration and guidance for Claude Code and Codex plugins.
- Add GPT-5.6 Luna, Terra, and Sol model aliases for Codex and Fable model support for Claude.
- Ship the first-class Pi package alongside Claude Code and Codex marketplace packaging.
- Harden asynchronous jobs with durable prompt/PID state, guarded terminal transitions, process-group cancellation, and robust child cleanup.

### Security
- Reject malformed and path-traversing async job IDs at input and storage boundaries.

### Fixes
- Prevent inherited Go test-helper state from launching peeragent recursively.
- Avoid Gemini sandbox false-positive authentication and timeout failures.
- Preserve cleanup when detached-process PID persistence or process release fails.

### Refactor
- Centralize agent identities, aliases, prompt names, and display names in one typed registry.
- Centralize persisted lifecycle statuses and terminal-state membership.
- Share target-adapter runner test support without weakening target-specific assertions.

### Documentation
- Document MCP setup, Pi packaging, supported models, async job controls, and the current repository layout.
