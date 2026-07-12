// Package testsupport contains focused doubles shared by adapter tests.
package testsupport

import (
	"context"

	"github.com/nklisch/peeragent/internal/executil"
)

// RecordingRunner captures one adapter invocation without executing a target CLI.
// Its exported fields let each target package keep its own argv assertions while
// sharing the recording behavior.
type RecordingRunner struct {
	Name   string
	Args   []string
	CWD    string
	Result executil.Result
}

var _ executil.Runner = (*RecordingRunner)(nil)

func (r *RecordingRunner) Run(_ context.Context, name string, args []string, cwd string) (executil.Result, error) {
	r.Name = name
	r.Args = args
	r.CWD = cwd
	return r.Result, nil
}
