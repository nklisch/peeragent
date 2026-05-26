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

	codexPrompt := prompt.Build(req.TaskText)
	execResult, execErr := codex.Exec(context.Background(), req.CWD, codexPrompt)

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
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
}

func writeResult(res result) error {
	encoded, err := json.Marshal(res)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(encoded))
	return err
}
