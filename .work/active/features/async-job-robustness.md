---
id: async-job-robustness
kind: feature
stage: implementing
tags: [infra]
parent: null
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-27
updated: 2026-05-27
---

# Async Job Robustness

## Brief

The first async-jobs implementation (epic-async-jobs, done) ships with five
latent bugs uncovered by peer-reviewed investigation: a blanket stdin drain
that hangs the parent when the invoking host holds stdin open, a child
process that reparses argv and ignores the persisted prompt source, an
initial PID-save race against fast-failing children, a wrapper that shares
the parent's session and has no process-group cancel, and a finish path
that can overwrite cancelled status.

This feature tightens the async path end-to-end so it survives realistic
invocation contexts (held-open stdin, host disconnects, cancellation
mid-run) and so the on-disk job becomes the single source of truth for
what the child should execute. Scope is remediation only — no new
capabilities, no new flags.

## Foundation References

- `docs/CONTRACT.md` — async state layout under `.peeragent/jobs/<id>/`
  and `--cancel` semantics; rolls forward when sidecar files land.
- `docs/ARCHITECTURE.md` — async flow and process-group handling; rolls
  forward when Setsid + group-kill ship.

## Strategic Decisions

- **Feature shape**: One feature with three child stories (stdin gate;
  job-as-source-of-truth; pid-sidecar + Setsid + group-kill +
  terminal-status guards). Coheres as one remediation arc.
- **Claude 90s timeout (surfaced during peer review)**: Skip entirely
  for now — not tracked. User will revisit if it bites again.
- **Job storage migration**: Clean break. Bump `job.json` schema: drop
  `TaskText` from the embedded record, persist prompt to a sidecar
  `prompt.txt`, move PID to a sidecar `pid` file, add a `spec` block
  carrying agent/profile/effort/model/resume/full-access/worktree/json.
  Pre-1.0 (v0.2.x); in-flight jobs at upgrade time are acceptable
  collateral.

## Design Decisions

- **Cancellation lifecycle ordering**: `cancelJob` writes `cancelled`
  to `job.json` and a minimal `result.json` (both with terminal-status
  guard) BEFORE signaling the process group. Child's `finishAsyncJob`
  reload-and-guards before writing — refuses to overwrite terminal
  states (`cancelled`, `failed`, `complete`). No Go signal handler in
  the child; the file-state is the coordination point, signals are pure
  process termination. Simpler than signal-based cooperative shutdown
  and works even when the child is wedged in I/O.
- **SIGTERM → SIGKILL grace period**: 5 seconds. Long enough for codex
  / claude / gemini to flush partial state, short enough that
  cancellation feels responsive. Not configurable in v0.2.x.
- **Atomic file writes**: write to `<path>.tmp` + `os.Rename` for
  `job.json`, `result.json`, `pid`, `prompt.txt`. Standard POSIX
  rename-is-atomic pattern. Reload-and-guard happens before the tmp
  write; the rename commits.
- **`prompt.txt` format**: raw bytes, exactly what `input.Parse`
  resolved. No transformation.
- **`pid` file format**: decimal integer + newline. `strconv.Atoi` on
  read.
- **ExecSpec position in `job.json`**: nested under a `"spec"` key
  rather than flattened. Forward-compat: future fields land inside spec
  without colliding with `id`/`status`/`created_at`/etc.
- **ExecSpec contents**: `agent`, `access` (default | full-access |
  worktree-style), `profile`, `effort`, `model`, `resume`, `json`. No
  `cwd` — that stays top-level on `Job` since the status/result/cancel
  paths also need it.

## Architectural Choice

Three plausible approaches were considered:

1. **Inline mutation of existing modules** — smallest diff; everything
   stays in `main.go` / `input.go` / `store.go`.
2. **Extract a new `jobrunner` package** owning the full async lifecycle
   (launch/run/cancel/finish) — cleanest separation, but a bigger diff
   than the remediation needs.
3. **Inline lifecycle + typed `ExecSpec` in `internal/jobs/`** —
   keeps `main.go` as the orchestrator (already its role), gives the
   spec a typed home in the jobs package, splits Setsid into platform
   files.

**Chose option 3.** Lifecycle stays where it is (no large refactor
under a remediation feature). The new responsibility (typed spec +
sidecar I/O) lands in `internal/jobs/`, where `Store` already owns
the on-disk job directory. Platform split for Setsid is a two-file
addition (`internal/jobs/launch_unix.go`, `internal/jobs/launch_windows.go`).

## Implementation Units

### Unit 1: stdin gate + `--job-run` allow-list (Story A)

**File**: `internal/input/input.go`
**Story**: `async-job-robustness-stdin-gate`

Replace the unconditional `io.ReadAll(stdin)` block at lines 64-72 with
a gated read:

```go
// Only read stdin when no positional, no --prompt-file, no job-control
// flag was supplied. Skip TTYs. Preserve current behavior for tests
// that pass non-*os.File readers (strings.Reader etc).
if stdin != nil && !isInteractiveTTY(stdin) {
    content, err := io.ReadAll(stdin)
    if err != nil {
        return Request{}, fmt.Errorf("read stdin: %w", err)
    }
    if text := strings.TrimSpace(string(content)); text != "" {
        parts = append(parts, text)
    }
}
```

```go
// isInteractiveTTY reports whether stdin is a TTY (character device).
// Non-*os.File readers (e.g. strings.Reader from tests) return false,
// preserving merge semantics.
func isInteractiveTTY(stdin io.Reader) bool {
    file, ok := stdin.(*os.File)
    if !ok {
        return false
    }
    info, err := file.Stat()
    if err != nil {
        return false
    }
    return info.Mode()&os.ModeCharDevice != 0
}
```

Add `parsed.jobRunID != ""` to the no-task-text allow list at lines
75-81:

```go
if taskText == "" {
    if parsed.statusJobID != "" || parsed.resultJobID != "" ||
       parsed.cancelJobID != "" || parsed.jobRunID != "" {
        taskText = ""
    } else {
        return Request{}, errors.New("no task text supplied")
    }
}
```

**Implementation Notes**:
- Existing `TestParseStdin` and `TestParseCombinesInputs` use
  `strings.NewReader` — they fall through `isInteractiveTTY` returning
  false, and behavior is preserved.
- Existing `TestParseRequiresTaskText` passes `nil` stdin — still errors.
- The new test exercises a real `*os.File` on a TTY-ish file descriptor.
  Cross-platform TTY testing is fiddly; use `os.Pipe()` to construct a
  non-TTY file pair (proves non-TTY reads), and skip the actual TTY
  positive test if we can't open `/dev/ptmx` (mark with `t.Skip`).

**Acceptance Criteria**:
- [ ] `Parse(["--job-run","id"], nil, getwd)` returns no error,
      empty `TaskText`, `JobRunID == "id"`.
- [ ] `Parse(["task"], strings.NewReader("ctx"), getwd)` still merges:
      `TaskText == "task\n\nctx"`.
- [ ] `Parse(["task"], strings.NewReader(""), getwd)` yields
      `TaskText == "task"` (existing merge behavior).
- [ ] New test: `*os.File` non-TTY pipe with data is read and merged.
- [ ] New test: `*os.File` TTY-mode stdin is skipped (best-effort; skip
      test if `/dev/ptmx` unavailable).
- [ ] Repro fix: `( sleep 5 | peeragent --async "task" )` returns
      job_id within 1s instead of hanging.

---

### Unit 2: ExecSpec + prompt sidecar (Story B, foundation)

**File**: `internal/jobs/store.go`
**Story**: `async-job-robustness-job-source-of-truth`

Change `Job` struct:

```go
type Job struct {
    ID         string    `json:"id"`
    Status     string    `json:"status"`
    CWD        string    `json:"cwd,omitempty"`
    Spec       ExecSpec  `json:"spec"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
    LogPath    string    `json:"log_path"`
    ResultPath string    `json:"result_path"`
    PromptPath string    `json:"prompt_path"`
    PIDPath    string    `json:"pid_path"`
}

type ExecSpec struct {
    Agent      string `json:"agent"`
    Access     string `json:"access"`            // "default" | "full-access" | "worktree"
    Profile    string `json:"profile,omitempty"`
    Effort     string `json:"effort,omitempty"`
    Model      string `json:"model,omitempty"`
    Resume     string `json:"resume,omitempty"`
    JSON       bool   `json:"json"`
    FullAccess bool   `json:"full_access,omitempty"`
    Worktree   bool   `json:"worktree,omitempty"`
}
```

Drop `TaskText` and `PID` fields. Add helpers:

```go
// Create now takes spec + prompt and writes everything before returning.
func (s Store) Create(cwd string, spec ExecSpec, prompt string) (Job, error)

func (s Store) WritePrompt(id, prompt string) error
func (s Store) ReadPrompt(id string) (string, error)
func (s Store) WritePID(id string, pid int) error
func (s Store) ReadPID(id string) (int, error)
func (s Store) RemovePID(id string) error   // child clears on exit

// SaveGuarded atomically writes job.json but refuses to overwrite a
// terminal status. Returns the prior status if guarded.
func (s Store) SaveGuarded(job Job) (priorStatus string, err error)
```

`SaveGuarded` semantics:

```go
func (s Store) SaveGuarded(job Job) (string, error) {
    existing, loadErr := s.Load(job.ID)
    if loadErr == nil && isTerminal(existing.Status) &&
       existing.Status != job.Status {
        return existing.Status, nil  // refuse overwrite
    }
    return "", s.Save(job)
}

func isTerminal(s string) bool {
    switch s {
    case "complete", "failed", "cancelled":
        return true
    }
    return false
}
```

`Save` itself uses atomic write (tmp + rename):

```go
func (s Store) Save(job Job) error {
    job.UpdatedAt = time.Now().UTC()
    content, err := json.MarshalIndent(job, "", "  ")
    if err != nil {
        return err
    }
    if err := os.MkdirAll(s.jobDir(job.ID), 0o755); err != nil {
        return err
    }
    target := filepath.Join(s.jobDir(job.ID), "job.json")
    return atomicWrite(target, append(content, '\n'))
}

func atomicWrite(target string, data []byte) error {
    tmp := target + ".tmp"
    if err := os.WriteFile(tmp, data, 0o644); err != nil {
        return err
    }
    return os.Rename(tmp, target)
}
```

**Implementation Notes**:
- `prompt.txt`, `pid`, `result.json`, `job.json` all go through
  `atomicWrite`.
- `Job.PromptPath` and `Job.PIDPath` are persisted so external tools
  can find sidecars without recomputing paths. They mirror the existing
  `LogPath` / `ResultPath` pattern.
- `Store.Create` now takes the resolved spec and prompt up-front; the
  signature change cascades into `launchAsync`.

**Acceptance Criteria**:
- [ ] `Job.TaskText` field removed; reading an old-shape job.json
      ignores the field (extra JSON keys are silently dropped by Go).
- [ ] `Create(cwd, spec, prompt)` writes `job.json` + `prompt.txt` and
      returns a job with no PID yet.
- [ ] `WritePID` + `ReadPID` round-trip with newline-terminated decimal.
- [ ] `SaveGuarded` refuses to overwrite `complete` / `failed` /
      `cancelled` with a non-terminal status.
- [ ] `SaveGuarded` permits terminal → same-terminal writes (idempotent).
- [ ] Atomic write proof: tmp file appears then disappears; final file
      exists; partial writes do not leave a half-written `job.json`.

---

### Unit 3: launchAsync writes spec/prompt, child loads from job (Story B, wiring)

**File**: `cmd/peeragent/main.go`
**Story**: `async-job-robustness-job-source-of-truth`

`launchAsync` becomes:

```go
func launchAsync(args []string, req input.Request) error {
    store := jobs.NewStore(req.CWD)
    spec := jobs.ExecSpec{
        Agent: agentID(req), Access: accessMode(req),
        Profile: req.Profile, Effort: req.Effort, Model: req.Model,
        Resume: req.Resume, JSON: req.JSON,
        FullAccess: req.FullAccess, Worktree: req.Worktree,
    }
    job, err := store.Create(req.CWD, spec, req.TaskText)
    if err != nil {
        return err
    }

    executable, err := os.Executable()
    if err != nil {
        return err
    }

    childArgs := []string{"--job-run", job.ID, "--cwd", req.CWD}
    cmd := exec.Command(executable, childArgs...)
    cmd.Dir = req.CWD
    applyDetachAttrs(cmd)  // platform split — Setsid on unix

    logFile, err := os.OpenFile(job.LogPath,
        os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
    if err != nil {
        return err
    }
    defer logFile.Close()
    cmd.Stdout = logFile
    cmd.Stderr = logFile

    if err := cmd.Start(); err != nil {
        return err
    }
    if err := store.WritePID(job.ID, cmd.Process.Pid); err != nil {
        // best-effort: log to job log, do not abort start
        fmt.Fprintf(logFile, "pid sidecar write failed: %v\n", err)
    }
    if err := cmd.Process.Release(); err != nil {
        return err
    }

    return writeResult(req, runningResult(req, job))
}
```

Note: parent no longer calls `store.Save(job)` after `cmd.Start()`.
`job.json` becomes child-owned for status/result transitions. The
initial `job.json` (status=running) is written by `store.Create`.

`runAsyncJob` becomes:

```go
func runAsyncJob(req input.Request) error {
    store := jobs.NewStore(req.CWD)
    job, err := store.Load(req.JobRunID)
    if err != nil {
        return err
    }
    prompt, err := store.ReadPrompt(job.ID)
    if err != nil {
        return finishAsyncJob(store, job, failedResult(req, job,
            fmt.Errorf("read prompt: %w", err)))
    }

    // Rebuild a Request from the spec + persisted CWD + prompt.
    execReq := requestFromJob(job, prompt)
    if job.Spec.Worktree {
        return finishAsyncJob(store, job, worktreeNotImplemented(execReq, job))
    }
    execResult, execErr := executeRequest(context.Background(), execReq)
    res := resultFromExecution(execReq, execResult, execErr)
    res.Metadata.JobID = job.ID
    return finishAsyncJob(store, job, res)
}

func requestFromJob(job jobs.Job, prompt string) input.Request {
    return input.Request{
        TaskText:   prompt,
        CWD:        job.CWD,
        JSON:       job.Spec.JSON,
        Agent:      job.Spec.Agent,
        FullAccess: job.Spec.FullAccess,
        Worktree:   job.Spec.Worktree,
        Profile:    job.Spec.Profile,
        Effort:     job.Spec.Effort,
        Model:      job.Spec.Model,
        Resume:     job.Spec.Resume,
    }
}
```

**Implementation Notes**:
- Child's `os.Stdin` is `/dev/null` (Go default for nil Stdin). After
  Story A, `input.Parse` won't try to read it anyway — but the child
  doesn't go through `input.Parse` for the prompt at all.
- `applyDetachAttrs` is defined in Unit 5 (platform-split files).
- Pre-Story C+D, this leaves `cancelJob` reading PID from `job.PID`
  which we just removed. Story C+D handles that path.

**Acceptance Criteria**:
- [ ] Child argv contains exactly `--job-run <id>` and `--cwd <cwd>`,
      no other flags.
- [ ] Async run with stdin-only original prompt now succeeds: parent
      reads stdin, persists to `prompt.txt`, child loads it.
- [ ] Parent never writes `job.json` after `cmd.Start()`.
- [ ] Worktree-mode job records `failed` with the not-implemented
      message via `finishAsyncJob` (terminal-state path exercised).

---

### Unit 4: finishAsyncJob terminal guards (Story C+D)

**File**: `cmd/peeragent/main.go`
**Story**: `async-job-robustness-process-lifecycle`

```go
func finishAsyncJob(store jobs.Store, job jobs.Job, res result.Result) error {
    // Reload — cancelJob may have written cancelled state.
    if current, err := store.Load(job.ID); err == nil {
        if isTerminal(current.Status) && current.Status == "cancelled" {
            // cancel won. Do not overwrite job.json or result.json.
            _ = store.RemovePID(job.ID)
            return nil
        }
    }
    encoded, err := result.FormatJSON(res)
    if err != nil {
        return err
    }
    // Guard result.json against overwrite of a cancelled result file.
    if existing, err := os.ReadFile(job.ResultPath); err == nil {
        var prior result.Result
        if json.Unmarshal(existing, &prior) == nil &&
           prior.Status == result.StatusCancelled {
            _ = store.RemovePID(job.ID)
            return nil
        }
    }
    if err := atomicWriteFile(job.ResultPath,
        append(encoded, '\n')); err != nil {
        return err
    }
    if res.Status == result.StatusSuccess {
        job.Status = "complete"
    } else {
        job.Status = "failed"
    }
    if _, err := store.SaveGuarded(job); err != nil {
        return err
    }
    _ = store.RemovePID(job.ID)
    return nil
}
```

**Implementation Notes**:
- Child clears PID sidecar on exit via `store.RemovePID`. cancelJob
  uses presence of pid file to know whether to attempt group-kill.
- Two guard layers: explicit cancelled-status check (fast path) +
  `SaveGuarded` (covers other terminal-vs-terminal races).
- `atomicWriteFile` lives in `internal/jobs/` and is exported.

**Acceptance Criteria**:
- [ ] If `job.json.status == "cancelled"` before finish, finish leaves
      both `job.json` and `result.json` untouched.
- [ ] If `result.json` already contains a cancelled result, finish
      leaves it untouched.
- [ ] If neither is cancelled, finish writes both atomically.
- [ ] After finish, `pid` sidecar is removed.

---

### Unit 5: Setsid platform split + group cancel (Story C+D)

**Files**:
- `internal/jobs/launch_unix.go` (build tag: `//go:build unix`)
- `internal/jobs/launch_windows.go` (build tag: `//go:build windows`)
- `cmd/peeragent/main.go` (cancel rewrite)
**Story**: `async-job-robustness-process-lifecycle`

`internal/jobs/launch_unix.go`:

```go
//go:build unix

package jobs

import (
    "os/exec"
    "syscall"
)

func ApplyDetachAttrs(cmd *exec.Cmd) {
    if cmd.SysProcAttr == nil {
        cmd.SysProcAttr = &syscall.SysProcAttr{}
    }
    cmd.SysProcAttr.Setsid = true
}

// SignalProcessGroup sends sig to the process group led by pid.
func SignalProcessGroup(pid int, sig syscall.Signal) error {
    return syscall.Kill(-pid, sig)
}
```

`internal/jobs/launch_windows.go`:

```go
//go:build windows

package jobs

import (
    "errors"
    "os/exec"
    "syscall"
)

func ApplyDetachAttrs(cmd *exec.Cmd) {
    // No equivalent on Windows in current scope; child remains in
    // parent's job object. Async cancel best-effort only.
}

func SignalProcessGroup(pid int, sig syscall.Signal) error {
    return errors.New("process-group signalling not implemented on windows")
}
```

`cancelJob` rewrite in `cmd/peeragent/main.go`:

```go
func cancelJob(req input.Request) error {
    store := jobs.NewStore(req.CWD)
    job, err := store.Load(req.CancelJobID)
    if err != nil {
        return writeJobLookupFailure(req, req.CancelJobID, err)
    }
    if isTerminalJobStatus(job.Status) {
        return writeResult(req, alreadyDoneResult(req, job))
    }

    // 1. Write cancelled result first — guarded against terminal overwrite.
    cancelled := cancelledResult(req, job)
    encoded, err := result.FormatJSON(cancelled)
    if err != nil {
        return err
    }
    _ = jobs.AtomicWriteFile(job.ResultPath, append(encoded, '\n'))

    // 2. Mark job.json cancelled, guarded.
    job.Status = "cancelled"
    if _, err := store.SaveGuarded(job); err != nil {
        return err
    }

    // 3. Kill the process group.
    pid, perr := store.ReadPID(job.ID)
    if perr == nil && pid > 0 {
        _ = jobs.SignalProcessGroup(pid, syscall.SIGTERM)
        if !waitForExit(pid, 5*time.Second) {
            _ = jobs.SignalProcessGroup(pid, syscall.SIGKILL)
        }
    }
    _ = store.RemovePID(job.ID)

    return writeResult(req, cancelled)
}

// waitForExit polls /proc (or kill -0) until pid is gone or timeout.
func waitForExit(pid int, timeout time.Duration) bool {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if err := syscall.Kill(pid, 0); err != nil {
            return true  // process gone
        }
        time.Sleep(100 * time.Millisecond)
    }
    return false
}
```

**Implementation Notes**:
- Cancel writes state FIRST, then signals. If the child happens to
  finish before the signal lands, the terminal guard in
  `finishAsyncJob` ensures cancel's writes win.
- `waitForExit` uses `kill -0` (signal 0 is the existence check on
  POSIX). Cross-platform shim available via `os.FindProcess` if needed.
- 5-second grace is a hardcoded constant; documented in CONTRACT.md.

**Acceptance Criteria**:
- [ ] On unix builds, async children run with `Setsid` set (process
      group == PID).
- [ ] `cancelJob` writes `job.json:cancelled` and `result.json:cancelled`
      atomically before signalling.
- [ ] Cancel reaches both the wrapper child AND its grandchild (codex
      process). Test target: spawn a `bash -c 'trap "" TERM; sleep 100'`
      tree, cancel, confirm group-kill terminates the whole subtree
      within `5s + epsilon`.
- [ ] Windows build compiles (`GOOS=windows go build ./...`) even
      though cancel is degraded.
- [ ] If `pid` sidecar is missing during cancel (race or upgrade
      collateral), cancelled status still lands; signal step is
      skipped with no error.

---

### Unit 6: foundation-doc roll-forward (Story C+D, ships with the behavior)

**Files**:
- `docs/CONTRACT.md`
- `docs/ARCHITECTURE.md`
**Story**: `async-job-robustness-process-lifecycle`

Per the rolling-foundation principle, edit in place — no "previously"
notes.

`docs/CONTRACT.md` — update the `Async State` section:

```text
.peeragent/jobs/<job-id>/
  job.json       lifecycle + ExecSpec, child-owned after launch
  prompt.txt     resolved task text, parent-written, child-read
  pid            child PGID for cancel, present while running
  agent.log      combined stdout+stderr from the child
  result.json    final result, written by child OR by --cancel
```

Tighten `--cancel` description to mention process-group kill and the
5s grace period.

`docs/ARCHITECTURE.md` — update the async flow to describe:
- Parent resolves the request, writes `job.json` + `prompt.txt`, spawns
  child as `peeragent --job-run <id> --cwd <cwd>` with Setsid.
- Child loads spec + prompt from the job dir; never reparses execution
  flags.
- Cancel writes terminal state before signalling; child finish reload-
  guards against overwriting cancelled state.

**Acceptance Criteria**:
- [ ] CONTRACT.md Async State section reflects the new layout exactly.
- [ ] ARCHITECTURE.md async flow section describes the new contract.
- [ ] No `previously`/`in v0.2.x`/migration prose anywhere.

## Implementation Order

1. **Story A** — `async-job-robustness-stdin-gate` — independent, ships
   alone as the smallest unblocker.
2. **Story B** — `async-job-robustness-job-source-of-truth` — depends
   on A so the new no-task-text allow-list for `--job-run` is in place
   before the child is spawned without prompt argv.
3. **Story C+D** — `async-job-robustness-process-lifecycle` — depends
   on B so `Job.PID` is gone and the lifecycle owns sidecars.

## Testing

### Unit tests

- `internal/input/input_test.go` — extend with:
  - `--job-run` allow-list path
  - non-TTY `*os.File` pipe read
  - TTY-shaped `*os.File` skip (best-effort, `t.Skip` if `/dev/ptmx`
    unavailable)
- `internal/jobs/store_test.go` — extend with:
  - `Create(cwd, spec, prompt)` round-trip
  - `WritePrompt` / `ReadPrompt` round-trip
  - `WritePID` / `ReadPID` / `RemovePID` round-trip
  - `SaveGuarded` refuses cancelled → running write
  - `SaveGuarded` permits cancelled → cancelled (idempotent)
  - Atomic write: no `.tmp` left after successful write
- `cmd/peeragent/main_test.go` — extend with:
  - `requestFromJob` reconstruction matches a known-good request
  - `finishAsyncJob` does not overwrite cancelled
  - `cancelJob` writes terminal state when pid sidecar missing

### Integration tests

A new `cmd/peeragent/main_async_test.go` that:
- Spawns the actual peeragent binary as a subprocess with `--async`,
  using a fake "agent" CLI on PATH that just sleeps and writes
  signal-handling-friendly output (a `bash -c 'sleep N'` script
  wrapped to look like codex).
- Confirms job_id returns within 1s even when parent stdin is held
  open.
- Confirms cancel terminates the whole group within 6s.
- Confirms cancel-then-natural-finish races resolve to cancelled.

This is the load-bearing test for the C+D story; the unit tests prove
the building blocks, the integration test proves they compose.

## Risks

- **TTY testing portability**: the non-TTY pipe positive test is
  straightforward; the TTY-mode skip test depends on `/dev/ptmx`
  availability. On constrained CI without ptmx, the test must
  `t.Skip` rather than fail. Already designed in.
- **`kill -0` semantics**: `syscall.Kill(pid, 0)` returns
  `ESRCH` when the process is gone, `EPERM` if it exists but we lack
  permission. We treat any non-nil err as "process gone enough." If a
  long-running EUID-mismatched zombie ever holds a pid we'd loop, but
  that's not a realistic scenario for a same-user wrapper.
- **PID reuse during cancel race**: between cancel reading the pid
  sidecar and signalling, the original child could exit and the
  kernel could reuse the pid for an unrelated process. Mitigated by
  cancel writing terminal state FIRST (so a delayed signal landing
  on a reused pid is rare) and by `Setsid` making the pid == pgid
  (a reused pid is unlikely to also lead a process group containing
  a peeragent child). Not a perfect race-free design — pre-1.0
  acceptable.
- **Windows degradation**: cancel can't reach a process group on
  Windows in this design. Documented in `launch_windows.go`. peeragent
  doesn't ship a Windows release today; this is forward-compat hygiene
  rather than a supported path.

## Foundation-Doc Roll-Forward (lands in Story C+D)

- `docs/CONTRACT.md`: Async State section + `--cancel` description.
- `docs/ARCHITECTURE.md`: async flow + detachment.

## Verification

`go test ./...` plus the new integration test plus a manual reproduction:
```
( sleep 30 | peeragent --async "task" )   # returns job_id < 1s
peeragent --cancel <job-id>                # full subtree gone < 6s
peeragent --result <job-id>                # status: cancelled
```

## Out of Scope

- Claude 90s timeout surfaced during peer review.
- Worktree mode implementation — still returns not-implemented.
- Bidirectional streaming / MCP / A2A transport.
- Cancel-grace-period configurability.
- Windows cancel semantics beyond "compiles."

<!-- Stories live at .work/active/stories/async-job-robustness-*.md -->
