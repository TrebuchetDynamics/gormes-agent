package main

import (
	"encoding/json"
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

func TestSetupComplexE2E_QuickSlackTargetStartsChannelBeforeCoreOnFreshInstall(t *testing.T) {
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
	seams.RunSetupProvider = func(*cobra.Command, bool) error {
		t.Fatal("slack target sent operator to provider before channel setup")
		return nil
	}
	seams.RunModelPicker = func(*cobra.Command) error {
		t.Fatal("slack target sent operator to model before channel setup")
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
		t.Fatal("slack target ran provider live test before provider setup")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("quick setup launched chat/TUI")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--target", "slack")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got, want := strings.Join(events, ","), "channel:slack"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
	}
	for _, want := range []string{
		"Slack channel setup checked.",
		"Provider/model setup is still required before `gormes gateway` can answer Slack.",
		"Next setup command: gormes setup provider",
		"After that, start gateway: gormes gateway",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{
		secret,
		"bearer token " + secret,
		"Quick Setup - configure missing items only",
		"Provider endpoint or auth is missing.",
		"Model/provider defaults are missing.",
		"Provider live test failed",
		"Channel setup checked. Start messaging",
		"Terminal chat ready",
		"Start chatting with: gormes",
	} {
		if strings.Contains(stdout, forbidden) || strings.Contains(stderr, forbidden) {
			t.Fatalf("quick slack setup leaked/printed forbidden %q\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
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

func TestSetupComplexE2E_ProviderNonInteractiveMergesDotenvWithoutDuplicateSecrets(t *testing.T) {
	home := t.TempDir()
	oldSecret := "sk-old-dotenv-secret"
	duplicateSecret := "sk-duplicate-dotenv-secret"
	newSecret := "sk-new-dotenv-secret"
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_ENDPOINT", "https://merge-dotenv.example/v1")
	t.Setenv("GORMES_MODEL", "merge-dotenv-model")
	t.Setenv("GORMES_API_KEY", newSecret)

	if err := os.MkdirAll(filepath.Dir(config.EnvPath()), 0o700); err != nil {
		t.Fatalf("mkdir env dir: %v", err)
	}
	priorEnv := strings.Join([]string{
		"# operator managed file",
		"GORMES_API_KEY=" + oldSecret,
		"UNRELATED_SERVICE_TOKEN=keep-this-token-value",
		"GORMES_API_KEY=" + duplicateSecret,
		`QUOTED_VALUE="preserve me"`,
		"",
	}, "\n")
	if err := os.WriteFile(config.EnvPath(), []byte(priorEnv), 0o644); err != nil {
		t.Fatalf("write prior dotenv: %v", err)
	}

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "provider", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Gormes Setup — Provider", "Provider configured.", "merge-dotenv-model", "API key:  stored (redacted)"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, leaked := range []string{oldSecret, duplicateSecret, newSecret, "keep-this-token-value", "sk-old", "sk-duplicate", "sk-new"} {
		if strings.Contains(stdout+stderr, leaked) {
			t.Fatalf("setup provider output leaked dotenv material %q:\nstdout=%s\nstderr=%s", leaked, stdout, stderr)
		}
	}

	envBody, err := os.ReadFile(config.EnvPath())
	if err != nil {
		t.Fatalf("read dotenv: %v", err)
	}
	envText := string(envBody)
	if count := strings.Count(envText, "GORMES_API_KEY="); count != 1 {
		t.Fatalf("dotenv GORMES_API_KEY occurrences = %d, want exactly 1 after merge:\n%s", count, envText)
	}
	for _, want := range []string{"# operator managed file", "GORMES_API_KEY=" + newSecret, "UNRELATED_SERVICE_TOKEN=keep-this-token-value", `QUOTED_VALUE="preserve me"`} {
		if !strings.Contains(envText, want) {
			t.Fatalf("dotenv missing preserved/new line %q:\n%s", want, envText)
		}
	}
	for _, forbidden := range []string{oldSecret, duplicateSecret} {
		if strings.Contains(envText, forbidden) {
			t.Fatalf("dotenv retained old duplicate secret %q:\n%s", forbidden, envText)
		}
	}
	info, err := os.Stat(config.EnvPath())
	if err != nil {
		t.Fatalf("stat dotenv: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("dotenv mode = %v, want 0600 after secret rewrite", info.Mode().Perm())
	}

	configBody, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(configBody), newSecret) || strings.Contains(string(configBody), "api_key") {
		t.Fatalf("config.toml leaked API key material:\n%s", string(configBody))
	}
}

func TestSetupComplexE2E_QuickTUITargetAliasRunsLiveTestThenLaunchesChatOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "sk-tui-target-secret")
	writeSetupGatewayFixtureConfig(t, `
[hermes]
provider = "openai-codex"
endpoint = "https://chatgpt.com/backend-api/codex"
model = "gpt-5.2-codex"
`)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: "openai-codex", Model: "gpt-5.2-codex"},
	}
	seams := fake.seams()
	seams.RunSetupProvider = func(*cobra.Command, bool) error {
		t.Fatal("configured tui quick setup ran provider setup")
		return nil
	}
	seams.RunModelPicker = func(*cobra.Command) error {
		t.Fatal("configured tui quick setup ran model picker")
		return nil
	}
	seams.RunGatewayPlatform = func(*cobra.Command, string) error {
		t.Fatal("tui target routed through channel setup")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		events = append(events, "live-test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		events = append(events, "chat")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--target", "tui")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got, want := strings.Join(events, ","), "live-test,chat"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
	}
	for _, want := range []string{"Quick Setup - configure missing items only", "Current model/provider: gpt-5.2-codex via openai-codex", "No missing core setup items detected."} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{"sk-tui-target-secret", "Channel setup command", "Channel setup checked", "Terminal chat ready", "setup gateway"} {
		if strings.Contains(stdout, forbidden) || strings.Contains(stderr, forbidden) {
			t.Fatalf("tui target output leaked/printed forbidden %q\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
		}
	}
	for _, path := range []string{config.SessionDBPath(), config.MemoryDBPath(), config.GatewayRuntimeStatusPath()} {
		if _, statErr := os.Stat(path); statErr == nil {
			t.Fatalf("quick tui setup created runtime artifact %s", path)
		} else if !os.IsNotExist(statErr) {
			t.Fatalf("stat runtime artifact %s: %v", path, statErr)
		}
	}
}

func TestSetupComplexE2E_ResetJSONDoesNotRunRequestedSectionOrLeakSecrets(t *testing.T) {
	home := t.TempDir()
	oldSecret := "sk-json-reset-old-secret"
	newSecret := "sk-json-reset-new-secret"
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_ENDPOINT", "https://json-reset.example/v1")
	t.Setenv("GORMES_API_KEY", newSecret)
	t.Setenv("GORMES_MODEL", "json-reset-model")

	priorConfig := strings.Join([]string{
		"[hermes]",
		`provider = 'openai'`,
		`endpoint = 'https://old-json-reset.example/v1'`,
		`model = 'old-json-reset-model'`,
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
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "--reset", "--json", "provider", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	var report struct {
		Action         string `json:"action"`
		ConfigPath     string `json:"config_path"`
		BreadcrumbPath string `json:"breadcrumb_path"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("reset JSON did not parse: %v\nstdout=%s", err, stdout)
	}
	if report.Action != "reset" || report.ConfigPath != config.ConfigPath() || report.BreadcrumbPath == "" {
		t.Fatalf("reset JSON report = %+v, want reset action/config/breadcrumb", report)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty for reset JSON", stderr)
	}
	for _, forbidden := range []string{oldSecret, newSecret, "Provider configured", "API key", "json-reset-model", "old-json-reset-model"} {
		if strings.Contains(stdout, forbidden) || strings.Contains(stderr, forbidden) {
			t.Fatalf("reset JSON leaked/ran forbidden %q\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
		}
	}
	breadcrumbBody, err := os.ReadFile(report.BreadcrumbPath)
	if err != nil {
		t.Fatalf("read breadcrumb: %v", err)
	}
	if string(breadcrumbBody) != priorConfig {
		t.Fatalf("breadcrumb body changed\n got=%q\nwant=%q", string(breadcrumbBody), priorConfig)
	}
	if _, err := os.Stat(config.EnvPath()); !os.IsNotExist(err) {
		t.Fatalf("reset JSON should not run provider section or write dotenv %s: %v", config.EnvPath(), err)
	}
	configBody, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read reset config: %v", err)
	}
	for _, forbidden := range []string{oldSecret, newSecret, "old-json-reset-model", "json-reset-model", "api_key"} {
		if strings.Contains(string(configBody), forbidden) {
			t.Fatalf("reset config leaked forbidden %q:\n%s", forbidden, string(configBody))
		}
	}
}

func TestSetupComplexE2E_FirstRunOpenClawMigrationRoutesWithoutSetupMutation(t *testing.T) {
	home := t.TempDir()
	gormesHome := filepath.Join(home, ".gormes")
	openclawSource := filepath.Join(home, ".openclaw")
	t.Setenv("HOME", home)
	t.Setenv("GORMES_HOME", gormesHome)
	if err := os.MkdirAll(openclawSource, 0o700); err != nil {
		t.Fatalf("mkdir openclaw source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(openclawSource, "config.toml"), []byte("api_key='sk-openclaw-migration-secret'\n"), 0o600); err != nil {
		t.Fatalf("write openclaw fixture: %v", err)
	}

	fake := &setupCommandFakeSeams{isTTY: true, freshInstall: true}
	fake.detectHermes = func() string { return "" }
	fake.detectOpenClaw = func() string { return openclawSource }
	seams := fake.seams()
	seams.ChooseSetupAction = func(_ *cobra.Command, options []setupMenuOption, defaultOption int) (setupAction, error) {
		if defaultOption != 0 {
			t.Fatalf("default option = %d, want quick setup default", defaultOption)
		}
		var labels []string
		for _, option := range options {
			labels = append(labels, option.Label)
		}
		if got := strings.Join(labels, ","); got != "Quick setup — provider, model & messaging (recommended),Full setup — configure everything,Migrate OpenClaw" {
			t.Fatalf("first-run options = %s", got)
		}
		return setupActionMigrateOpenClaw, nil
	}
	seams.RunSetupProvider = func(*cobra.Command, bool) error {
		t.Fatal("migration route ran provider setup")
		return nil
	}
	seams.RunFullWizard = func(*cobra.Command, bool) error {
		t.Fatal("migration route ran full setup wizard")
		return nil
	}
	seams.RunSetupGateway = func(*cobra.Command, bool) error {
		t.Fatal("migration route ran gateway setup")
		return nil
	}
	seams.RunSetupTools = func(*cobra.Command, bool) error {
		t.Fatal("migration route ran tools setup")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams)
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Gormes Agent Setup Wizard",
		"No existing Gormes configuration was found.",
		"Migrate from openclaw",
		"Found openclaw at: " + openclawSource,
		"gormes migrate openclaw --dry-run",
		"gormes migrate openclaw --yes --source " + openclawSource,
		"does not overwrite files without --overwrite",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{"sk-openclaw-migration-secret", "Provider configured", "Setup Complete", "API key", "gormes chat"} {
		if strings.Contains(stdout, forbidden) || strings.Contains(stderr, forbidden) {
			t.Fatalf("migration route leaked/printed forbidden %q\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
		}
	}
	for _, path := range []string{config.ConfigPath(), config.EnvPath(), config.SessionDBPath(), config.GatewayRuntimeStatusPath()} {
		if _, statErr := os.Stat(path); statErr == nil {
			t.Fatalf("migration route created setup/runtime artifact %s", path)
		} else if !os.IsNotExist(statErr) {
			t.Fatalf("stat migration artifact %s: %v", path, statErr)
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
