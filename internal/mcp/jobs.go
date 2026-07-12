package mcp

import (
	"context"
	"fmt"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nklisch/peeragent/internal/app"
	"github.com/nklisch/peeragent/internal/result"
)

// JobService is the application boundary for repository-local async job
// control. MCP handlers do not read job files or signal processes directly.
type JobService interface {
	JobStatus(context.Context, app.JobRequest) (result.Result, error)
	JobResult(context.Context, app.JobRequest) (result.Result, error)
	CancelJob(context.Context, app.JobRequest) (result.Result, error)
}

// ServerService is the complete service surface required by the peeragent MCP
// server. Keeping both capabilities in one typed contract prevents a server
// instance from silently advertising an incomplete async workflow.
type ServerService interface {
	DelegationService
	JobService
}

type JobInput struct {
	JobID string `json:"job_id" jsonschema:"required peeragent async job id"`
	CWD   string `json:"cwd,omitempty" jsonschema:"repository directory; defaults to the server working directory"`
}

func registerJobTools(server *sdkmcp.Server, service JobService) {
	readOnly := false
	destructive := true
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "job_status",
		Description: "Inspect the compact status of one repository-local async peeragent job.",
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true, DestructiveHint: &readOnly},
	}, jobStatusHandler(service))
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "job_result",
		Description: "Retrieve the structured result of one repository-local async peeragent job.",
		Annotations: &sdkmcp.ToolAnnotations{ReadOnlyHint: true, DestructiveHint: &readOnly},
	}, jobResultHandler(service))
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "job_cancel",
		Description: "Cancel one async peeragent job after explicit user intent; cancellation terminates its detached process group.",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: &destructive,
			IdempotentHint:  true,
		},
	}, jobCancelHandler(service))
}

func jobStatusHandler(service JobService) sdkmcp.ToolHandlerFor[JobInput, result.Result] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, in JobInput) (*sdkmcp.CallToolResult, result.Result, error) {
		req, err := jobRequest(in)
		if err != nil {
			return nil, result.Result{}, err
		}
		if service == nil {
			return nil, result.Result{}, fmt.Errorf("job service is unavailable")
		}
		got, err := service.JobStatus(ctx, req)
		return nil, got, err
	}
}

func jobResultHandler(service JobService) sdkmcp.ToolHandlerFor[JobInput, result.Result] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, in JobInput) (*sdkmcp.CallToolResult, result.Result, error) {
		req, err := jobRequest(in)
		if err != nil {
			return nil, result.Result{}, err
		}
		if service == nil {
			return nil, result.Result{}, fmt.Errorf("job service is unavailable")
		}
		got, err := service.JobResult(ctx, req)
		return nil, got, err
	}
}

func jobCancelHandler(service JobService) sdkmcp.ToolHandlerFor[JobInput, result.Result] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, in JobInput) (*sdkmcp.CallToolResult, result.Result, error) {
		req, err := jobRequest(in)
		if err != nil {
			return nil, result.Result{}, err
		}
		if service == nil {
			return nil, result.Result{}, fmt.Errorf("job service is unavailable")
		}
		got, err := service.CancelJob(ctx, req)
		return nil, got, err
	}
}

func boolPointer(value bool) *bool {
	return &value
}

func jobRequest(in JobInput) (app.JobRequest, error) {
	jobID := strings.TrimSpace(in.JobID)
	if jobID == "" {
		return app.JobRequest{}, fmt.Errorf("invalid job request: job id is required")
	}
	return app.JobRequest{JobID: jobID, CWD: strings.TrimSpace(in.CWD)}, nil
}
