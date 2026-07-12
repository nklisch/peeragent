---
id: epic-mcp-server-job-control
kind: feature
stage: done
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

- **Tool granularity**: Expose `job_status`, `job_result`, and `job_cancel` separately. Cancellation needs distinct destructive/write annotations. Status and result are both read-only but intentionally separate: status is a compact polling operation that never returns potentially large target details, while result retrieves the full terminal contract only when the host asks for it.
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
- [x] CLI `--status` and `--result` output and exit behavior stay compatible.
- [x] Running, complete, failed, cancelled, missing, and corrupt state paths are tested.
- [x] MCP and CLI consume the same returned `result.Result` rather than reading files independently.

### Unit 2: Shared cancellation service
**Files**: `internal/app/cancel.go`, `internal/app/cancel_test.go`, `cmd/peeragent/main.go`
**Story**: `epic-mcp-server-job-control-application-services`

```go
type ProcessController interface {
    TerminateAndWait(pid int, termGrace, killGrace time.Duration) error
}

func (s *Service) CancelJob(context.Context, JobRequest) (result.Result, error)
```

Move the TERM-then-KILL/wait sequence behind one injectable process controller rather than exposing each signal and clock operation separately. Once the locked cancelled result is persisted, finish process termination and PID cleanup independently of the caller context, with the existing bounded grace periods, so a disconnect cannot strand the child. Preserve terminal result conflict detection and idempotent repeated cancellation.

**Acceptance criteria**:
- [x] Completion wins if its terminal result exists before cancellation commits.
- [x] Cancellation wins atomically when no competing terminal result exists.
- [x] TERM escalates to KILL after the existing grace period and PID state is removed.
- [x] Repeated cancellation is idempotent and never signals an already-complete process.
- [x] After the cancelled result is persisted, TERM/KILL and PID removal complete even if the caller context is cancelled mid-call.
- [x] No package under `internal/` calls `os.Exit` or writes `os.Stdout`; validation enforces this adapter boundary.
- [x] Tests use fake process control and do not signal real process groups.

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
- [x] Tool discovery lists all four total server tools with correct schemas and annotations.
- [x] Status and result calls do not mutate persisted state.
- [x] Cancellation is visibly write/destructive to hosts and returns the shared cancelled or race-winning result.
- [x] Invalid/missing ids fail before store or process operations.
- [x] Concurrent MCP calls preserve job-store locking and terminal-state invariants.

## Implementation order

1. `epic-mcp-server-job-control-application-services`
2. `epic-mcp-server-job-control-tools`

## Testing

- Port the existing async CLI tests to assert the shared service first, retaining CLI-level compatibility cases.
- Add table-driven service tests for every persisted status, absent results, missing ids, missing jobs, malformed JSON, repeated cancellation, and competing completion.
- Inject process control and short wait policies to deterministically test TERM/KILL behavior.
- Extend in-memory MCP integration tests to invoke all job tools and inspect tool annotations.
- Add a deterministic test with at least eight concurrent status/result/cancel calls against one job and assert one allowed terminal winner with consistent `job.json`/`result.json`; run `go test -race` for the application and MCP packages in validation.

## Risks

- **Cancellation refactor**: This is the highest-risk unit because terminal files, locks, PIDs, and real process state cross boundaries. Preserve write order and add tests before moving logic.
- **Context lifetime**: Blindly honoring a disconnected MCP context after persisting cancellation can strand a target. Cleanup needs a bounded continuation context, while pre-commit cancellation can still abort safely.
- **Host approval**: Tool annotations are hints, not authorization. Plugin configuration and server instructions must still recommend prompting for `delegate` and `job_cancel`.
- **Cross-repository scope**: Optional `cwd` permits intentional operation outside the server's starting repository. Documentation must identify that capability explicitly and recommend omitting `cwd` unless the user requested cross-repository work.

## Implementation summary
- Execution capability: highest, selected by the autopilot caller because the feature crosses filesystem terminal races, detached process cleanup, generated MCP schemas, destructive tool annotations, and protocol concurrency.
- Review weight: standard (autopilot default).
- Delivered shared `JobStatus`, `JobResult`, and `CancelJob` application services; CLI adapters now format returned results and retain exit-code behavior while MCP shares the same service boundary.
- Delivered context-independent post-commit TERM/KILL cleanup through injectable `ProcessController`, repository-local cwd/job-id normalization, structured exit-code-4 missing-job failures, corrupt-state infrastructure errors, and shared locked terminal transitions.
- Delivered typed `job_status`, `job_result`, and `job_cancel` tools with generated schemas, read-only/destructive/idempotent annotations, complete async workflow instructions, and a combined `ServerService` contract.
- Verification evidence: `go test -race ./internal/app ./internal/mcp ./cmd/peeragent` passed (65 tests); `go test ./...` passed (183 tests across 12 packages); `go vet ./...` passed; `go build -o /tmp/peeragent-job-control ./cmd/peeragent` passed; internal stdout/exit boundary grep found no `os.Exit` or `os.Stdout` references.
- Design decisions recorded in child items: service-level working-directory resolver for one lookup normalizer; context-free process-control port for disconnect-safe cleanup; typed combined MCP service contract to prevent incomplete tool registration.
- Discrepancies from design: none.
- Adjacent issues parked: none.

## Review notes

- Effective review weight: standard; lane: fresh-context deep review due cancellation, process lifecycle, persistence races, and destructive tool semantics.
- Reviewer: GLM 5.2. It manually traced every terminal transition and verified completion/cancellation winners, stale-state repair, post-commit cleanup, schemas, annotations, cwd behavior, concurrency, and CLI compatibility.
- Verdict: approve with comments. Removed six extraction-residue wrappers from `cmd/peeragent/main.go` and pointed tests at the authoritative application functions. The remaining boolean-pointer style nit is optional.
