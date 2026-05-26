package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/nklisch/codex-implement/internal/codex"
	"github.com/nklisch/codex-implement/internal/input"
	"github.com/nklisch/codex-implement/internal/prompt"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	req, err := input.Parse(args, os.Stdin, os.Getwd)
	if err != nil {
		return writeResult(result{
			Status:   "failed",
			Summary:  err.Error(),
			CWD:      "",
			ExitCode: 2,
		})
	}
	if req.Worktree {
		if err := writeResult(result{
			Status:   "failed",
			Summary:  "worktree mode is recognized but not implemented yet",
			CWD:      req.CWD,
			Access:   accessMode(req),
			Profile:  req.Profile,
			ExitCode: 2,
		}); err != nil {
			return err
		}
		os.Exit(1)
	}

	codexPrompt := prompt.Build(req.TaskText)
	execResult, execErr := codex.Exec(context.Background(), codex.Options{
		CWD:        req.CWD,
		Prompt:     codexPrompt,
		FullAccess: req.FullAccess,
		Profile:    req.Profile,
	})

	status := "success"
	summary := "Codex implementation completed"
	if execErr != nil {
		status = "failed"
		summary = execErr.Error()
	} else if execResult.ExitCode != 0 {
		status = "failed"
		summary = "Codex exited with non-zero status"
	}

	if err := writeResult(result{
		Status:   status,
		Summary:  summary,
		CWD:      req.CWD,
		Access:   accessMode(req),
		Profile:  req.Profile,
		ExitCode: execResult.ExitCode,
		Stdout:   execResult.Stdout,
		Stderr:   execResult.Stderr,
	}); err != nil {
		return err
	}
	if status != "success" {
		os.Exit(1)
	}
	return nil
}

type result struct {
	Status   string `json:"status"`
	Summary  string `json:"summary"`
	CWD      string `json:"cwd,omitempty"`
	Access   string `json:"access,omitempty"`
	Profile  string `json:"profile,omitempty"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
}

func accessMode(req input.Request) string {
	switch {
	case req.Worktree:
		return "worktree"
	case req.FullAccess:
		return "full-access"
	default:
		return "default"
	}
}

func writeResult(res result) error {
	encoded, err := json.Marshal(res)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(encoded))
	return err
}
