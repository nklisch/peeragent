// Package mcp adapts peeragent application services to the Model Context
// Protocol stdio transport. It owns protocol concerns only; target execution,
// job persistence, and result semantics remain in internal/app.
package mcp

import (
	"context"
	"log/slog"
	"os"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/result"
)

type DelegationService interface {
	Delegate(context.Context, input.Delegation) (result.Result, error)
	Launch(context.Context, input.Delegation) (result.Result, error)
}

const serverInstructions = `peeragent delegates focused implementation, research, review, debugging, and documentation tasks to a local coding agent.

Use delegate with async=true for substantive work that may exceed the host MCP tool timeout. Use blocking calls only for short passes. Choose an agent explicitly when needed; supported agents are codex, claude, gemini, and zai, with codex as the default. full_access is an explicit opt-in that disables the target's bounded mode. cwd defaults to the peeragent server's working directory; job ids are repository-local, so provide cwd only when the caller explicitly requested another repository.

For asynchronous work, use delegate(async=true), poll with job_status, and retrieve the full structured result with job_result. Use job_cancel only after explicit user intent; it is destructive because it marks cancellation and terminates the detached process group. Status and result are read-only.`

func NewServer(service ServerService, getwd func() (string, error)) *sdkmcp.Server {
	if getwd == nil {
		getwd = os.Getwd
	}

	server := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "peeragent", Version: "dev"},
		&sdkmcp.ServerOptions{
			Instructions: serverInstructions,
			// Stdio stdout is reserved for MCP frames. Keep SDK diagnostics on
			// stderr so a client never receives a non-protocol line.
			Logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
		},
	)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "delegate",
		Description: "Delegate one focused task to a local coding agent, blocking or launching a tracked async job.",
		Annotations: &sdkmcp.ToolAnnotations{DestructiveHint: boolPointer(true)},
	}, delegateHandler(service, getwd))
	registerJobTools(server, service)
	return server
}

func RunStdio(ctx context.Context, service ServerService) error {
	if ctx == nil {
		ctx = context.Background()
	}
	server := NewServer(service, os.Getwd)
	return server.Run(ctx, &sdkmcp.StdioTransport{})
}
