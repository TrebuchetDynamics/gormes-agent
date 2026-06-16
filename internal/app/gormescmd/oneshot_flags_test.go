package gormescmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/spf13/cobra"
)

func TestOneshotFlags_ModelFlagParsesWithoutTUIOrHealthCheck(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	var got oneshotInvocation
	var oneshotCalls int
	cmd := newRootCommandWithRuntime(rootRuntime{
		runTUI: func(*cobra.Command, []string) error {
			t.Fatal("runTUI was called for oneshot")
			return nil
		},
		runOneshot: func(_ *cobra.Command, invocation oneshotInvocation) error {
			oneshotCalls++
			got = invocation
			return nil
		},
	})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "--model", "fixture-model", "chat", "-q", "hi")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	if oneshotCalls != 1 {
		t.Fatalf("runOneshot calls = %d, want 1", oneshotCalls)
	}
	if got.Prompt != "hi" {
		t.Fatalf("Prompt = %q, want hi", got.Prompt)
	}
	if got.Inference.Model != "fixture-model" || got.Inference.ModelSource != config.InferenceValueSourceFlag {
		t.Fatalf("model resolution = %+v, want fixture-model from flag", got.Inference)
	}
	if got.Inference.Provider != "" || got.Inference.ProviderSource != config.InferenceValueSourceUnset {
		t.Fatalf("provider resolution = %+v, want unset provider", got.Inference)
	}
	if !got.Inference.ProviderAutoDetectRequired {
		t.Fatalf("ProviderAutoDetectRequired = false, want true for explicit model without provider")
	}
	if strings.Contains(stderr, "api_server") {
		t.Fatalf("stderr contains api_server health output:\n%s", stderr)
	}
}

func TestOneshotFlags_ProviderWithoutExplicitModelExits2BeforeRunners(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
model = "stale-configured-model"
`))

	var tuiCalls, oneshotCalls int
	cmd := newRootCommandWithRuntime(rootRuntime{
		runTUI: func(*cobra.Command, []string) error {
			tuiCalls++
			return nil
		},
		runOneshot: func(*cobra.Command, oneshotInvocation) error {
			oneshotCalls++
			return nil
		},
	})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "--provider", "openrouter", "chat", "-q", "hi")
	if err == nil {
		t.Fatalf("Execute() error = nil, want provider/model ambiguity error\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code != 2 {
		t.Fatalf("exit code = %d, want 2 (err=%v)", code, err)
	}
	if tuiCalls != 0 || oneshotCalls != 0 {
		t.Fatalf("runner calls = tui:%d oneshot:%d, want none", tuiCalls, oneshotCalls)
	}
	for _, want := range []string{
		"gormes chat -q: --provider requires --model (or GORMES_INFERENCE_MODEL)",
		"Pass both explicitly, or neither to use your configured defaults.",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr missing %q\nstderr=%s", want, stderr)
		}
	}
	if strings.Contains(stderr, "api_server") {
		t.Fatalf("stderr contains api_server health output:\n%s", stderr)
	}
}

func TestOneshotFlags_ProviderFlagAllowsEnvModelWithoutMutatingConfig(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
model = "configured-model"
`))
	t.Setenv("GORMES_INFERENCE_MODEL", "env-model")

	var got oneshotInvocation
	cmd := newRootCommandWithRuntime(rootRuntime{
		runTUI: func(*cobra.Command, []string) error {
			t.Fatal("runTUI was called for oneshot")
			return nil
		},
		runOneshot: func(_ *cobra.Command, invocation oneshotInvocation) error {
			got = invocation
			return nil
		},
	})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "--provider", "openrouter", "chat", "-q", "hi")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	if got.Inference.Model != "env-model" || got.Inference.ModelSource != config.InferenceValueSourceEnv {
		t.Fatalf("model resolution = %+v, want env-model from env", got.Inference)
	}
	if got.Inference.Provider != "openrouter" || got.Inference.ProviderSource != config.InferenceValueSourceFlag {
		t.Fatalf("provider resolution = %+v, want openrouter from flag", got.Inference)
	}
	if got.Inference.ProviderAutoDetectRequired {
		t.Fatalf("ProviderAutoDetectRequired = true, want false when provider is explicit")
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load(nil): %v", err)
	}
	if cfg.Hermes.Model != "configured-model" {
		t.Fatalf("cfg.Hermes.Model = %q, want persisted configured-model", cfg.Hermes.Model)
	}
}

func TestOneshotFlags_EnvModelAndProviderFallbacksAreRecorded(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_INFERENCE_MODEL", "env-model")
	t.Setenv("GORMES_INFERENCE_PROVIDER", "openrouter")

	var got oneshotInvocation
	cmd := newRootCommandWithRuntime(rootRuntime{
		runTUI: func(*cobra.Command, []string) error {
			t.Fatal("runTUI was called for oneshot")
			return nil
		},
		runOneshot: func(_ *cobra.Command, invocation oneshotInvocation) error {
			got = invocation
			return nil
		},
	})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "chat", "-q", "hi")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	if got.Inference.Model != "env-model" || got.Inference.ModelSource != config.InferenceValueSourceEnv {
		t.Fatalf("model resolution = %+v, want env-model from env", got.Inference)
	}
	if got.Inference.Provider != "openrouter" || got.Inference.ProviderSource != config.InferenceValueSourceEnv {
		t.Fatalf("provider resolution = %+v, want openrouter from env", got.Inference)
	}
	if got.Inference.ProviderAutoDetectRequired {
		t.Fatalf("ProviderAutoDetectRequired = true, want false when provider comes from env")
	}
}

func TestOneshotFlags_ConfigModelFallbackIsRecorded(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
model = "configured-model"
`))

	var got oneshotInvocation
	cmd := newRootCommandWithRuntime(rootRuntime{
		runTUI: func(*cobra.Command, []string) error {
			t.Fatal("runTUI was called for oneshot")
			return nil
		},
		runOneshot: func(_ *cobra.Command, invocation oneshotInvocation) error {
			got = invocation
			return nil
		},
	})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "chat", "-q", "hi")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	if got.Inference.Model != "configured-model" || got.Inference.ModelSource != config.InferenceValueSourceConfig {
		t.Fatalf("model resolution = %+v, want configured-model from config", got.Inference)
	}
	if got.Inference.ProviderAutoDetectRequired {
		t.Fatalf("ProviderAutoDetectRequired = true, want false for config defaults")
	}
}

func setupOneshotFlagTestEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes"))
	t.Setenv("CODEX_HOME", filepath.Join(root, "codex"))
	t.Setenv("GORMES_ENDPOINT", "")
	t.Setenv("GORMES_MODEL", "")
	t.Setenv("GORMES_API_KEY", "")
	t.Setenv("GORMES_INFERENCE_MODEL", "")
	t.Setenv("GORMES_INFERENCE_PROVIDER", "")
	for _, key := range []string{
		"GATEWAY_PROXY_KEY",
		"GATEWAY_PROXY_URL",
		"GITHUB_TOKEN",
		"GH_TOKEN",
		"GORMES_BROWSER_CDP_URL",
		"BROWSER_CDP_URL",
		"CHROME_REMOTE_DEBUGGING_URL",
		"GORMES_TELEGRAM_BOT_TOKEN",
		"GORMES_TELEGRAM_TOKEN",
		"HERMES_TELEGRAM_BOT_TOKEN",
		"HERMES_TELEGRAM_TOKEN",
		"TELEGRAM_BOT_TOKEN",
		"TELEGRAM_TOKEN",
		"GORMES_TELEGRAM_HOME_CHANNEL",
		"GORMES_TELEGRAM_CHAT_ID",
		"HERMES_TELEGRAM_HOME_CHANNEL",
		"HERMES_TELEGRAM_CHAT_ID",
		"TELEGRAM_HOME_CHANNEL",
		"TELEGRAM_CHAT_ID",
		"GORMES_TELEGRAM_HOME_CHANNEL_NAME",
		"HERMES_TELEGRAM_HOME_CHANNEL_NAME",
		"TELEGRAM_HOME_CHANNEL_NAME",
		"GORMES_TELEGRAM_HOME_CHANNEL_THREAD_ID",
		"HERMES_TELEGRAM_HOME_CHANNEL_THREAD_ID",
		"TELEGRAM_HOME_CHANNEL_THREAD_ID",
		"GORMES_TELEGRAM_ALLOWED_USERS",
		"HERMES_TELEGRAM_ALLOWED_USERS",
		"TELEGRAM_ALLOWED_USERS",
		"GORMES_TELEGRAM_ALLOWED_CHATS",
		"HERMES_TELEGRAM_ALLOWED_CHATS",
		"TELEGRAM_ALLOWED_CHATS",
		"GORMES_TELEGRAM_GUEST_MODE",
		"HERMES_TELEGRAM_GUEST_MODE",
		"TELEGRAM_GUEST_MODE",
		"GORMES_TELEGRAM_NOTIFICATIONS",
		"HERMES_TELEGRAM_NOTIFICATIONS",
		"TELEGRAM_NOTIFICATIONS",
		"GORMES_DISCORD_TOKEN",
		"GORMES_DISCORD_CHANNEL_ID",
		"GORMES_DISCORD_ALLOWED_CHANNELS",
		"DISCORD_ALLOWED_CHANNELS",
		"GORMES_DISCORD_IGNORED_CHANNELS",
		"DISCORD_IGNORED_CHANNELS",
		"GORMES_DISCORD_FREE_RESPONSE_CHANNELS",
		"DISCORD_FREE_RESPONSE_CHANNELS",
		"GORMES_DISCORD_NO_THREAD_CHANNELS",
		"DISCORD_NO_THREAD_CHANNELS",
		"GORMES_DISCORD_REQUIRE_MENTION",
		"DISCORD_REQUIRE_MENTION",
		"GORMES_DISCORD_AUTO_THREAD",
		"DISCORD_AUTO_THREAD",
		"GORMES_DISCORD_REPLY_TO_MODE",
		"DISCORD_REPLY_TO_MODE",
		"GORMES_DISCORD_ALLOW_BOTS",
		"DISCORD_ALLOW_BOTS",
		"GORMES_DISCORD_SERVER_ACTIONS",
		"GORMES_SLACK_ENABLED",
		"GORMES_SLACK_BOT_TOKEN",
		"GORMES_SLACK_APP_TOKEN",
		"GORMES_SLACK_CHANNEL_ID",
		"GORMES_SLACK_ALLOWED_CHANNELS",
		"SLACK_ALLOWED_CHANNELS",
		"GORMES_SLACK_COALESCE_MS",
		"GORMES_SLACK_FIRST_RUN_DISCOVERY",
		"GORMES_SLACK_REQUIRE_MENTION",
		"GORMES_SLACK_STRICT_MENTION",
		"GORMES_SLACK_FREE_RESPONSE_CHANNELS",
		"GORMES_SLACK_REPLY_IN_THREAD",
		"GORMES_NAVIVOX_ENABLED",
		"GORMES_NAVIVOX_BIND_HOST",
		"GORMES_NAVIVOX_PORT",
		"GORMES_NAVIVOX_EXPOSURE_MODE",
		"GORMES_NAVIVOX_AUTH_MODE",
		"GORMES_NAVIVOX_TOKEN",
		"GORMES_NAVIVOX_ALLOW_ORIGINS",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("GORMES_KANBAN_DB", "")
	t.Setenv("GORMES_KANBAN_HOME", "")
	t.Setenv("GORMES_KANBAN_TASK", "")
	t.Setenv("HERMES_KANBAN_BOARD", "")
	t.Setenv("HERMES_KANBAN_DB", "")
}

func writeOneshotFlagConfig(t *testing.T, data []byte) {
	t.Helper()
	path := config.ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func executeOneshotFlagCommand(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestOneshotFirstRunDoesNotAffectScriptedChat(t *testing.T) {
	freshInstallE2EHome(t)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
provider = "openai"
endpoint = "https://api.openai.com/v1"
model = "gpt-4o-mini"
api_key = "sk-test-oneshot"
`))

	var setupCalls int
	var tuiCalls int
	var gotPrompt string
	cmd := newRootCommandWithRuntime(rootRuntime{
		isTTY: func() bool { return true },
		runFirstRunSetup: func(*cobra.Command) error {
			setupCalls++
			return nil
		},
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
			tuiCalls++
			return nil
		},
		runOneshot: func(_ *cobra.Command, invocation oneshotInvocation) error {
			gotPrompt = invocation.Prompt
			return nil
		},
	})

	stdout, stderr, err := executeOneshotFlagCommand(cmd, "chat", "-q", "hello")
	if err != nil {
		t.Fatalf("chat -q: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if gotPrompt != "hello" {
		t.Fatalf("scripted chat prompt = %q, want hello", gotPrompt)
	}
	if setupCalls != 0 {
		t.Fatalf("runFirstRunSetup calls = %d, want 0", setupCalls)
	}
	if tuiCalls != 0 {
		t.Fatalf("runResolvedTUI calls = %d, want 0", tuiCalls)
	}
}
