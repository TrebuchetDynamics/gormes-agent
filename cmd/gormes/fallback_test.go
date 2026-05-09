package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

func TestFallbackCommandListReadsConfiguredChain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	configPath := filepath.Join(home, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
[hermes]
provider = "openai-codex"
model = "gpt-5.5"

[[fallback_providers]]
provider = "anthropic"
model = "claude-opus-4-20250514"
base_url = "https://api.anthropic.example"

[[fallback_providers]]
provider = "openai"
model = "gpt-4o-mini"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}),
		"fallback", "list",
	)
	if err != nil {
		t.Fatalf("fallback list: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Primary:   gpt-5.5  (via openai-codex)",
		"Fallback chain (2 entries):",
		"1. claude-opus-4-20250514  (via anthropic)  [https://api.anthropic.example]",
		"2. gpt-4o-mini  (via openai)",
		"Tried in order when the primary fails",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("fallback list output missing %q:\n%s", want, stdout)
		}
	}
}

func TestFallbackCommandAddAppendsSelectionWithoutChangingPrimary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	configPath := filepath.Join(home, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
[hermes]
provider = "openai-codex"
model = "gpt-5.5"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	seams := (&modelCommandFakeSeams{
		isTTY:     true,
		current:   cli.ProviderModel{Provider: "openai-codex", Model: "gpt-5.5"},
		providers: []cli.ProviderMenuEntry{{ID: "anthropic", Label: "Anthropic"}},
		model:     "claude-opus-4-20250514",
	}).seams()
	stdout, stderr, err := runFallbackTestCommand(t, seams, "add")
	if err != nil {
		t.Fatalf("fallback add: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Added fallback: claude-opus-4-20250514  (via anthropic)") {
		t.Fatalf("stdout missing added fallback evidence:\n%s", stdout)
	}

	cfg, err := loadFallbackConfig(configPath)
	if err != nil {
		t.Fatalf("load fallback config: %v", err)
	}
	if cfg.Primary.Provider != "openai-codex" || cfg.Primary.Model != "gpt-5.5" {
		t.Fatalf("primary = %#v, want unchanged openai-codex/gpt-5.5", cfg.Primary)
	}
	if len(cfg.Chain) != 1 || cfg.Chain[0].Provider != "anthropic" || cfg.Chain[0].Model != "claude-opus-4-20250514" {
		t.Fatalf("fallback chain = %#v, want one anthropic fallback", cfg.Chain)
	}
	body, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(body), "fallback_model") {
		t.Fatalf("config retained legacy fallback_model after write:\n%s", body)
	}
}

func TestFallbackCommandAddSkipsExactDuplicate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	configPath := filepath.Join(home, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
[[fallback_providers]]
provider = "anthropic"
model = "claude-opus-4-20250514"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	seams := (&modelCommandFakeSeams{
		isTTY:     true,
		providers: []cli.ProviderMenuEntry{{ID: "anthropic", Label: "Anthropic"}},
		model:     "claude-opus-4-20250514",
	}).seams()
	stdout, stderr, err := runFallbackTestCommand(t, seams, "add")
	if err != nil {
		t.Fatalf("fallback add duplicate: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "is already in the fallback chain — skipped") {
		t.Fatalf("stdout missing duplicate skip evidence:\n%s", stdout)
	}
	cfg, err := loadFallbackConfig(configPath)
	if err != nil {
		t.Fatalf("load fallback config: %v", err)
	}
	if len(cfg.Chain) != 1 {
		t.Fatalf("fallback chain len = %d, want duplicate skipped: %#v", len(cfg.Chain), cfg.Chain)
	}
}

func TestFallbackCommandRemoveDeletesSelectedEntry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	configPath := filepath.Join(home, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
[[fallback_providers]]
provider = "anthropic"
model = "claude-opus-4-20250514"

[[fallback_providers]]
provider = "openai"
model = "gpt-4o-mini"

[[fallback_providers]]
provider = "openrouter"
model = "nous/hermes-4"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	stdout, stderr, err := runFallbackTestCommandWithInput(t, modelCommandSeams{}, "2\n", "remove")
	if err != nil {
		t.Fatalf("fallback remove: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Removed fallback: gpt-4o-mini  (via openai)") {
		t.Fatalf("stdout missing removed fallback evidence:\n%s", stdout)
	}
	cfg, err := loadFallbackConfig(configPath)
	if err != nil {
		t.Fatalf("load fallback config: %v", err)
	}
	if len(cfg.Chain) != 2 {
		t.Fatalf("fallback chain len = %d, want 2: %#v", len(cfg.Chain), cfg.Chain)
	}
	if cfg.Chain[0].Provider != "anthropic" || cfg.Chain[1].Provider != "openrouter" {
		t.Fatalf("fallback chain = %#v, want first and third entries retained", cfg.Chain)
	}
}

func TestFallbackCommandClearRequiresConfirmation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	configPath := filepath.Join(home, "config.toml")
	if err := os.WriteFile(configPath, []byte(`
fallback_model = { provider = "anthropic", model = "claude-opus-4-20250514" }
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	stdout, stderr, err := runFallbackTestCommandWithInput(t, modelCommandSeams{}, "y\n", "clear")
	if err != nil {
		t.Fatalf("fallback clear: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Current fallback chain (1 entry):") || !strings.Contains(stdout, "Fallback chain cleared.") {
		t.Fatalf("stdout missing clear confirmation evidence:\n%s", stdout)
	}
	cfg, err := loadFallbackConfig(configPath)
	if err != nil {
		t.Fatalf("load fallback config: %v", err)
	}
	if len(cfg.Chain) != 0 {
		t.Fatalf("fallback chain = %#v, want empty after clear", cfg.Chain)
	}
	body, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(body), "fallback_model") {
		t.Fatalf("clear retained legacy fallback_model:\n%s", body)
	}
}

func runFallbackTestCommand(t *testing.T, seams modelCommandSeams, args ...string) (string, string, error) {
	return runFallbackTestCommandWithInput(t, seams, "", args...)
}

func runFallbackTestCommandWithInput(t *testing.T, seams modelCommandSeams, input string, args ...string) (string, string, error) {
	t.Helper()
	cmd := newFallbackCommandWithSeams(seams)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if input != "" {
		cmd.SetIn(strings.NewReader(input))
	}
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}
