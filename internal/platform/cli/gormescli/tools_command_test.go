package gormescli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestToolsCommandListShowsRuntimeToolsets(t *testing.T) {
	setupToolsCommandTestEnv(t)
	writeToolsCommandFixtureConfig(t, `
platform_toolsets = { cli = ["terminal", "web"] }
`)

	stdout, stderr, err := executeToolsCommandForTest(newToolsCommandForTest(), "tools", "list")
	if err != nil {
		t.Fatalf("tools list error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Tools for CLI", "web", "enabled", "terminal"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("tools list stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "hermes_command_unavailable") || strings.Contains(stdout, "row-backed") {
		t.Fatalf("tools list still reports row-backed unavailable:\n%s", stdout)
	}
}

func TestToolsCommandEnableDisablePersistsCLISelection(t *testing.T) {
	setupToolsCommandTestEnv(t)
	writeToolsCommandFixtureConfig(t, `
platform_toolsets = { cli = ["terminal"] }
`)

	stdout, stderr, err := executeToolsCommandForTest(newToolsCommandForTest(), "tools", "enable", "web")
	if err != nil {
		t.Fatalf("tools enable error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "enabled: web") || !strings.Contains(stdout, "session_reset_required=true") {
		t.Fatalf("tools enable stdout = %q, want enabled web with reset evidence", stdout)
	}
	got := readToolsCommandCLIPlatformToolsets(t)
	if !slices.Contains(got, "terminal") || !slices.Contains(got, "web") {
		t.Fatalf("toolsets after enable = %v, want terminal and web", got)
	}

	stdout, stderr, err = executeToolsCommandForTest(newToolsCommandForTest(), "tools", "disable", "terminal")
	if err != nil {
		t.Fatalf("tools disable error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "disabled: terminal") || !strings.Contains(stdout, "session_reset_required=true") {
		t.Fatalf("tools disable stdout = %q, want disabled terminal with reset evidence", stdout)
	}
	got = readToolsCommandCLIPlatformToolsets(t)
	if slices.Contains(got, "terminal") || !slices.Contains(got, "web") {
		t.Fatalf("toolsets after disable = %v, want web only", got)
	}
}

func setupToolsCommandTestEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("HASS_TOKEN", "")
}

func newToolsCommandForTest() *cobra.Command {
	return NewToolsCommand(ToolsCommandOptions{})
}

func executeToolsCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	return executeCobraCommandForTest(cmd, cobraCommandExecutionOptions{StripLeadingArg: "tools"}, args...)
}

func writeToolsCommandFixtureConfig(t *testing.T, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config home: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func readToolsCommandCLIPlatformToolsets(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var doc map[string]any
	if err := toml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse config: %v\n%s", err, string(data))
	}
	platformToolsets, _ := doc["platform_toolsets"].(map[string]any)
	raw, _ := platformToolsets["cli"].([]any)
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		out = append(out, value.(string))
	}
	return out
}
