package executil

import (
	"context"
	"strings"
	"unicode"
)

// Progress carries target observations without coupling adapters to job storage.
type Progress struct {
	Event         string
	LastMessage   string
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

// MessagePreview bounds work even for very large reports and produces one line.
func MessagePreview(message string) string {
	const limit = 160
	var out strings.Builder
	count := 0
	space := false
	for _, r := range message {
		if unicode.IsSpace(r) {
			space = out.Len() > 0
			continue
		}
		if count >= limit {
			out.WriteRune('…')
			break
		}
		if space {
			out.WriteByte(' ')
			count++
			space = false
			if count >= limit {
				out.WriteRune('…')
				break
			}
		}
		out.WriteRune(r)
		count++
	}
	return out.String()
}
