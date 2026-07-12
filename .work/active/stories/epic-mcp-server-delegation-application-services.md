---
id: epic-mcp-server-delegation-application-services
kind: story
stage: implementing
tags: [infra]
parent: epic-mcp-server-delegation
depends_on: []
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# Extract delegation application services

## Scope

Create the canonical delegation request normalizer and extract blocking execution plus async launch from `cmd/peeragent` into an `internal/app` service. Keep the CLI as an adapter that formats results and chooses exit codes; application code returns values and never writes stdout or exits the process.

## Acceptance criteria

- [ ] CLI parsing and request validation use `input.NormalizeDelegation`.
- [ ] Blocking execution and async launch are callable through injected application ports.
- [ ] Existing CLI, target routing, result, launch-cleanup, and async tests remain green.
- [ ] Application tests cover successful, target-failed, and infrastructure-failed paths without local agent CLIs.
