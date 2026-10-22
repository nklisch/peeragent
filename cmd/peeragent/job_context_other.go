//go:build !unix

package main

import "context"

func asyncJobContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}
