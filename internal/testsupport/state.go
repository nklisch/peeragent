package testsupport

import (
	"fmt"
	"os"
)

// RunWithPrivateState prevents test delegations from writing fake jobs or
// sessions into the user's lookup index. Helper subprocesses inherit this
// directory so they share their parent's job controls without creating orphans
// when a process-lifecycle test intentionally terminates a helper.
func RunWithPrivateState(run func() int) int {
	if os.Getenv("PEERAGENT_TEST_PRIVATE_STATE") == "1" {
		return run()
	}
	root, err := os.MkdirTemp("", "peeragent-test-state-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(root)
	if err := os.Setenv("PEERAGENT_STATE_DIR", root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := os.Setenv("PEERAGENT_TEST_PRIVATE_STATE", "1"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return run()
}
