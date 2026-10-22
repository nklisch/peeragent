package result

import (
	"bytes"
	"fmt"
	"text/tabwriter"
	"time"
)

// JobStatus is the compact multi-job status contract; single-job status keeps
// using Result for compatibility with existing hosts.
type JobStatus struct {
	JobID           string    `json:"job_id"`
	Status          Status    `json:"status"`
	Model           string    `json:"model"`
	Effort          string    `json:"effort"`
	AgentSession    string    `json:"agent_session"`
	StartedAt       time.Time `json:"started_at"`
	CapacityBackoff bool      `json:"capacity_backoff"`
	FinalReceived   bool      `json:"final_received"`
	HeadCommit      string    `json:"head_commit,omitempty"`
	CWD             string    `json:"cwd"`
}

func FormatJobStatuses(records []JobStatus) string {
	var out bytes.Buffer
	w := tabwriter.NewWriter(&out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "JOB\tSTATUS\tMODEL\tEFFORT\tAGENT_SESSION\tSTARTED\tCAPACITY_BACKOFF\tHEAD\tNOTE")
	for _, r := range records {
		note := ""
		if r.FinalReceived {
			note = "final message received, process still alive"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%t\t%s\t%s\n", r.JobID, r.Status, r.Model, r.Effort, r.AgentSession, r.StartedAt.UTC().Format(time.RFC3339), r.CapacityBackoff, r.HeadCommit, note)
	}
	_ = w.Flush()
	return out.String()
}
