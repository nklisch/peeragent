package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

// JobRequest identifies a repository-local asynchronous job. CWD is optional
// at the application boundary; an empty value resolves through the service's
// working-directory port so CLI and MCP callers share the same lookup rules.
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

	return jobStatusResult(req, job), nil
}

// JobResult returns the persisted terminal result, or running while the child
// has not produced result.json yet. Result JSON is decoded here so MCP never
// has to read job files directly and corrupt storage remains distinguishable
// from an ordinary in-flight job.
func (s *Service) JobResult(ctx context.Context, raw JobRequest) (result.Result, error) {
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
			return runningJobResultForRequest(req, job), nil
		}
		return result.Result{}, fmt.Errorf("read async job %q result: %w", req.JobID, err)
	}

	var stored result.Result
	if err := json.Unmarshal(content, &stored); err != nil {
		return result.Result{}, fmt.Errorf("decode async job %q result: %w", req.JobID, err)
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
	return result.Result{
		Status:       ResultStatusFromJob(job.Status),
		Summary:      fmt.Sprintf("Async job %s is %s", job.ID, job.Status),
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata: result.Metadata{
			CWD:      req.CWD,
			ExitCode: 0,
			JobID:    job.ID,
		},
	}
}

func runningJobResultForRequest(req JobRequest, job jobs.Job) result.Result {
	return result.Result{
		Status:       result.StatusRunning,
		Summary:      fmt.Sprintf("Async job %s is still running", job.ID),
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata: result.Metadata{
			CWD:      req.CWD,
			ExitCode: 0,
			JobID:    job.ID,
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
	switch status {
	case "complete":
		return result.StatusSuccess
	case "failed":
		return result.StatusFailed
	case "cancelled":
		return result.StatusCancelled
	default:
		return result.StatusRunning
	}
}

// IsTerminalJobStatus is shared by child completion and cancellation so both
// paths agree on which persisted states can win a terminal transition.
func IsTerminalJobStatus(status string) bool {
	switch status {
	case "complete", "failed", "cancelled":
		return true
	default:
		return false
	}
}

// JobStatusFromResult maps an application result back to the persisted job
// lifecycle value used by job.json.
func JobStatusFromResult(status result.Status) string {
	switch status {
	case result.StatusSuccess:
		return "complete"
	case result.StatusCancelled:
		return "cancelled"
	default:
		return "failed"
	}
}

// FinishJob commits a natural child result using the same lock and terminal
// conflict rules as cancellation. It remains an application operation even
// though the child-run CLI is the adapter that invokes it.
func FinishJob(store jobs.Store, job jobs.Job, res result.Result) error {
	targetStatus := JobStatusFromResult(res.Status)
	err := store.WithJobLock(job.ID, func() error {
		current, err := store.Load(job.ID)
		if err != nil {
			return err
		}
		if IsTerminalJobStatus(current.Status) && current.Status != targetStatus {
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
	switch status {
	case result.StatusSuccess, result.StatusFailed, result.StatusCancelled:
		return true
	default:
		return false
	}
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
