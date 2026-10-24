package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"
)

// Claude Code's local inbox consumes newline-delimited auth and user frames.
// It queues user frames separately from the terminal editor. There is no
// application acknowledgement: a successful write is submission, not proof
// that a model has read or accepted the message. Never fall back to pane input.
func deliverClaudeInbox(ctx context.Context, channel LauncherChannel, text string) error {
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", channel.ClaudeSocket)
	if err != nil {
		return fmt.Errorf("connect to Claude Code inbox: %w", err)
	}
	defer conn.Close()
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(10 * time.Second)
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(map[string]string{"type": "auth", "token": channel.ClaudeToken}); err != nil {
		return fmt.Errorf("write inbox authentication: %w", err)
	}
	frame := struct {
		SessionID string `json:"session_id,omitempty"`
		Type      string `json:"type"`
		From      string `json:"from"`
		Message   struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	}{Type: "user", From: "peeragent", SessionID: channel.ClaudeSession}
	frame.Message.Role, frame.Message.Content = "user", text
	if err := encoder.Encode(frame); err != nil {
		return fmt.Errorf("write inbox message: %w", err)
	}
	if err := conn.(*net.UnixConn).CloseWrite(); err != nil {
		return err
	}
	// The server closes its write half after consuming the input stream.
	_, err = io.Copy(io.Discard, conn)
	return err
}
