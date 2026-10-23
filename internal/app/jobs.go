package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/nklisch/peeragent/internal/heavy"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
	"github.com/nklisch/peeragent/internal/version"
)

// JobRequest identifies a repository-local asynchronous job. CWD is optional
// at the application boundary; an empty value uses the per-user job index,
// then falls back to the service's working-directory port for unindexed jobs.
type JobRequest struct {
	CWD   string
	JobID string
}

// JobStatus returns the compact lifecycle state for one repository-local job.
// Missing job state is a structured exit-code-4 result for CLI compatibility;
// malformed persisted state is returned as an infrastructure error.
func (s *Service) JobStatus(ctx context.Context, raw JobRequest) (result.Result, error) {
	req, err := s.normalizeJobRequest(ctx, raw)
	if err != nil {
		return result.Result{}, err
	}

	job, err := jobs.NewStore(req.CWD).Load(req.JobID)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return JobLookupFailureResult(req, err), nil
		}
		return result.Result{}, fmt.Errorf("load async job %q: %w", req.JobID, err)
	}

	res := jobStatusResult(req, job)
	if err := addMessaging(ctx, &res, job, false); err != nil {
		return result.Result{}, err
	}
	if err := addProgress(&res, job); err != nil {
		return result.Result{}, err
	}
	return res, nil
}

// JobResult returns the persisted terminal result, or running while the child
// has not produced result.json yet. Result JSON is decoded here so job-state
// interpretation remains centralized and corrupt storage stays distinguishable
// from an ordinary in-flight job.
func (s *Service) JobResult(ctx context.Context, raw JobRequest) (result.Result, error) {
	return s.jobResult(ctx, raw, true)
}

// PollJobResult is the waiter's result read. In-flight messages can be large;
// waiters need only lifecycle metadata until a terminal result is available.
func (s *Service) PollJobResult(ctx context.Context, raw JobRequest) (result.Result, error) {
	return s.jobResult(ctx, raw, false)
}

func (s *Service) jobResult(ctx context.Context, raw JobRequest, includeRunningMessage bool) (result.Result, error) {
	req, err := s.normalizeJobRequest(ctx, raw)
	if err != nil {
		return result.Result{}, err
	}

	store := jobs.NewStore(req.CWD)
	job, err := store.Load(req.JobID)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return JobLookupFailureResult(req, err), nil
		}
		return result.Result{}, fmt.Errorf("load async job %q: %w", req.JobID, err)
	}

	content, err := os.ReadFile(job.ResultPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			res := runningJobResultForRequest(req, job)
			res.Metadata.MessagePath = job.MessagePath
			res.Metadata.FinalReceived = job.FinalReceived
			if !includeRunningMessage {
				return res, nil
			}
			if err := enrichJobResult(&res, job); err != nil {
				return result.Result{}, err
			}
			err := addMessaging(ctx, &res, job, true)
			return res, err
		}
		return result.Result{}, fmt.Errorf("read async job %q result: %w", req.JobID, err)
	}

	var stored result.Result
	if err := json.Unmarshal(content, &stored); err != nil {
		return result.Result{}, fmt.Errorf("decode async job %q result: %w", req.JobID, err)
	}
	if err := enrichJobResult(&stored, job); err != nil {
		return result.Result{}, err
	}
	if err := addMessaging(ctx, &stored, job, true); err != nil {
		return result.Result{}, err
	}
	return stored, nil
}

func (s *Service) normalizeJobRequest(ctx context.Context, raw JobRequest) (JobRequest, error) {
	if s == nil {
		return JobRequest{}, errors.New("job service is unavailable")
	}
	if ctx == nil {
		return JobRequest{}, errors.New("job context is nil")
	}
	if err := ctx.Err(); err != nil {
		return JobRequest{}, err
	}

	raw.JobID = strings.TrimSpace(raw.JobID)
	if raw.JobID == "" {
		return JobRequest{}, errors.New("job id is required")
	}
	if err := jobs.ValidateID(raw.JobID); err != nil {
		return JobRequest{}, err
	}
	raw.CWD = strings.TrimSpace(raw.CWD)
	if raw.CWD == "" {
		if s.workingDirectory == nil {
			return JobRequest{}, errors.New("resolve cwd: working-directory resolver is nil")
		}
		indexed, err := jobs.LookupJob(raw.JobID)
		if err != nil {
			return JobRequest{}, err
		}
		if indexed != "" {
			raw.CWD = indexed
			return raw, nil
		}
		cwd, err := s.workingDirectory()
		if err != nil {
			return JobRequest{}, fmt.Errorf("resolve cwd: %w", err)
		}
		raw.CWD = strings.TrimSpace(cwd)
		if raw.CWD == "" {
			return JobRequest{}, errors.New("resolve cwd: working directory is empty")
		}
	}
	return raw, nil
}

func jobStatusResult(req JobRequest, job jobs.Job) result.Result {
	res := result.Result{
		Status:       ResultStatusFromJob(string(job.Status)),
		Summary:      fmt.Sprintf("Async job %s is %s", job.ID, job.Status),
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata: result.Metadata{
			Version:          version.Version,
			MessagingEnabled: job.MessagingEnabled,
			KeepTerminalEnv:  job.Spec.KeepTerminalEnv,
			AgentSession:     job.Session(),
			CWD:              req.CWD,
			ExitCode:         0,
			JobID:            job.ID,
		},
	}
	if job.FinalReceived && !jobs.IsTerminalStatus(job.Status) {
		res.Summary = "final message received, process still alive"
	}
	res.Metadata.FinalReceived = job.FinalReceived
	if job.Status == jobs.StatusCapacity {
		res.Summary = result.CapacitySummary(job.Session())
	}
	if job.Status == jobs.StatusOutputLimit {
		res.Summary = result.OutputLimitSummary(job.Session())
	}
	return res
}

func runningJobResultForRequest(req JobRequest, job jobs.Job) result.Result {
	return result.Result{
		Status:       result.StatusRunning,
		Summary:      fmt.Sprintf("Async job %s is still running", job.ID),
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata: result.Metadata{
			Version:          version.Version,
			MessagingEnabled: job.MessagingEnabled,
			KeepTerminalEnv:  job.Spec.KeepTerminalEnv,
			AgentSession:     job.Session(),
			CWD:              req.CWD,
			ExitCode:         0,
			JobID:            job.ID,
		},
	}
}

// JobLookupFailureResult preserves the CLI's established missing-job contract
// for all inbound adapters without terminating the current process.
func JobLookupFailureResult(req JobRequest, err error) result.Result {
	return result.Result{
		Status:       result.StatusFailed,
		Summary:      fmt.Sprintf("async job %s lookup failed: %v", req.JobID, err),
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata: result.Metadata{
			CWD:      req.CWD,
			ExitCode: 4,
			JobID:    req.JobID,
		},
	}
}

// ResultStatusFromJob is the one mapping from persisted lifecycle values to
// the public result contract. Unknown values stay conservatively running.
func ResultStatusFromJob(status string) result.Status {
	switch jobs.Status(status) {
	case jobs.StatusComplete:
		return result.StatusSuccess
	case jobs.StatusOutputLimit:
		return result.StatusOutputLimit
	case jobs.StatusCapacity:
		return result.StatusCapacity
	case jobs.StatusFailed:
		return result.StatusFailed
	case jobs.StatusCancelled:
		return result.StatusCancelled
	default:
		return result.StatusRunning
	}
}

// IsTerminalJobStatus keeps the application-facing helper compatible with its
// string boundary while delegating terminal membership to the jobs package.
func IsTerminalJobStatus(status string) bool {
	return jobs.IsTerminalStatus(jobs.Status(status))
}

// JobStatusFromResult maps an application result back to the typed persisted
// job lifecycle value used by job.json.
func JobStatusFromResult(status result.Status) jobs.Status {
	switch status {
	case result.StatusOutputLimit:
		return jobs.StatusOutputLimit
	case result.StatusCapacity:
		return jobs.StatusCapacity
	case result.StatusSuccess:
		return jobs.StatusComplete
	case result.StatusCancelled:
		return jobs.StatusCancelled
	default:
		return jobs.StatusFailed
	}
}

// FinishJob commits a natural child result using the same lock and terminal
// conflict rules as cancellation. It remains an application operation even
// though the child-run CLI is the adapter that invokes it.
func FinishJob(store jobs.Store, job jobs.Job, res result.Result) error {
	pid, _ := store.ReadPID(job.ID)
	session := res.Metadata.AgentSession
	if session == "" {
		session = job.Session()
	}
	res.PendingHeavySteps = heavy.Pending(job.CWD, pid, session)
	targetStatus := JobStatusFromResult(res.Status)
	err := store.WithJobLock(job.ID, func() error {
		current, err := store.Load(job.ID)
		if err != nil {
			return err
		}
		if jobs.IsTerminalStatus(current.Status) && current.Status != targetStatus {
			return nil
		}
		prior, exists, err := readStoredResult(current.ResultPath)
		if err != nil {
			return err
		}
		if exists && isTerminalResultStatus(prior.Status) && prior.Status != res.Status {
			current.Status = JobStatusFromResult(prior.Status)
			_, err := store.SaveGuarded(current)
			return err
		}
		if err := WriteJobResult(current.ResultPath, res); err != nil {
			return err
		}
		if res.Metadata.AgentSession != "" {
			current.Spec.Resume = res.Metadata.AgentSession
			current.AgentSession = res.Metadata.AgentSession
		}
		current.CapacityBackoffUntil = nil
		current.Status = targetStatus
		_, err = store.SaveGuarded(current)
		return err
	})
	removeErr := store.RemovePID(job.ID)
	if err != nil {
		return err
	}
	return removeErr
}

// WriteJobResult is the canonical atomic result-file writer used by child
// completion and cancellation.
func WriteJobResult(path string, res result.Result) error {
	encoded, err := result.FormatJSON(res)
	if err != nil {
		return err
	}
	return jobs.AtomicWriteFile(path, append(encoded, '\n'), 0o644)
}

func readStoredResult(path string) (result.Result, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return result.Result{}, false, nil
		}
		return result.Result{}, false, err
	}
	var stored result.Result
	if err := json.Unmarshal(content, &stored); err != nil {
		return result.Result{}, true, fmt.Errorf("decode async job result: %w", err)
	}
	return stored, true, nil
}

func isTerminalResultStatus(status result.Status) bool {
	for _, persisted := range []jobs.Status{
		jobs.StatusCapacity,
		jobs.StatusOutputLimit,
		jobs.StatusComplete,
		jobs.StatusFailed,
		jobs.StatusCancelled,
	} {
		if ResultStatusFromJob(string(persisted)) == status && jobs.IsTerminalStatus(persisted) {
			return true
		}
	}
	return false
}

func terminalJobResult(req JobRequest, job jobs.Job) (result.Result, error) {
	stored, exists, err := readStoredResult(job.ResultPath)
	if err != nil {
		return result.Result{}, err
	}
	if exists {
		return stored, nil
	}
	return jobStatusResult(req, job), nil
}
