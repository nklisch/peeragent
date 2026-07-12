package jobs

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

// ErrInvalidID identifies a job id that cannot be used as a repository-local
// job directory name. Callers can classify this as an input error without
// mistaking it for a missing job.
var ErrInvalidID = errors.New("invalid job id")

const jobIDTimestampLayout = "20060102T150405Z"

var jobIDPattern = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}Z-[0-9a-f]{8}$`)

// ValidateID is the authoritative validator for generated asynchronous job
// ids. The timestamp is checked as a real UTC time in addition to the exact
// ASCII shape, so a value can never become a path component merely because it
// resembles a generated id.
func ValidateID(id string) error {
	if !jobIDPattern.MatchString(id) {
		return fmt.Errorf("%w: want YYYYMMDDTHHMMSSZ-xxxxxxxx", ErrInvalidID)
	}
	if _, err := time.Parse(jobIDTimestampLayout, id[:16]); err != nil {
		return fmt.Errorf("%w: invalid timestamp", ErrInvalidID)
	}
	return nil
}
