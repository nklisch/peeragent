package result

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func FormatJSON(res Result) ([]byte, error) {
	return json.Marshal(res)
}

func FormatText(res Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Alt Subagent: %s\n\n", res.Status)
	fmt.Fprintf(&b, "Summary:\n%s\n\n", res.Summary)

	if len(res.ChangedFiles) > 0 {
		b.WriteString("Changed Files:\n")
		for _, file := range res.ChangedFiles {
			fmt.Fprintf(&b, "- %s\n", file)
		}
		b.WriteString("\n")
	}

	if len(res.Verification) > 0 {
		b.WriteString("Verification:\n")
		for _, check := range res.Verification {
			fmt.Fprintf(&b, "- %s: %s\n", check.Command, check.Status)
		}
		b.WriteString("\n")
	}

	if res.Details != "" {
		fmt.Fprintf(&b, "Details:\n%s\n\n", res.Details)
	}

	if len(res.Conversation) > 0 {
		b.WriteString("Conversation:\n")
		for _, entry := range res.Conversation {
			fmt.Fprintf(&b, "- %s %s from %s", entry.Time.UTC().Format(time.RFC3339Nano), entry.Direction, entry.From)
			if entry.Status != "" {
				fmt.Fprintf(&b, " (%s)", entry.Status)
			}
			fmt.Fprintf(&b, ": %s", entry.Text)
			if entry.Reason != "" {
				fmt.Fprintf(&b, " [%s]", entry.Reason)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Metadata:\n")
	if stats := res.Metadata.MessageStats; stats != nil {
		fmt.Fprintf(&b, "- inbox_unread: %d\n- outbox_sent: %d\n", stats.InboxUnread, stats.OutboxSent)
		if stats.LastMessageAt != nil {
			fmt.Fprintf(&b, "- last_message_at: %s\n", stats.LastMessageAt.UTC().Format(time.RFC3339Nano))
		}
	}
	if res.Metadata.KeepTerminalEnv {
		fmt.Fprintln(&b, "- keep_terminal_env: true")
	}
	if res.Metadata.JobID != "" {
		fmt.Fprintf(&b, "- job_id: %s\n", res.Metadata.JobID)
	}
	if res.Metadata.LastEventAt != nil {
		fmt.Fprintf(&b, "- last_event_at: %s\n", res.Metadata.LastEventAt.Format(time.RFC3339Nano))
	}
	if res.Metadata.LastMessage != "" {
		fmt.Fprintf(&b, "- last_message: %s\n", res.Metadata.LastMessage)
	}
	if res.Metadata.IndexWarning != "" {
		fmt.Fprintf(&b, "- index_warning: %s\n", res.Metadata.IndexWarning)
	}
	if res.Metadata.CapacityRetries > 0 {
		fmt.Fprintf(&b, "- capacity_retries: %d\n", res.Metadata.CapacityRetries)
	}
	if res.Metadata.CWD != "" {
		fmt.Fprintf(&b, "- cwd: %s\n", res.Metadata.CWD)
	}
	if res.Metadata.Agent != "" {
		fmt.Fprintf(&b, "- agent: %s\n", res.Metadata.Agent)
	}
	if res.Metadata.Access != "" {
		fmt.Fprintf(&b, "- access: %s\n", res.Metadata.Access)
	}
	if res.Metadata.Profile != "" {
		fmt.Fprintf(&b, "- profile: %s\n", res.Metadata.Profile)
	}
	if res.Metadata.Effort != "" {
		fmt.Fprintf(&b, "- effort: %s\n", res.Metadata.Effort)
	}
	if res.Metadata.Model != "" {
		fmt.Fprintf(&b, "- model: %s\n", res.Metadata.Model)
	}
	if res.Metadata.AgentSession != "" {
		fmt.Fprintf(&b, "- agent_session: %s\n", res.Metadata.AgentSession)
	}
	if res.Metadata.MessagePath != "" {
		fmt.Fprintf(&b, "- message_path: %s\n", res.Metadata.MessagePath)
	}
	if res.Metadata.FinalReceived {
		fmt.Fprintln(&b, "- final_received: true")
	}
	if res.Metadata.LogPath != "" {
		fmt.Fprintf(&b, "- log_path: %s\n", res.Metadata.LogPath)
	}
	fmt.Fprintf(&b, "- exit_code: %d\n", res.Metadata.ExitCode)

	return strings.TrimRight(b.String(), "\n")
}
