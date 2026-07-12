---
name: patterns
description: "Project code patterns and conventions. Auto-loads when implementing, designing, verifying, or reviewing code. Provides detailed pattern definitions with code examples."
user-invocable: false
allowed-tools: Read, Glob, Grep
---

# Project Patterns Reference

This skill contains detailed pattern documentation for this project. See individual pattern files for full rationale, examples, and common violations.

Available patterns:
- [target-executor-adapter.md](target-executor-adapter.md) — Keep each agent CLI behind one adapter package exposing Exec/ExecWithRunner/buildArgs over the shared runner/result port.
- [mcp-typed-tool-handler.md](mcp-typed-tool-handler.md) — Normalize typed MCP input, guard the application service, call one operation, and return the shared structured result.
- [job-state-fs-error-categorization.md](job-state-fs-error-categorization.md) — Translate missing job files into call-site domain states and wrap every other filesystem error.
- [runner-test-double-per-package.md](runner-test-double-per-package.md) — Drive each target adapter through an offline recording runner and cleanup-restored lookPath seam.
- [lifecycle-status-dictionary.md](lifecycle-status-dictionary.md) — Map persisted and wire lifecycle states only through named dictionary and terminal-membership functions.
