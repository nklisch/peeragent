package executil

import "context"

// Progress carries target observations without coupling adapters to job storage.
type Progress struct {
	AgentSession  string
	Message       string
	FinalReceived bool
}

type progressKey struct{}

func WithProgress(ctx context.Context, report func(Progress) error) context.Context {
	return context.WithValue(ctx, progressKey{}, report)
}

func ReportProgress(ctx context.Context, progress Progress) error {
	if report, ok := ctx.Value(progressKey{}).(func(Progress) error); ok {
		return report(progress)
	}
	return nil
}
