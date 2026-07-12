---
id: epic-mcp-server-job-control-tools
kind: story
stage: implementing
tags: [infra]
parent: epic-mcp-server-job-control
depends_on: [epic-mcp-server-job-control-application-services]
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# Add MCP async job tools

## Scope

Register typed `job_status`, `job_result`, and `job_cancel` tools over the shared application service. Include generated schemas, read/destructive annotations, structured peeragent results, and server instructions for the complete async workflow.

## Acceptance criteria

- [ ] Tool discovery exposes delegate plus all three job tools with correct schemas.
- [ ] Read-only annotations distinguish status/result from destructive cancellation.
- [ ] Running, terminal, missing, corrupt, and cancellation-race outcomes map correctly.
- [ ] Repository cwd defaults and explicit overrides use the same normalizer.
- [ ] Concurrent MCP calls preserve persisted terminal-state invariants.
