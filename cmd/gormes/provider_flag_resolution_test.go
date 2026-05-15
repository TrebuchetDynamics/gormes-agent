package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/spf13/cobra"
)

func TestProviderFlagResolution_RootHelpExposesEndpointAndAPIKey(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "--help")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	for _, want := range []string{"--endpoint", "--api-key", "--model", "--provider"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("root help missing %q\nstdout=%s", want, stdout)
		}
	}
}

func TestRootHelpDoesNotAdvertiseUnimplementedShortcuts(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "--help")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	for _, forbidden := range []string{"gormes agent create", "gormes agent edit", "gormes mcp list", "gormes mcp add", "gormes profile set"} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("root help advertised unimplemented shortcut %q:\n%s", forbidden, stdout)
		}
	}
	for _, want := range []string{"gormes agent reset", "gormes mcp login <server>", "gormes profile use <name>"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("root help missing implemented shortcut %q:\n%s", want, stdout)
		}
	}
}

func TestProviderFlagResolution_OneshotEndpointAPIKeyAndModelOverrideConfigAndEnv(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
endpoint = "http://config-endpoint:8642"
api_key = "config-secret"
model = "config-model"
`))
	t.Setenv("GORMES_ENDPOINT", "http://env-endpoint:8642")
	t.Setenv("GORMES_API_KEY", "env-secret")
	t.Setenv("GORMES_MODEL", "env-model")

	var gotCfg config.Config
	var gotInvocation oneshotInvocation
	cmd := newRootCommandWithRuntime(rootRuntime{
		runTUI: func(*cobra.Command, []string) error {
			t.Fatal("runTUI was called for oneshot")
			return nil
		},
		newOneshotClient: func(_ context.Context, cfg config.Config, invocation oneshotInvocation) (hermes.Client, error) {
			gotCfg = cfg
			gotInvocation = invocation
			return nil, errors.New("stop before kernel")
		},
	})

	stdout, stderr, err := executeOneshotFlagCommand(
		cmd,
		"--endpoint", "http://flag-endpoint:8642",
		"--api-key", "sk-flag-secret-123456",
		"--model", "flag-model",
		"--provider", "openrouter",
		"chat", "-q", "hi",
	)
	if err == nil {
		t.Fatalf("Execute() error = nil, want setup failure\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if gotCfg.Hermes.Endpoint != "http://flag-endpoint:8642" {
		t.Fatalf("client endpoint = %q, want flag endpoint", gotCfg.Hermes.Endpoint)
	}
	if gotCfg.Hermes.APIKey != "sk-flag-secret-123456" {
		t.Fatalf("client api key = %q, want flag key", gotCfg.Hermes.APIKey)
	}
	if gotInvocation.Inference.Model != "flag-model" || gotInvocation.Inference.ModelSource != config.InferenceValueSourceFlag {
		t.Fatalf("model resolution = %+v, want flag-model from flag", gotInvocation.Inference)
	}
	if gotInvocation.Inference.Provider != "openrouter" || gotInvocation.Inference.ProviderSource != config.InferenceValueSourceFlag {
		t.Fatalf("provider resolution = %+v, want openrouter from flag", gotInvocation.Inference)
	}

	reloaded, loadErr := config.Load(nil)
	if loadErr != nil {
		t.Fatalf("Load(nil): %v", loadErr)
	}
	if reloaded.Hermes.Endpoint != "http://env-endpoint:8642" || reloaded.Hermes.APIKey != "env-secret" {
		t.Fatalf("persisted/env config mutated or bypassed: endpoint=%q api_key=%q", reloaded.Hermes.Endpoint, reloaded.Hermes.APIKey)
	}
}

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

	var got tuiInvocation
	cmd := newRootCommandWithRuntime(rootRuntime{
		runResolvedTUI: func(_ *cobra.Command, invocation tuiInvocation) error {
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

func TestProviderFlagResolution_RedactsAPIKeyFromReturnedErrorsAndStderr(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	const secret = "sk-redact-me-123456"

	cmd := newRootCommandWithRuntime(rootRuntime{
		newOneshotClient: func(_ context.Context, cfg config.Config, _ oneshotInvocation) (hermes.Client, error) {
			return nil, errors.New("provider rejected api_key=" + cfg.Hermes.APIKey)
		},
	})

	stdout, stderr, err := executeOneshotFlagCommand(cmd, "--model", "flag-model", "--api-key", secret, "chat", "-q", "hi")
	if err == nil {
		t.Fatalf("Execute() error = nil, want provider setup failure\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("returned error leaked API key: %v", err)
	}
	if strings.Contains(stderr, secret) {
		t.Fatalf("stderr leaked API key:\n%s", stderr)
	}
	if !strings.Contains(stderr, "[REDACTED]") {
		t.Fatalf("stderr missing redaction marker:\n%s", stderr)
	}
}
