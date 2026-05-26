---
id: epic-wrapper-cli-inputs
kind: feature
stage: done
tags: [infra]
parent: epic-wrapper-cli
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Wrapper Inputs

## Brief

This feature teaches the Go wrapper to collect task text from command-line arguments, standard input, and `--prompt-file`. It also resolves the working directory through `--cwd` or the current process directory.

The feature exists so Claude can pass short tasks ergonomically while long tasks and generated prompts have stable input paths. It does not invoke Codex.

## Epic Context

- Parent epic: `epic-wrapper-cli`
- Position in epic: foundation feature for prompt construction and execution.

## Foundation References

- `docs/SPEC.md` — invocation modes and Codex CLI strategy.
- `docs/CONTRACT.md` — CLI synopsis, `--prompt-file`, and `--cwd`.

## Design Decisions

- **Task input forms**: Support CLI args, stdin, and `--prompt-file`.

## Architectural Choice

Create an `internal/input` package that parses wrapper flags and returns a normalized request containing task text and cwd. Keeping input parsing outside `main` makes it directly testable and keeps later prompt/execution packages independent of shell details.

Alternative considered: parse in `main.go`. Rejected because prompt construction and execution need a clean request object, and input precedence deserves unit tests.

## Implementation Units

### Unit 1: Input Package

**File**: `internal/input/input.go`

```go
package input

type Request struct {
	TaskText string
	CWD      string
	JSON     bool
}

func Parse(args []string, stdin io.Reader, getwd func() (string, error)) (Request, error)
```

**Implementation Notes**:
- `--cwd <path>` overrides process cwd.
- `--prompt-file <path>` reads task text from a file.
- Positional args join with spaces.
- If stdin has data, append it after arg text, separated by a blank line.
- If both prompt-file and positional args are present, append args before file content so caller context can prefix the larger prompt.
- Error when no task text is supplied.

**Acceptance Criteria**:
- [ ] Positional args produce task text.
- [ ] `--prompt-file` reads file content.
- [ ] stdin can provide task text.
- [ ] args plus stdin combine deterministically.
- [ ] `--cwd` overrides cwd.
- [ ] missing task text returns an error.

### Unit 2: Main Integration

**File**: `cmd/codex-implement/main.go`

```go
req, err := input.Parse(args, os.Stdin, os.Getwd)
```

**Implementation Notes**:
- Keep placeholder JSON output, but include enough of the parsed request to prove parsing is wired.
- Full result contract lands later.

**Acceptance Criteria**:
- [ ] `go test ./...` passes.
- [ ] `go run ./cmd/codex-implement hello` succeeds.

## Implementation Order

1. Create `internal/input`.
2. Add input parsing tests.
3. Wire `main.go` to use the parser.
4. Run `gofmt` and `go test ./...`.

## Testing

### Unit Tests

Add tests for args, prompt file, stdin, cwd override, combined inputs, and missing text.

## Risks

Reading stdin without blocking is subtle. Treat stdin as an explicit reader in tests and let command execution read whatever Claude pipes to the process.

## Implementation Notes

- Added `internal/input` with normalized request parsing.
- Supports args, stdin, `--prompt-file`, `--cwd`, and default JSON mode.
- Wired `cmd/codex-implement` to use parsed input and emit placeholder JSON containing cwd and task text.
- Added unit coverage for all input paths and missing task text.

## Review

Approved. The parser supports interspersed known flags without swallowing arbitrary task text, has focused unit coverage, and gives downstream prompt/execution work a stable request object.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
