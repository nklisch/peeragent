---
id: epic-mcp-server-job-control
kind: feature
stage: implementing
tags: [infra]
parent: epic-mcp-server
depends_on: [epic-mcp-server-delegation]
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# MCP async job control

## Brief

Expose peeragent's existing async job lifecycle through MCP: status inspection, result retrieval, and cancellation. The tools must preserve terminal-state race handling, job lookup semantics, repository scoping, and the shared result schema rather than reimplementing filesystem or process behavior inside protocol handlers.

This feature extends the server and application boundary established by `epic-mcp-server-delegation`. It does not add a dashboard, job listing, remote transport, or review orchestration.

## Epic context
- Parent epic: `epic-mcp-server`
- Position in epic: consumer of the shared application and MCP server foundation

## Foundation references
- `docs/SPEC.md` — async lifecycle and MCP tool surface
- `docs/ARCHITECTURE.md` — async flow, cancellation, and MCP adapter role

## Design decisions

- **Tool granularity**: Expose `job_status`, `job_result`, and `job_cancel` separately. Each operation has distinct read/write annotations and host approval behavior; one overloaded job tool would obscure cancellation risk.
- **Lookup scope**: Every tool accepts `job_id` plus optional `cwd`, defaulting to the MCP server working directory. Job IDs remain repository-local and are never searched globally.
- **Result semantics**: Missing jobs return the existing structured failed result with metadata exit code 4. A job without `result.json` returns `status: running`. Corrupt persisted state is a tool/infrastructure error because no truthful domain result can be reconstructed.
- **Cancellation**: Preserve the existing lock, result-first terminal transition, process-group TERM/KILL sequence, and completed-vs-cancelled race winner exactly. MCP request cancellation must not abort cleanup after cancellation has committed terminal state.

## Architectural choice

Move job status/result/cancel behavior into the existing `internal/app.Service` and make both CLI and MCP thin adapters. The application service continues to use `jobs.Store` and platform signal helpers as outbound infrastructure. MCP handlers only map typed inputs, call one service method, and return structured results.

Alternatives rejected:

1. **Handlers read `.peeragent/jobs` directly** — small but creates a second implementation that will drift on terminal races and lookup behavior.
2. **Handlers invoke CLI job flags** — preserves behavior through a subprocess but loses structured errors, adds output parsing, and risks recursive executable resolution.
3. **One generic `job` tool** — fewer registrations but poorer schemas and unsafe approval semantics for cancellation.

## Implementation units

### Unit 1: Shared job query services
**Files**: `internal/app/jobs.go`, `internal/app/jobs_test.go`, `cmd/peeragent/main.go`
**Story**: `epic-mcp-server-job-control-application-services`

```go
type JobRequest struct {
    CWD   string
    JobID string
}

func (s *Service) JobStatus(context.Context, JobRequest) (result.Result, error)
func (s *Service) JobResult(context.Context, JobRequest) (result.Result, error)
```

Normalize cwd and require a non-empty job id before opening the repository-local store. `JobStatus` maps persisted job states through the existing result status mapping. `JobResult` returns running when the result file is absent and decodes the terminal shared result otherwise. Missing job state produces the existing exit-code-4 failed result.

**Acceptance criteria**:
- [ ] CLI `--status` and `--result` output and exit behavior stay compatible.
- [ ] Running, complete, failed, cancelled, missing, and corrupt state paths are tested.
- [ ] MCP and CLI consume the same returned `result.Result` rather than reading files independently.

### Unit 2: Shared cancellation service
**Files**: `internal/app/cancel.go`, `internal/app/cancel_test.go`, `cmd/peeragent/main.go`
**Story**: `epic-mcp-server-job-control-application-services`

```go
type ProcessController interface {
    TerminateGroup(pid int) error
    KillGroup(pid int) error
    GroupExists(pid int) bool
}

func (s *Service) CancelJob(context.Context, JobRequest) (result.Result, error)
```

Move cancellation orchestration behind an injectable process controller and clock/wait policy. Once the locked cancelled result is persisted, finish PID cleanup with a bounded internal context even if the caller disconnects, so the child is not stranded. Preserve terminal result conflict detection and idempotent repeated cancellation.

**Acceptance criteria**:
- [ ] Completion wins if its terminal result exists before cancellation commits.
- [ ] Cancellation wins atomically when no competing terminal result exists.
- [ ] TERM escalates to KILL after the existing grace period and PID state is removed.
- [ ] Repeated cancellation is idempotent and never signals an already-complete process.
- [ ] Tests use fake process control and do not signal real process groups.

### Unit 3: MCP job tools
**Files**: `internal/mcp/jobs.go`, `internal/mcp/server.go`, `internal/mcp/server_test.go`
**Story**: `epic-mcp-server-job-control-tools`

```go
type JobService interface {
    JobStatus(context.Context, app.JobRequest) (result.Result, error)
    JobResult(context.Context, app.JobRequest) (result.Result, error)
    CancelJob(context.Context, app.JobRequest) (result.Result, error)
}

type JobInput struct {
    JobID string `json:"job_id" jsonschema:"required peeragent async job id"`
    CWD   string `json:"cwd,omitempty" jsonschema:"repository directory; defaults to the server working directory"`
}
```

Register three typed tools. Mark `job_status` and `job_result` read-only/non-destructive; mark `job_cancel` write-capable and destructive. All return `result.Result` as structured content. Update server instructions with the async workflow: `delegate(async=true)` → `job_status` → `job_result`, with `job_cancel` only on explicit intent.

**Acceptance criteria**:
- [ ] Tool discovery lists all four total server tools with correct schemas and annotations.
- [ ] Status and result calls do not mutate persisted state.
- [ ] Cancellation is visibly write/destructive to hosts and returns the shared cancelled or race-winning result.
- [ ] Invalid/missing ids fail before store or process operations.
- [ ] Concurrent MCP calls preserve job-store locking and terminal-state invariants.

## Implementation order

1. `epic-mcp-server-job-control-application-services`
2. `epic-mcp-server-job-control-tools`

## Testing

- Port the existing async CLI tests to assert the shared service first, retaining CLI-level compatibility cases.
- Add table-driven service tests for every persisted status, absent results, missing ids, missing jobs, malformed JSON, repeated cancellation, and competing completion.
- Inject process control and short wait policies to deterministically test TERM/KILL behavior.
- Extend in-memory MCP integration tests to invoke all job tools and inspect tool annotations.
- Run race-enabled package tests for concurrent status/result/cancel calls where CI budget permits.

## Risks

- **Cancellation refactor**: This is the highest-risk unit because terminal files, locks, PIDs, and real process state cross boundaries. Preserve write order and add tests before moving logic.
- **Context lifetime**: Blindly honoring a disconnected MCP context after persisting cancellation can strand a target. Cleanup needs a bounded continuation context, while pre-commit cancellation can still abort safely.
- **Host approval**: Tool annotations are hints, not authorization. Plugin configuration and server instructions must still recommend prompting for `delegate` and `job_cancel`.
