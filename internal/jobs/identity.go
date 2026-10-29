package jobs

import "os"

// ProcessIdentity distinguishes a worker from a later process reusing its PID.
// An unavailable probe is unknown, never evidence that a live worker died.
type ProcessIdentity struct {
	PID       int    `json:"pid"`
	BootID    string `json:"boot_id,omitempty"`
	StartTime string `json:"start_time,omitempty"`
}

func IdentifyProcess(pid int) ProcessIdentity {
	return ProcessIdentity{PID: pid, BootID: bootID(), StartTime: processStartTime(pid)}
}

func (p ProcessIdentity) Gone() bool {
	if p.BootID != "" {
		if boot := bootID(); boot != "" && boot != p.BootID {
			return true
		}
	}
	if p.PID <= 1 {
		return false
	}
	if !ProcessExists(p.PID) {
		return true
	}
	start := processStartTime(p.PID)
	return p.StartTime != "" && start != "" && start != p.StartTime
}

// WorkerGone leaves launch's short pre-PID window alone. Legacy records can
// prove death by PID absence, but cannot prove that a reused PID is their worker.
func (s Store) WorkerGone(job Job) (bool, error) {
	if job.Worker != nil {
		return job.Worker.Gone(), nil
	}
	pid, err := s.ReadPID(job.ID)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !ProcessExists(pid), nil
}
