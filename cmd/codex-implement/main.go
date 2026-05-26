package main

import (
	"fmt"
	"os"

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
		return err
	}

	codexPrompt := prompt.Build(req.TaskText)

	_, err = fmt.Fprintf(os.Stdout, "{\"status\":\"blocked\",\"summary\":\"codex-implement wrapper is not implemented yet\",\"cwd\":%q,\"prompt\":%q}\n", req.CWD, codexPrompt)
	return err
}
