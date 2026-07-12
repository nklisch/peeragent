---
id: epic-mcp-server-delegation
kind: feature
stage: implementing
tags: [infra]
parent: epic-mcp-server
depends_on: []
release_binding: null
gate_origin: null
created: 2026-07-12
updated: 2026-07-12
---

# MCP delegation server

## Brief

Add the shared application boundary and stdio MCP server needed to expose peeragent delegation outside the CLI. This feature owns protocol initialization, instructions and tool discovery, the blocking delegation call, and async delegation launch. It must derive tool inputs from the same validation rules as CLI requests and return the existing result contract without leaking target or diagnostic output onto protocol stdout.

This feature also establishes the server entry mode and the internal application services that both CLI and MCP adapters call. It does not expose status, result, or cancellation tools, and it does not package the server into host plugins.

## Epic context
- Parent epic: `epic-mcp-server`
- Position in epic: foundation capability — job-control and plugin-distribution features depend on its application and protocol boundaries

## Foundation references
- `docs/SPEC.md` — MCP stdio and execution contracts
- `docs/ARCHITECTURE.md` — inbound adapters, application services, and stdout purity

## Design decisions

- **SDK**: Use the official `github.com/modelcontextprotocol/go-sdk/mcp` SDK at v1.6.1. Its typed `AddTool` API generates and validates input/output schemas, supports structured content, provides in-memory test transports, and is the protocol owner's maintained implementation. This requires raising the module baseline from Go 1.22 to Go 1.23, matching the SDK's declared minimum. Verified 2026-07-12 against the [v1.6.1 release](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.6.1), repository `go.mod`, `examples/server/hello/main.go`, `examples/server/toolschemas/main.go`, and `mcp/{server,tool}.go`: the concrete APIs used below are `mcp.NewServer`, generic `mcp.AddTool`, `mcp.StdioTransport`, `mcp.NewInMemoryTransports`, `ServerOptions.Instructions`, and typed structured output.
- **Tool shape**: Expose one `delegate` tool with an `async` boolean rather than separate blocking and launch tools. This mirrors the existing CLI contract and keeps target/model/access fields in one schema.
- **Failure semantics**: A valid peeragent `failed`, `blocked`, `cancelled`, or `running` result is structured tool output, not an MCP protocol error. Invalid arguments remain invalid-params errors; server/infrastructure failures become MCP tool errors.
- **Working directory**: `cwd` is optional and defaults to the MCP server process working directory. Normalize it once at the application boundary before execution or job creation.
- **Permissions**: Expose explicit `full_access`; omit `worktree` because that mode is intentionally unimplemented. The tool is write-capable and not read-only.
- **Long calls**: Server instructions tell hosts to use `async: true` for substantive agent work and reserve blocking calls for short passes that fit the host's MCP tool timeout.

## Architectural choice

Use Ports & Adapters: extract target execution and async launch from `cmd/peeragent` into an `internal/app` application service, leave target CLI packages and the job store as outbound adapters, and add CLI and MCP as sibling inbound adapters. The command package remains the composition root.

Alternatives rejected:

1. **MCP handlers invoke the peeragent CLI recursively** — lowest initial change, but duplicates serialization, complicates cancellation, adds a process hop, and makes stdout protocol purity fragile.
2. **MCP handlers call current `cmd/peeragent` functions** — avoids a process hop but preserves a package-main dependency and functions that write stdout or call `os.Exit`, making handlers untestable and unsafe.
3. **Hand-roll JSON-RPC/MCP** — avoids a Go baseline bump but assumes protocol maintenance and schema correctness the official SDK already owns.

## Implementation units

### Unit 1: Canonical delegation request normalization
**Files**: `internal/input/input.go`, `internal/input/delegation.go`, `internal/input/input_test.go`
**Story**: `epic-mcp-server-delegation-application-services`

```go
type Delegation struct {
    TaskText   string
    CWD        string
    Agent      string
    FullAccess bool
    Profile    string
    Effort     string
    Model      string
    Resume     string
}

func NormalizeDelegation(raw Delegation, getwd func() (string, error)) (Delegation, error)
```

`Parse` constructs a `Delegation`, calls `NormalizeDelegation`, and then adds CLI-only job/control flags. MCP maps its typed arguments into the same type and calls the same normalizer. Agent, model, effort, task, and cwd rules therefore have one implementation.

**Acceptance criteria**:
- [ ] CLI behavior and existing error messages remain compatible.
- [ ] MCP callers cannot bypass model/effort/agent validation.
- [ ] Empty task text and unresolved cwd fail before any target process starts.

### Unit 2: Shared application service
**Files**: `internal/app/service.go`, `internal/app/execute.go`, `internal/app/async.go`, `internal/app/service_test.go`, `cmd/peeragent/main.go`
**Story**: `epic-mcp-server-delegation-application-services`

```go
type Options struct {
    Executor   TargetExecutor
    Launcher   JobLauncher
    Executable func() (string, error)
}

type Service struct {
    executor   TargetExecutor
    launcher   JobLauncher
    executable func() (string, error)
}

type TargetExecutor interface {
    Execute(context.Context, input.Delegation) (executil.Result, error)
}

type JobLauncher interface {
    Launch(executable string, job jobs.Job) error
}

func NewService(opts Options) *Service
func (s *Service) Delegate(context.Context, input.Delegation) (result.Result, error)
func (s *Service) Launch(context.Context, input.Delegation) (result.Result, error)
```

The production executor routes to Codex, Claude, Gemini, or Z.AI and builds the stable agent prompt. `Delegate` converts target exit/error state to the existing `result.Result`. `Launch` creates the current job spec, starts the detached child, persists its PID, and returns `status: running`. The CLI adapter formats returned values and owns process exit codes; application code never writes stdout or calls `os.Exit`.

**Acceptance criteria**:
- [ ] Blocking and async CLI tests remain behaviorally unchanged.
- [ ] Target failure is represented by the existing failed result contract.
- [ ] Launch cleanup still kills a started child when PID persistence or release fails.
- [ ] No package under `internal/` calls `os.Exit` or writes `os.Stdout`; those remain CLI-adapter concerns and validation enforces the boundary.
- [ ] Application tests inject fake execution and launching without invoking local agent CLIs.

### Unit 3: MCP server and delegate tool
**Files**: `internal/mcp/server.go`, `internal/mcp/tools.go`, `internal/mcp/server_test.go`, `cmd/peeragent/main.go`, `go.mod`, `go.sum`
**Story**: `epic-mcp-server-delegation-stdio-server`

```go
type DelegationService interface {
    Delegate(context.Context, input.Delegation) (result.Result, error)
    Launch(context.Context, input.Delegation) (result.Result, error)
}

type DelegateInput struct {
    Task       string `json:"task" jsonschema:"task for the peer agent"`
    Agent      string `json:"agent,omitempty" jsonschema:"codex, claude, gemini, or zai; defaults to codex"`
    CWD        string `json:"cwd,omitempty" jsonschema:"repository directory; defaults to the server working directory"`
    Profile    string `json:"profile,omitempty" jsonschema:"Codex profile override"`
    Effort     string `json:"effort,omitempty" jsonschema:"target reasoning effort"`
    Model      string `json:"model,omitempty" jsonschema:"target model override"`
    Resume     string `json:"resume,omitempty" jsonschema:"target agent session to resume"`
    FullAccess bool   `json:"full_access,omitempty" jsonschema:"explicitly disable the target sandbox"`
    Async      bool   `json:"async,omitempty" jsonschema:"launch a tracked job and return immediately"`
}

func NewServer(service DelegationService, getwd func() (string, error)) *mcp.Server
func RunStdio(ctx context.Context, service DelegationService) error
```

Register `delegate` with the official typed `mcp.AddTool`; return `result.Result` as structured output. Server instructions explain target selection, explicit full access, async-first guidance for long work, and the later job-control tools. Route the literal first argument `mcp` before normal CLI parsing, then run `server.Run(ctx, &mcp.StdioTransport{})`. Configure SDK logging to stderr or discard; stdout is exclusively the transport.

Use `mcp.NewInMemoryTransports` in tests to initialize a real client session, list tools, validate the generated schema, invoke blocking and async paths, and verify structured outputs and tool errors.

**Acceptance criteria**:
- [ ] `peeragent mcp` completes MCP initialization and advertises only `delegate` in this feature.
- [ ] Blocking calls return the full peeragent result as structured content.
- [ ] `async: true` returns a running result with a job id.
- [ ] MCP `delegate` maps through `input.NormalizeDelegation`; no second agent/model/effort validation path exists.
- [ ] Invalid target/model/effort combinations are rejected before the service executes.
- [ ] Cancellation of the MCP call propagates through context to blocking target execution.
- [ ] Protocol stdout contains no help, logs, target output, or plain JSON result lines.

## Implementation order

1. `epic-mcp-server-delegation-application-services`
2. `epic-mcp-server-delegation-stdio-server`

## Testing

- Preserve all `cmd/peeragent` CLI and async tests as compatibility coverage.
- Add table-driven normalization tests over CLI and MCP-originated request shapes.
- Unit-test application services with fake executor/launcher ports, including target failures and launch cleanup failures.
- Use official SDK in-memory transports for initialization, tool discovery, schema rejection, blocking results, async results, context cancellation, and concurrent calls.
- Add a subprocess stdio smoke test that sends initialize/list-tools frames and asserts every stdout line is valid MCP protocol output while diagnostics remain on stderr.

## Risks

- **Go baseline**: Official SDK v1.6.1 requires Go 1.23. CI derives its toolchain from `go.mod`, but implementation must also audit workflows, build scripts, and installation docs for independent 1.22 assumptions and state the new minimum.
- **Main-package extraction**: Directly moving behavior can alter exit codes or races. Keep compatibility tests green after each application-service move rather than rewriting tests around new behavior.
- **SDK schema details**: Typed schema generation is authoritative, but optional fields and descriptions must be inspected in a real list-tools response; customize only where generated constraints are insufficient.
- **Timeouts**: Blocking agent work can outlive host MCP defaults. Async guidance is a product constraint, not something the server can override for every host.
