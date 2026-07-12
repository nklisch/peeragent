package input

import (
	"errors"
	"testing"
)

func TestNormalizeDelegationDefaultsAndCanonicalizes(t *testing.T) {
	got, err := NormalizeDelegation(Delegation{
		TaskText: "  do work  ",
		Agent:    "z.ai",
		Effort:   "XHIGH",
		Model:    "zai/glm-5.2",
		Profile:  " profile-a ",
		Resume:   " session-1 ",
	}, func() (string, error) { return "/repo", nil })
	if err != nil {
		t.Fatal(err)
	}
	if got.TaskText != "do work" || got.CWD != "/repo" || got.Agent != "zai" || got.Effort != "xhigh" || got.Model != "glm-5.2" {
		t.Fatalf("delegation = %#v", got)
	}
	if got.Profile != "profile-a" || got.Resume != "session-1" {
		t.Fatalf("trimmed fields = %#v", got)
	}
}

func TestNormalizeDelegationRejectsInvalidTargetOptions(t *testing.T) {
	tests := []Delegation{
		{TaskText: "do work", Agent: "unknown"},
		{TaskText: "do work", Agent: "claude", Effort: "medium"},
		{TaskText: "do work", Agent: "zai", Model: "glm-5.1"},
	}
	for _, raw := range tests {
		if _, err := NormalizeDelegation(raw, fixedCWD); err == nil {
			t.Fatalf("NormalizeDelegation(%#v) returned nil error", raw)
		}
	}
}

func TestNormalizeDelegationRejectsEmptyTaskAndCWDFailure(t *testing.T) {
	if _, err := NormalizeDelegation(Delegation{TaskText: "  "}, fixedCWD); err == nil {
		t.Fatal("expected empty task error")
	}
	cwdErr := errors.New("cwd unavailable")
	if _, err := NormalizeDelegation(Delegation{TaskText: "do work"}, func() (string, error) { return "", cwdErr }); !errors.Is(err, cwdErr) {
		t.Fatalf("cwd error = %v, want %v", err, cwdErr)
	}
}

func TestParseUsesCanonicalDelegationNormalization(t *testing.T) {
	req, err := Parse([]string{"--agent", "z.ai", "--effort", "xhigh", "--model", "glm-5.2", "do work"}, nil, fixedCWD)
	if err != nil {
		t.Fatal(err)
	}
	if req.Agent != "zai" || req.Effort != "xhigh" || req.Model != "glm-5.2" || req.CWD != "/repo" {
		t.Fatalf("request = %#v", req)
	}
}
