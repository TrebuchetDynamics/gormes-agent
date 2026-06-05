package providers

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

	stdout, stderr, err := executeFallbackCommandForTest(
		NewFallbackCommandWithSeams(ModelCommandSeams{}),
		"list",
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

func executeFallbackCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}
