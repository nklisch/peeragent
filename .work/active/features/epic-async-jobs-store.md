---
id: epic-async-jobs-store
kind: feature
stage: implementing
tags: [infra]
parent: epic-async-jobs
depends_on: []
release_binding: null
gate_origin: null
created: 2026-05-25
updated: 2026-05-25
---

# Async Job Store

## Brief

This feature creates a local job store for async Codex Implement jobs. It defines where job metadata, logs, and final results live, and provides helpers to create and read job records.

The feature exists so async launch/status/result/cancel commands share one durable state model.

## Epic Context

- Parent epic: `epic-async-jobs`
- Position in epic: foundation for all async behavior.

## Foundation References

- `docs/ARCHITECTURE.md` — async flow.
- `docs/CONTRACT.md` — async job contract.

## Architectural Choice

Use a repo-local `.codex-implement/jobs/<job-id>/` directory. Each job has `job.json`, `result.json`, and `codex.log` paths. Job ids are timestamp plus random suffix so they are sortable and unlikely to collide without central coordination.

Alternative considered: use OS temp directories. Rejected because async jobs need durable status across Claude sessions in the same repository.

## Implementation Units

### Unit 1: Job Store Package

**File**: `internal/jobs/store.go`

```go
type Job struct { ... }
type Store struct { Root string }
func NewStore(cwd string) Store
func (s Store) Create(taskText string) (Job, error)
func (s Store) Load(id string) (Job, error)
func (s Store) Save(Job) error
```

**Acceptance Criteria**:
- [ ] New jobs create a directory and metadata file.
- [ ] Existing jobs load from metadata.
- [ ] Job metadata includes id, status, pid, prompt/task reference, timestamps, log path, and result path.

## Implementation Order

1. Add `internal/jobs`.
2. Add store tests using `t.TempDir`.
3. Run tests.

## Testing

### Unit Tests

Create, save, and load a job in a temp directory.

## Risks

Job metadata should not become a second substrate. Keep it operational and local to async process tracking.

<!-- The design pass on this feature (`/agile-workflow:feature-design`, refactor-design, or perf-design) will fill in interfaces, signatures, and implementation units. -->
