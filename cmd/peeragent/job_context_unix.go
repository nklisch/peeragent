//go:build unix

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func asyncJobContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
