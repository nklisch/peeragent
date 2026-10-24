package input

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nklisch/peeragent/internal/jobs"
)

func parseMessage(args []string, getwd func() (string, error)) (Request, error) {
	req := Request{MessageCommand: args[0]}
	args = args[1:]
	if req.MessageCommand == "tell" {
		if len(args) == 0 {
			return req, fmt.Errorf("tell requires a job id and message text")
		}
		req.MessageJobID = args[0]
		args = args[1:]
		if err := jobs.ValidateID(req.MessageJobID); err != nil {
			return req, err
		}
	}
	var text []string
	promptFile := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			text = append(text, args[i+1:]...)
			break
		}
		if (req.MessageCommand == "tell" && args[i] == "--cwd") || (req.MessageCommand != "inbox" && args[i] == "--prompt-file") {
			option := args[i]
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" || strings.HasPrefix(args[i+1], "-") {
				return req, fmt.Errorf("%s requires a non-empty value", option)
			}
			i++
			if option == "--cwd" {
				req.CWD = args[i]
			} else {
				promptFile = args[i]
			}
			continue
		}
		if strings.HasPrefix(args[i], "-") {
			return req, fmt.Errorf("unknown messaging option %q; use -- before literal text beginning with a dash", args[i])
		}
		text = append(text, args[i])
	}
	req.TaskText = strings.Join(text, " ")
	if promptFile != "" {
		content, err := os.ReadFile(promptFile)
		if err != nil {
			return req, fmt.Errorf("read prompt file: %w", err)
		}
		parts := []string{}
		for _, part := range []string{req.TaskText, string(content)} {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				parts = append(parts, trimmed)
			}
		}
		req.TaskText = strings.Join(parts, "\n\n")
	}
	if req.MessageCommand == "inbox" {
		if len(text) != 0 {
			return req, fmt.Errorf("inbox takes no arguments")
		}
	} else if req.MessageCommand == "tell" && strings.TrimSpace(req.TaskText) == "" {
		return req, fmt.Errorf("%s requires message text", req.MessageCommand)
	}
	if req.MessageCommand == "tell" {
		if req.CWD == "" {
			cwd, err := jobs.LookupJob(req.MessageJobID)
			if err != nil {
				return req, fmt.Errorf("lookup working directory (use --cwd to override): %w", err)
			}
			req.CWD = cwd
		}
		if req.CWD == "" {
			cwd, err := getwd()
			if err != nil {
				return req, err
			}
			req.CWD = cwd
		}
		cwd, err := filepath.Abs(req.CWD)
		if err != nil {
			return req, err
		}
		req.CWD = cwd
	}
	return req, nil
}
