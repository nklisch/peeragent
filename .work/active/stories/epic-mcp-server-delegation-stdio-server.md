---
id: epic-mcp-server-delegation-stdio-server
kind: story
stage: drafting
tags: [infra]
parent: epic-mcp-server-delegation
depends_on: [epic-mcp-server-delegation-application-services]
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# Add stdio MCP delegate tool

## Scope

Raise the Go baseline to 1.23, add the official MCP Go SDK v1.6.1 in this story (and no earlier story), route `peeragent mcp` to a stdio server, and expose the typed `delegate` tool over the shared application service. The tool supports blocking and async calls and returns the existing result contract as structured output.

## Acceptance criteria

- [ ] MCP initialization, instructions, list-tools, generated schemas, and delegate calls work through an in-memory client/server test.
- [ ] Blocking and async results preserve the peeragent result schema.
- [ ] Invalid delegation input is rejected before execution.
- [ ] Context cancellation reaches blocking target execution.
- [ ] A subprocess smoke test proves protocol stdout purity.
- [ ] CI workflows, build scripts, and installation docs are audited for independent Go 1.22 assumptions; the supported minimum is stated as Go 1.23.
- [ ] Existing CLI behavior remains green under Go 1.23.

## Implementation discovery
The fixed SDK choice and the Go 1.23 minimum are contradictory in the resolved dependency. `github.com/modelcontextprotocol/go-sdk@v1.6.1/go.mod` declares `go 1.25.0`; verified locally with `go list -m all` under `GOTOOLCHAIN=go1.23.0`, which fails before compilation with `module ... requires go >= 1.25.0`. Proceeding would either falsely advertise Go 1.23 support or violate the feature's fixed SDK version. The application-services story is complete at review, but this story is returned to drafting for the design owner to resolve the SDK/baseline choice. No MCP code or dependency changes were retained.
