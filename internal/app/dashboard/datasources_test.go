package dashboard

import (
	"strings"
	"testing"
)

func TestBuildEnvStatusReportsPresenceWithoutLeakingSecret(t *testing.T) {
	const secret = "sk-super-secret-value-do-not-leak"
	t.Setenv("ANTHROPIC_API_KEY", secret)
	t.Setenv("OPENAI_API_KEY", "")

	keys := buildEnvStatus()

	var anthropic, openai *struct {
		set    bool
		source string
	}
	for _, k := range keys {
		// The secret value must never appear in any surfaced field.
		if strings.Contains(k.Name, secret) || strings.Contains(k.Source, secret) {
			t.Fatalf("env status leaked secret value in %+v", k)
		}
		switch k.Name {
		case "ANTHROPIC_API_KEY":
			anthropic = &struct {
				set    bool
				source string
			}{k.Set, k.Source}
		case "OPENAI_API_KEY":
			openai = &struct {
				set    bool
				source string
			}{k.Set, k.Source}
		}
	}
	if anthropic == nil || !anthropic.set || anthropic.source != "env" {
		t.Fatalf("expected ANTHROPIC_API_KEY set from env, got %+v", anthropic)
	}
	if openai == nil || openai.set {
		t.Fatalf("expected OPENAI_API_KEY unset, got %+v", openai)
	}
}

func TestBuildConfigSummaryReportsPaths(t *testing.T) {
	entries := buildConfigSummary()
	keys := map[string]bool{}
	for _, e := range entries {
		keys[e.Key] = true
	}
	if !keys["config_path"] || !keys["gormes_home"] {
		t.Fatalf("config summary missing path facts: %+v", entries)
	}
}

func TestBuildSkillsListDoesNotPanic(t *testing.T) {
	// Filesystem-dependent; the contract is that it returns a (possibly empty)
	// slice without panicking regardless of what is installed.
	_ = buildSkillsList()
}
