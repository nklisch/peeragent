package mcp

import (
	"context"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nklisch/peeragent/internal/input"
	"github.com/nklisch/peeragent/internal/result"
)

type DelegateInput struct {
	Task       string `json:"task" jsonschema:"task for the peer agent"`
	Agent      string `json:"agent,omitempty" jsonschema:"codex, claude, gemini, or zai; defaults to codex"`
	CWD        string `json:"cwd,omitempty" jsonschema:"repository directory; defaults to the server working directory"`
	Profile    string `json:"profile,omitempty" jsonschema:"Codex profile override"`
	Effort     string `json:"effort,omitempty" jsonschema:"target reasoning effort"`
	Model      string `json:"model,omitempty" jsonschema:"target model override"`
	Resume     string `json:"resume,omitempty" jsonschema:"target agent session to resume"`
	FullAccess bool   `json:"full_access,omitempty" jsonschema:"explicitly disable the target sandbox"`
	Async      bool   `json:"async,omitempty" jsonschema:"launch a tracked job and return immediately"`
}

func delegateHandler(service DelegationService, getwd func() (string, error)) sdkmcp.ToolHandlerFor[DelegateInput, result.Result] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, in DelegateInput) (*sdkmcp.CallToolResult, result.Result, error) {
		delegation, err := input.NormalizeDelegation(input.Delegation{
			TaskText:   in.Task,
			CWD:        in.CWD,
			Agent:      in.Agent,
			FullAccess: in.FullAccess,
			Profile:    in.Profile,
			Effort:     in.Effort,
			Model:      in.Model,
			Resume:     in.Resume,
		}, getwd)
		if err != nil {
			return nil, result.Result{}, fmt.Errorf("invalid delegation: %w", err)
		}
		if service == nil {
			return nil, result.Result{}, fmt.Errorf("delegation service is unavailable")
		}

		if in.Async {
			delegated, err := service.Launch(ctx, delegation)
			return nil, delegated, err
		}
		delegated, err := service.Delegate(ctx, delegation)
		return nil, delegated, err
	}
}
