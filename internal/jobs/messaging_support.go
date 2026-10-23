package jobs

import (
	"fmt"
	"os"
	"strings"

	"github.com/nklisch/peeragent/internal/prompt"
)

type MessagingSupport struct {
	Enabled bool
	Source  string
	Warning string
}

// MessagingSupport reuses the caller's mailbox snapshot, never taking a second
// job lock. Older writers may save lifecycle updates without knowing the flag;
// lack of that flag alone is not evidence of a missing messaging preamble.
func (s Store) MessagingSupport(job Job, mailbox Mailbox) (MessagingSupport, error) {
	if job.messagingRecorded {
		return MessagingSupport{Enabled: job.MessagingEnabled, Source: "recorded in job.json", Warning: job.MessagingWarning()}, nil
	}
	text, err := s.ReadPrompt(job.ID)
	if err == nil && strings.Contains(strings.ReplaceAll(text, "\r\n", "\n"), prompt.MessagingPreamble) {
		return MessagingSupport{Enabled: true, Source: "inferred from prompt"}, nil
	}
	if len(mailbox.Outbox) > 0 {
		return MessagingSupport{Enabled: true, Source: "inferred from outbox"}, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return MessagingSupport{}, fmt.Errorf("read job %s messaging evidence: %w", job.ID, err)
	}
	return MessagingSupport{Source: "no messaging evidence", Warning: job.MessagingWarning()}, nil
}
