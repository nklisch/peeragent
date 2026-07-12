// Package agent owns the canonical identity and user-facing metadata for each
// supported delegation target. The canonical ID is also the application
// dispatch key; target-specific CLI calls remain in internal/app.
package agent

import "strings"

// ID is the canonical target identifier persisted in requests and job specs.
type ID string

const (
	CodexID  ID = "codex"
	GeminiID ID = "gemini"
	ClaudeID ID = "claude"
	ZAIID    ID = "zai"
)

// Definition is the complete shared metadata for one target. Aliases are
// normalized case-insensitively after surrounding whitespace is removed.
type Definition struct {
	ID             ID
	Aliases        []string
	PromptIdentity string
	DisplayName    string
}

// registry is deliberately the only target metadata table. Keep canonical IDs
// in Aliases too: this makes the accepted input vocabulary explicit and lets
// normalization and metadata parity tests exercise the same source of truth.
var registry = [...]Definition{
	{
		ID:             CodexID,
		Aliases:        []string{"", "codex"},
		PromptIdentity: "Codex",
		DisplayName:    "Codex",
	},
	{
		ID:             GeminiID,
		Aliases:        []string{"gemini", "agy", "antigravity"},
		PromptIdentity: "Gemini through Antigravity CLI",
		DisplayName:    "Gemini",
	},
	{
		ID:             ClaudeID,
		Aliases:        []string{"claude"},
		PromptIdentity: "Claude Code",
		DisplayName:    "Claude",
	},
	{
		ID:             ZAIID,
		Aliases:        []string{"zai", "z.ai", "glm", "glm-5.2", "glm5.2", "pi-zai", "pi-glm"},
		PromptIdentity: "Z.AI GLM 5.2 through Pi",
		DisplayName:    "Z.AI GLM 5.2",
	},
}

// DefaultID returns the default target without duplicating the default in a
// second validation switch. Registry order is fixed and intentionally starts
// with Codex, matching the historical CLI default.
func DefaultID() ID {
	return registry[0].ID
}

// Lookup finds metadata by canonical ID. Aliases belong at the input boundary;
// application callers should only receive canonical IDs after normalization.
func Lookup(id ID) (Definition, bool) {
	for _, definition := range registry {
		if definition.ID == id {
			return clone(definition), true
		}
	}
	return Definition{}, false
}

// Normalize resolves an input spelling to a canonical target ID.
func Normalize(raw string) (ID, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	for _, definition := range registry {
		for _, alias := range definition.Aliases {
			if value == alias {
				return definition.ID, true
			}
		}
	}
	return "", false
}

// Definitions returns a defensive copy for parity checks and presentation
// tooling without exposing the registry's mutable alias slices.
func Definitions() []Definition {
	definitions := make([]Definition, 0, len(registry))
	for _, definition := range registry {
		definitions = append(definitions, clone(definition))
	}
	return definitions
}

func clone(definition Definition) Definition {
	definition.Aliases = append([]string(nil), definition.Aliases...)
	return definition
}
