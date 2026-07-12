package agent

import "testing"

func TestRegistryMetadataAndAliases(t *testing.T) {
	for _, definition := range Definitions() {
		definition := definition
		t.Run(string(definition.ID), func(t *testing.T) {
			if definition.ID == "" || definition.PromptIdentity == "" || definition.DisplayName == "" {
				t.Fatalf("incomplete definition: %#v", definition)
			}
			if _, ok := Lookup(definition.ID); !ok {
				t.Fatalf("Lookup(%q) did not find registered target", definition.ID)
			}
			for _, alias := range definition.Aliases {
				got, ok := Normalize(alias)
				if !ok || got != definition.ID {
					t.Fatalf("Normalize(%q) = %q, %v; want %q, true", alias, got, ok, definition.ID)
				}
				if alias != "" {
					got, ok = Normalize("  " + alias + " ")
					if !ok || got != definition.ID {
						t.Fatalf("trimmed Normalize(%q) = %q, %v; want %q, true", alias, got, ok, definition.ID)
					}
				}
			}
		})
	}
}

func TestNormalizeIsCaseInsensitiveAndRejectsUnknownTargets(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  ID
	}{
		{input: "CODEX", want: CodexID},
		{input: "AnTiGrAvItY", want: GeminiID},
		{input: "CLAUDE", want: ClaudeID},
		{input: "Z.AI", want: ZAIID},
	} {
		got, ok := Normalize(tt.input)
		if !ok || got != tt.want {
			t.Fatalf("Normalize(%q) = %q, %v; want %q, true", tt.input, got, ok, tt.want)
		}
	}
	for _, input := range []string{"llama", "codex-pro", "target"} {
		if got, ok := Normalize(input); ok || got != "" {
			t.Fatalf("Normalize(%q) = %q, %v; want empty, false", input, got, ok)
		}
	}
}

func TestDefaultIDComesFromCodexRegistryEntry(t *testing.T) {
	if got := DefaultID(); got != CodexID {
		t.Fatalf("DefaultID() = %q, want %q", got, CodexID)
	}
}
