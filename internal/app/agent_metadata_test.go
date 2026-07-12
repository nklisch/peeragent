package app

import (
	"testing"

	"github.com/nklisch/peeragent/internal/agent"
	"github.com/nklisch/peeragent/internal/input"
)

func TestApplicationAgentMetadataMatchesRegistry(t *testing.T) {
	for _, definition := range agent.Definitions() {
		definition := definition
		t.Run(string(definition.ID), func(t *testing.T) {
			delegation := input.Delegation{Agent: string(definition.ID)}
			if got := agentPromptName(delegation); got != definition.PromptIdentity {
				t.Fatalf("agentPromptName(%q) = %q, want %q", definition.ID, got, definition.PromptIdentity)
			}
			if got := agentDisplayName(delegation); got != definition.DisplayName {
				t.Fatalf("agentDisplayName(%q) = %q, want %q", definition.ID, got, definition.DisplayName)
			}
		})
	}
}

func TestApplicationAgentMetadataDefaultsUnknownAndEmptyToCodex(t *testing.T) {
	definition, ok := agent.Lookup(agent.DefaultID())
	if !ok {
		t.Fatal("default target is not registered")
	}
	for _, delegation := range []input.Delegation{{}, {Agent: "unknown"}} {
		if got := agentPromptName(delegation); got != definition.PromptIdentity {
			t.Fatalf("agentPromptName(%q) = %q, want %q", delegation.Agent, got, definition.PromptIdentity)
		}
		if got := agentDisplayName(delegation); got != definition.DisplayName {
			t.Fatalf("agentDisplayName(%q) = %q, want %q", delegation.Agent, got, definition.DisplayName)
		}
	}
}
