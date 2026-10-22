package executil

import (
	"context"
	"strings"
)

type jobContext struct{ ID, Dir string }
type jobContextKey struct{}

func WithJobContext(ctx context.Context, id, dir string) context.Context {
	return context.WithValue(ctx, jobContextKey{}, jobContext{id, dir})
}

func jobEnvironment(ctx context.Context, env []string) []string {
	job, ok := ctx.Value(jobContextKey{}).(jobContext)
	if !ok {
		return env
	}
	// Replace inherited parent-job identity for nested delegations.
	filtered := make([]string, 0, len(env)+2)
	for _, item := range env {
		key, _, _ := strings.Cut(item, "=")
		if strings.EqualFold(key, "PEERAGENT_JOB_ID") || strings.EqualFold(key, "PEERAGENT_JOB_DIR") {
			continue
		}
		filtered = append(filtered, item)
	}
	return append(filtered, "PEERAGENT_JOB_ID="+job.ID, "PEERAGENT_JOB_DIR="+job.Dir)
}
