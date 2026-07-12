# MCP Typed Tool Handler

A typed MCP handler factory closes over an application-service interface, normalizes boundary input, guards the service seam, and returns the shared structured result.

## Rationale

The official Go SDK derives schemas and validates typed input while `internal/mcp` stays a protocol adapter. Application services own execution and persistence; handlers only map requests and preserve tool-error semantics.

## Examples

- `internal/mcp/tools.go:24` — `delegateHandler` maps `DelegateInput` through canonical delegation normalization before dispatch.
- `internal/mcp/jobs.go:58` — `jobStatusHandler` validates job input and calls `JobStatus`.
- `internal/mcp/jobs.go:71` — `jobResultHandler` uses the same skeleton for result retrieval.
- `internal/mcp/jobs.go:84` — `jobCancelHandler` exposes destructive cancellation through the shared service.

## When to use

Use for a new MCP operation that maps one typed input to one application-service call and a `result.Result` output.

## When not to use

Do not use this exact shape for streaming, multi-step host orchestration, or non-result content.

## Common violations

- Reading job files or signaling processes directly in a protocol handler.
- Reimplementing validation instead of calling the canonical boundary normalizer.
- Populating ad hoc text content when typed structured output is the contract.
