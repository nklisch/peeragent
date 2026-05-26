package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	_ = args
	_, err := fmt.Fprintln(os.Stdout, `{"status":"blocked","summary":"codex-implement wrapper is not implemented yet"}`)
	return err
}
