package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/nklisch/peeragent/internal/executil"
	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

// LauncherChannel is kept out of public job/result metadata. File mode 0600
// protects the terminal address from other users, not from the trusted owner.
type LauncherChannel struct {
	Pane      string `json:"pane,omitempty"`
	Socket    string `json:"socket,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	Binary    string `json:"binary,omitempty"`
}

func recordLauncher(job jobs.Job) error {
	channel := LauncherChannel{Pane: os.Getenv("HERDR_PANE_ID"), Socket: os.Getenv("HERDR_SOCKET_PATH"), Workspace: os.Getenv("HERDR_WORKSPACE_ID")}
	if channel.Pane == "" {
		return nil
	}
	channel.Binary = os.Getenv("HERDR_BIN_PATH")
	if channel.Binary == "" {
		channel.Binary, _ = exec.LookPath("herdr")
	}
	if channel.Binary != "" {
		var err error
		channel.Binary, err = filepath.Abs(channel.Binary)
		if err != nil {
			return err
		}
	}
	data, err := json.Marshal(channel)
	if err != nil {
		return err
	}
	return jobs.AtomicWriteFile(filepath.Join(filepath.Dir(job.LogPath), "launcher.json"), data, 0o600)
}

type TellReceipt struct {
	JobID  string
	Agent  string
	Status jobs.Status
	jobs.MessageStats
	MessagingSource  string
	MessagingEnabled bool
	Warning          string
}

func (s *Service) Tell(ctx context.Context, raw JobRequest, from, text string) error {
	_, err := s.TellWithReceipt(ctx, raw, from, text)
	return err
}

func (s *Service) TellWithReceipt(ctx context.Context, raw JobRequest, from, text string) (receipt TellReceipt, err error) {
	req, err := s.normalizeJobRequest(ctx, raw)
	if err != nil {
		return receipt, err
	}
	if strings.TrimSpace(text) == "" {
		return receipt, errors.New("tell requires message text")
	}
	if from == "" {
		from = "launcher"
	}
	store := jobs.NewStore(req.CWD)
	err = store.WithJobLock(req.JobID, func() error {
		job, err := store.Load(req.JobID)
		if err != nil {
			return fmt.Errorf("load job %s: %w", req.JobID, err)
		}
		if jobs.IsTerminalStatus(job.Status) {
			return fmt.Errorf("job %s is finished; use --resume %s to continue", job.ID, resumeHint(job.Session()))
		}
		mailbox, err := jobs.MailboxSnapshotLocked(ctx, store, job.ID)
		if err != nil {
			return err
		}
		support, err := store.MessagingSupport(job, mailbox)
		if err != nil {
			return err
		}
		if !support.Enabled && strings.TrimSpace(job.Session()) == "" {
			return fmt.Errorf("message not queued: job %s has no messaging inbox and no agent session to --resume; wait for its session to be recorded and try again", job.ID)
		}
		now := time.Now().UTC()
		if err := store.AppendMessageLocked(job.ID, "inbox", jobs.Message{Time: now, From: from, Text: text}); err != nil {
			return err
		}
		stats := mailbox.StatsAt(now)
		if stats.InboxUnread == 0 || (stats.OldestUnreadAt != nil && now.Before(*stats.OldestUnreadAt)) {
			stats.OldestUnreadAt = &now
			stats.OldestUnreadAge = "0s"
		}
		stats.InboxUnread++
		if stats.LastMessageAt == nil || now.After(*stats.LastMessageAt) {
			stats.LastMessageAt = &now
		}
		receipt = TellReceipt{JobID: job.ID, Agent: job.Spec.Agent, Status: job.Status, MessageStats: stats, MessagingEnabled: support.Enabled, MessagingSource: support.Source, Warning: support.Warning}
		return nil
	})
	return receipt, err
}

func resumeHint(session string) string {
	if session == "" {
		return "<agent-session>"
	}
	return session
}

// labelPart cannot inject a header, a newline, or terminal escape sequences.
func labelPart(text string) string {
	return strings.Map(func(r rune) rune {
		if directionFormatting(r) {
			return -1
		}
		if bracketReplacement(r) != r || unicode.IsControl(r) || (unicode.IsSpace(r) && r != ' ') {
			return '_'
		}
		return r
	}, text)
}

func messageHeader(job jobs.Job) string {
	branch := job.Branch
	if branch == "" {
		branch = "unknown"
	}
	return fmt.Sprintf("⟦peer · %s job %s · %s@%s · reply: peeragent tell %s \"…\"⟧ ", labelPart(job.Spec.Agent), job.ID, labelPart(filepath.Base(job.CWD)), labelPart(branch), job.ID)
}

func directionFormatting(r rune) bool {
	return r == '\u200e' || r == '\u200f' || (r >= '\u202a' && r <= '\u202e') || (r >= '\u2066' && r <= '\u2069')
}

func bracketReplacement(r rune) rune {
	switch r {
	case '⟦', '〚', '⟬':
		return '['
	case '⟧', '〛', '⟭':
		return ']'
	}
	return r
}

func messageBody(text string) string {
	return strings.Map(func(r rune) rune {
		if directionFormatting(r) {
			return -1
		}
		if unicode.IsControl(r) || (unicode.IsSpace(r) && r != ' ') {
			return ' '
		}
		return bracketReplacement(r)
	}, text)
}

// Say records a pending reservation before calling Herdr outside the lock.
// Pending counts against the bound too: a killed sender may have delivered.
func (s *Service) Say(ctx context.Context, raw JobRequest, text string) error {
	req, err := s.normalizeJobRequest(ctx, raw)
	if err != nil {
		return err
	}
	store := jobs.NewStore(req.CWD)
	var job jobs.Job
	var channel LauncherChannel
	entry := jobs.Message{Time: time.Now().UTC(), Text: messageBody(text), Status: "pending"}
	refusal := ""
	err = store.WithJobLock(req.JobID, func() error {
		var err error
		job, err = store.Load(req.JobID)
		if err != nil {
			return fmt.Errorf("load job %s: %w", req.JobID, err)
		}
		entry.From = job.Spec.Agent
		mailbox, err := jobs.MailboxSnapshotLocked(ctx, store, job.ID)
		if err != nil {
			return err
		}
		entry.ID = fmt.Sprintf("%s-%d", job.ID, len(mailbox.Outbox)+1)
		switch {
		case runtime.GOOS == "windows":
			refusal = "no launcher channel on this platform"
		case strings.TrimSpace(text) == "":
			refusal = "say requires message text"
		case utf8.RuneCountInString(text) > 2000:
			refusal = "message exceeds 2,000 characters"
		default:
			recent := 0
			for _, prior := range mailbox.Outbox {
				if prior.DeliveryAttempted && prior.Time.After(entry.Time.Add(-5*time.Minute)) {
					recent++
				}
			}
			if recent >= 3 {
				refusal = "rate limit: at most 3 messages per 5 minutes per job"
				break
			}
			data, err := os.ReadFile(filepath.Join(filepath.Dir(job.LogPath), "launcher.json"))
			if os.IsNotExist(err) {
				refusal = "no recorded launcher pane"
				break
			}
			if err != nil {
				refusal = "cannot read launcher channel"
				break
			}
			if err := json.Unmarshal(data, &channel); err != nil {
				refusal = "cannot decode launcher channel"
				break
			}
			if channel.Pane == "" || channel.Binary == "" {
				refusal = "no recorded launcher pane or Herdr binary"
			}
		}
		if refusal != "" {
			entry.Status = "refused"
			entry.Reason = refusal
		} else {
			entry.DeliveryAttempted = true
		}
		return store.AppendMessageLocked(job.ID, "outbox", entry)
	})
	if err != nil {
		return fmt.Errorf("say: %w; use your handback instead", err)
	}
	if refusal != "" {
		return fmt.Errorf("say: %s; use your handback instead", refusal)
	}
	deliveryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(deliveryCtx, channel.Binary, "agent", "prompt", channel.Pane, messageHeader(job)+entry.Text)
	// Only this explicit, labelled transport receives the recorded address.
	cmd.Env = append(os.Environ(), "HERDR_PANE_ID="+channel.Pane, "HERDR_SOCKET_PATH="+channel.Socket, "HERDR_WORKSPACE_ID="+channel.Workspace)
	if err := cmd.Run(); err != nil {
		entry.Status = "refused"
		entry.Reason = "Herdr refused or could not deliver the message"
	} else {
		entry.Status = "sent"
	}
	err = store.WithJobLock(job.ID, func() error { return store.AppendMessageLocked(job.ID, "outbox", entry) })
	if err != nil {
		return fmt.Errorf("record say outcome: %w; use your handback instead", err)
	}
	if entry.Status == "refused" {
		return fmt.Errorf("say: %s; use your handback instead", entry.Reason)
	}
	return nil
}

// TurnInbox reserves replies for one attempt so prompt delivery and an inbox
// command cannot deliver the same entry. Its durable outcome lets readers
// commit acceptance or recover unread replies after a worker crash.
type TurnInbox struct {
	Text    string
	batches []jobs.InboxSource
	owner   jobs.InboxSource
	attempt jobs.InboxAttempt
}

func PrepareTurnInbox(ctx context.Context, job jobs.Job) (carry TurnInbox, prepareErr error) {
	attempt, err := jobs.BeginInboxAttempt(job)
	if err != nil {
		return carry, err
	}
	carry = TurnInbox{owner: jobs.InboxSource{JobID: job.ID, CWD: job.CWD}, attempt: attempt}
	defer func() {
		if prepareErr != nil {
			prepareErr = errors.Join(prepareErr, carry.Acknowledge(executil.Result{}, prepareErr))
		}
	}()
	sources := append([]jobs.InboxSource{}, job.InboxSources...)
	sources = append(sources, jobs.InboxSource{JobID: job.ID, CWD: job.CWD})
	type incoming struct {
		id    string
		entry jobs.Message
	}
	var entries []incoming
	seen := map[string]bool{}
	for _, source := range sources {
		if seen[source.JobID] {
			continue
		}
		seen[source.JobID] = true
		carry.batches = append(carry.batches, source)
		messages, err := jobs.ReserveInbox(ctx, jobs.NewStore(source.CWD), source.JobID, attempt)
		if err != nil {
			return carry, fmt.Errorf("reserve resume inbox for %s: %w", source.JobID, err)
		}
		for _, entry := range messages {
			entries = append(entries, incoming{source.JobID, entry})
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].entry.Time.Before(entries[j].entry.Time) })
	var text strings.Builder
	if len(entries) > 0 {
		text.WriteString("\n\nLauncher messages sent while the job ran (reserved for this attempt):\n")
	}
	for _, item := range entries {
		fmt.Fprintf(&text, "[%s · %s · job %s] %s\n", item.entry.Time.Format(time.RFC3339Nano), item.entry.From, item.id, item.entry.Text)
	}
	if len(entries) > 0 {
		text.WriteString("\n")
	}
	carry.Text = text.String()
	return carry, nil
}

func (c TurnInbox) Acknowledge(execution executil.Result, executionErr error) error {
	if c.attempt.ID == "" {
		return nil
	}
	accepted := executionErr == nil && execution.TurnStarted && !execution.CapacityExhausted && !execution.AuthRequired && !execution.Interrupted
	var ownerErr error
	if accepted {
		var owner jobs.Job
		owner, ownerErr = jobs.NewStore(c.owner.CWD).Load(c.owner.JobID)
		accepted = ownerErr == nil && owner.Status != jobs.StatusCancelled
	}
	outcome, err := jobs.FinishInboxAttempt(c.attempt, accepted)
	if err != nil {
		return errors.Join(ownerErr, err)
	}
	var settleErr error
	for _, source := range c.batches {
		settleErr = errors.Join(settleErr, jobs.NewStore(source.CWD).SettleInboxAttempt(source.JobID, c.attempt.ID, outcome))
	}
	return errors.Join(ownerErr, settleErr)
}

func addMessaging(ctx context.Context, res *result.Result, job jobs.Job, conversation bool) error {
	m, err := jobs.MailboxSnapshot(ctx, jobs.NewStore(job.CWD), job.ID)
	if err != nil {
		return fmt.Errorf("read job %s mailbox: %w", job.ID, err)
	}
	stats := m.Stats()
	res.Metadata.MessageStats = &stats
	support, err := jobs.NewStore(job.CWD).MessagingSupport(job, m)
	if err != nil {
		return err
	}
	res.Metadata.MessagingEnabled = support.Enabled
	res.Metadata.MessagingSource = support.Source
	res.Metadata.MessagingWarning = support.Warning
	if conversation {
		res.Conversation = m.Conversation()
	}
	return nil
}
