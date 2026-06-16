package tuiapp

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestProviderFlagResolution_TUIUsesInvocationOnlyEndpointAndAPIKey(t *testing.T) {
	setupTUIModelOverrideTestEnv(t)
	writeTUIModelOverrideConfig(t, []byte(`
[hermes]
endpoint = "http://config-endpoint:8642"
api_key = "config-secret"
model = "config-model"
`))
	t.Setenv("GORMES_ENDPOINT", "http://env-endpoint:8642")
	t.Setenv("GORMES_API_KEY", "env-secret")

	var got Invocation
	cmd := newRootCommandWithRuntime(Runtime{
		RunResolvedTUI: func(_ *cobra.Command, invocation Invocation) error {
			got = invocation
			return nil
		},
	})

	stdout, stderr, err := executeTUIModelOverrideCommand(
		cmd,
		"--offline",
		"--endpoint", "http://flag-endpoint:8642",
		"--api-key", "sk-tui-secret-123456",
		"--model", "flag-model",
		"--provider", "openrouter",
	)
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	if got.Config.Hermes.Endpoint != "http://flag-endpoint:8642" {
		t.Fatalf("TUI endpoint = %q, want flag endpoint", got.Config.Hermes.Endpoint)
	}
	if got.Config.Hermes.APIKey != "sk-tui-secret-123456" {
		t.Fatalf("TUI api key = %q, want flag key", got.Config.Hermes.APIKey)
	}
	if got.Inference.Model != "flag-model" || got.Inference.Provider != "openrouter" {
		t.Fatalf("TUI inference = %+v, want flag model/provider", got.Inference)
	}

	configBytes, readErr := os.ReadFile(config.ConfigPath())
	if readErr != nil {
		t.Fatalf("read config: %v", readErr)
	}
	if strings.Contains(string(configBytes), "sk-tui-secret-123456") || strings.Contains(string(configBytes), "http://flag-endpoint:8642") {
		t.Fatalf("invocation-only flags were persisted to config.toml:\n%s", string(configBytes))
	}
}
