package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestSetupComplexE2E_ResetThenProviderNonInteractivePreservesBreadcrumbAndScrubsSecrets(t *testing.T) {
	home := t.TempDir()
	oldSecret := "sk-old-reset-secret-must-stay-in-breadcrumb-only"
	newSecret := "sk-new-provider-secret-must-only-enter-dotenv"
	t.Setenv("GORMES_HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "xdg-data"))
	t.Setenv("GORMES_ENDPOINT", "https://provider-reset.example/v1")
	t.Setenv("GORMES_API_KEY", newSecret)
	t.Setenv("GORMES_MODEL", "reset-provider-e2e-model")

	priorConfig := strings.Join([]string{
		"[hermes]",
		`provider = 'stale-provider'`,
		`endpoint = 'https://stale-provider.example/v1'`,
		`model = 'stale-model'`,
		`api_key = '` + oldSecret + `'`,
		"",
	}, "\n")
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(priorConfig), 0o600); err != nil {
		t.Fatalf("write prior config: %v", err)
	}

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "--reset", "provider", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Configuration reset to defaults.",
		"Prior config preserved at",
		"Gormes Setup — Provider",
		"Provider configured.",
		"reset-provider-e2e-model",
		"API key:  stored (redacted)",
		"Test it:  gormes chat",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, leaked := range []string{oldSecret, newSecret, "sk-old", "sk-new", "must-stay", "must-only"} {
		if strings.Contains(stdout+stderr, leaked) {
			t.Fatalf("setup reset/provider output leaked secret material %q:\nstdout=%s\nstderr=%s", leaked, stdout, stderr)
		}
	}

	breadcrumb := findSetupResetBreadcrumb(t)
	breadcrumbBody, err := os.ReadFile(breadcrumb)
	if err != nil {
		t.Fatalf("read breadcrumb: %v", err)
	}
	if string(breadcrumbBody) != priorConfig {
		t.Fatalf("breadcrumb body changed\n got=%q\nwant=%q", string(breadcrumbBody), priorConfig)
	}
	info, err := os.Stat(breadcrumb)
	if err != nil {
		t.Fatalf("stat breadcrumb: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("breadcrumb mode = %v, want 0600 because it contains prior secrets", info.Mode().Perm())
	}

	configBody, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read new config: %v", err)
	}
	for _, want := range []string{`endpoint = 'https://provider-reset.example/v1'`, `model = 'reset-provider-e2e-model'`} {
		if !strings.Contains(string(configBody), want) {
			t.Fatalf("new config missing %q:\n%s", want, string(configBody))
		}
	}
	for _, forbidden := range []string{oldSecret, newSecret, "stale-provider", "stale-model", "api_key"} {
		if strings.Contains(string(configBody), forbidden) {
			t.Fatalf("new config leaked stale/secret value %q:\n%s", forbidden, string(configBody))
		}
	}
	envBody, err := os.ReadFile(config.EnvPath())
	if err != nil {
		t.Fatalf("read dotenv: %v", err)
	}
	if !strings.Contains(string(envBody), "GORMES_API_KEY="+newSecret) {
		t.Fatalf("dotenv missing new API key entry:\n%s", string(envBody))
	}
	if strings.Contains(string(envBody), oldSecret) {
		t.Fatalf("dotenv retained old secret:\n%s", string(envBody))
	}
}

func TestSetupComplexE2E_QuickSlackTargetOrdersCoreChannelLiveAndStopsOnRedactedFailure(t *testing.T) {
	home := t.TempDir()
	secret := "sk-slack-quick-live-secret"
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", secret)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: " ", Model: " "},
	}
	seams := fake.seams()
	seams.RunSetupProvider = func(_ *cobra.Command, nonInteractive bool) error {
		events = append(events, fmt.Sprintf("provider:%t", nonInteractive))
		fake.current = cli.ProviderModel{Provider: "openai", Model: " "}
		return nil
	}
	seams.RunModelPicker = func(*cobra.Command) error {
		events = append(events, "model-picker")
		fake.current = cli.ProviderModel{Provider: "openai", Model: "gpt-4o-mini"}
		return nil
	}
	seams.RunGatewayPlatform = func(_ *cobra.Command, platform string) error {
		events = append(events, "channel:"+platform)
		if platform != string(cli.SetupTargetSlack) {
			t.Fatalf("RunGatewayPlatform platform = %q, want slack", platform)
		}
		return nil
	}
	seams.RunWhatsAppSetup = func(*cobra.Command) error {
		t.Fatal("slack target routed through WhatsApp setup")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		events = append(events, "live-test")
		return fmt.Errorf("live test rejected bearer token %s", secret)
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("quick setup launched chat after failed live test")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--target", "slack")
	if err == nil {
		t.Fatalf("Execute() error = nil, want live-test failure\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code != 1 {
		t.Fatalf("exit code = %d, want 1 err=%v", code, err)
	}
	if got, want := strings.Join(events, ","), "provider:false,model-picker,channel:slack,live-test"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
	}
	for _, want := range []string{
		"Quick Setup - configure missing items only",
		"Provider endpoint or auth is missing.",
		"Model/provider defaults are missing.",
		"Provider live test failed. Chat was not opened.",
		"Repair: gormes setup --quick --target slack",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{
		secret,
		"bearer token " + secret,
		"Channel setup checked. Start messaging",
		"Terminal chat ready",
		"Start chatting with: gormes",
	} {
		if strings.Contains(stdout, forbidden) || strings.Contains(stderr, forbidden) || strings.Contains(err.Error(), forbidden) {
			t.Fatalf("quick slack failure leaked/printed forbidden %q\nstdout=%s\nstderr=%s\nerr=%v", forbidden, stdout, stderr, err)
		}
	}
	for _, path := range []string{config.SessionDBPath(), config.MemoryDBPath(), config.GatewayRuntimeStatusPath()} {
		if _, statErr := os.Stat(path); statErr == nil {
			t.Fatalf("quick setup failure created runtime artifact %s", path)
		} else if !os.IsNotExist(statErr) {
			t.Fatalf("stat runtime artifact %s: %v", path, statErr)
		}
	}
}

func TestSetupComplexE2E_QuickHeadlessDiscordTargetNeverPromptsOrStartsRuntime(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "sk-headless-discord-secret")

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   false,
		current: cli.ProviderModel{Provider: " ", Model: " "},
	}
	seams := fake.seams()
	seams.ChooseSetupTarget = func(*cobra.Command, []cli.SetupTargetOption, int) (cli.SetupTargetID, error) {
		t.Fatal("headless quick setup prompted for target")
		return "", nil
	}
	seams.RunSetupProvider = func(_ *cobra.Command, nonInteractive bool) error {
		events = append(events, fmt.Sprintf("provider:%t", nonInteractive))
		fake.current = cli.ProviderModel{Provider: "anthropic", Model: "claude-sonnet-4"}
		return nil
	}
	seams.RunGatewayPlatform = func(*cobra.Command, string) error {
		t.Fatal("headless quick setup started interactive gateway platform setup")
		return nil
	}
	seams.RunWhatsAppSetup = func(*cobra.Command) error {
		t.Fatal("discord target routed through WhatsApp setup")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		events = append(events, "live-test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("headless channel setup launched chat/TUI")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--non-interactive", "--target", "discord")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got, want := strings.Join(events, ","), "provider:true,live-test"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
	}
	for _, want := range []string{
		"Quick Setup - configure missing items only",
		"Provider endpoint or auth is missing.",
		"Current model/provider: claude-sonnet-4 via anthropic",
		"No missing core setup items detected.",
		"Channel setup command: gormes setup gateway",
		"Channel setup checked. Start messaging with: gormes gateway",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{"sk-headless-discord-secret", "Select target", "Select option", "Terminal chat ready", "Start chatting with: gormes\n"} {
		if strings.Contains(stdout, forbidden) || strings.Contains(stderr, forbidden) {
			t.Fatalf("headless discord setup leaked/printed forbidden %q\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
		}
	}
	for _, path := range []string{config.SessionDBPath(), config.MemoryDBPath(), config.GatewayRuntimeStatusPath()} {
		if _, statErr := os.Stat(path); statErr == nil {
			t.Fatalf("headless setup created runtime artifact %s", path)
		} else if !os.IsNotExist(statErr) {
			t.Fatalf("stat runtime artifact %s: %v", path, statErr)
		}
	}
}

func findSetupResetBreadcrumb(t *testing.T) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(config.ConfigPath()))
	if err != nil {
		t.Fatalf("read config dir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "config.toml.before-reset.") {
			return filepath.Join(filepath.Dir(config.ConfigPath()), entry.Name())
		}
	}
	t.Fatalf("reset breadcrumb not found in %s; entries=%v", filepath.Dir(config.ConfigPath()), dirNames(entries))
	return ""
}
