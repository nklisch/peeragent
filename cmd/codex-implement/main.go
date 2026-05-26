package main

import (
	"fmt"
	"os"

	"github.com/nklisch/codex-implement/internal/input"
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

	_, err = fmt.Fprintf(os.Stdout, "{\"status\":\"blocked\",\"summary\":\"codex-implement wrapper is not implemented yet\",\"cwd\":%q,\"task_text\":%q}\n", req.CWD, req.TaskText)
	return err
}
