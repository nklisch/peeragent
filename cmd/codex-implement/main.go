package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nklisch/codex-implement/internal/codex"
	"github.com/nklisch/codex-implement/internal/input"
	"github.com/nklisch/codex-implement/internal/prompt"
	"github.com/nklisch/codex-implement/internal/result"
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
		return writeResult(input.Request{JSON: true}, result.Result{
			Status:  result.StatusFailed,
			Summary: err.Error(),
			Metadata: result.Metadata{
				ExitCode: 2,
			},
		})
	}
	if req.Worktree {
		if err := writeResult(req, result.Result{
			Status:  result.StatusFailed,
			Summary: "worktree mode is recognized but not implemented yet",
			Metadata: result.Metadata{
				CWD:      req.CWD,
				Access:   accessMode(req),
				Profile:  req.Profile,
				ExitCode: 2,
			},
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

	status := result.StatusSuccess
	summary := "Codex implementation completed"
	if execErr != nil {
		status = result.StatusFailed
		summary = execErr.Error()
	} else if execResult.ExitCode != 0 {
		status = result.StatusFailed
		summary = "Codex exited with non-zero status"
	}

	if err := writeResult(req, result.Result{
		Status:       status,
		Summary:      summary,
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Details:      details(execResult.Stdout, execResult.Stderr),
		Metadata: result.Metadata{
			CWD:      req.CWD,
			Access:   accessMode(req),
			Profile:  req.Profile,
			ExitCode: execResult.ExitCode,
		},
	}); err != nil {
		return err
	}
	if status != result.StatusSuccess {
		os.Exit(1)
	}
	return nil
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

func writeResult(req input.Request, res result.Result) error {
	if req.JSON {
		encoded, err := result.FormatJSON(res)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(os.Stdout, string(encoded))
		return err
	}
	_, err := fmt.Fprintln(os.Stdout, result.FormatText(res))
	return err
}

func details(stdout string, stderr string) string {
	switch {
	case stdout != "" && stderr != "":
		return "stdout:\n" + stdout + "\n\nstderr:\n" + stderr
	case stdout != "":
		return "stdout:\n" + stdout
	case stderr != "":
		return "stderr:\n" + stderr
	default:
		return ""
	}
}
