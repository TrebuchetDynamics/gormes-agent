package gormescmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/spf13/cobra"
)

func TestSetupEntryMode_ExistingBareRunsFullWizardNoChooser(t *testing.T) {
	fullCalls := 0
	chooserCalls := 0
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: "openai-codex", Model: "gpt-5.5"},
	}
	seams := fake.seams()
	seams.ChooseSetupAction = func(*cobra.Command, []setupMenuOption, int) (setupAction, error) {
		chooserCalls++
		return setupActionExit, nil
	}
	seams.RunFullWizard = func(cmd *cobra.Command, nonInteractive bool) error {
		fullCalls++
		if nonInteractive {
			t.Fatal("existing bare setup ran full wizard in non-interactive mode")
		}
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams)
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if fullCalls != 1 {
		t.Fatalf("RunFullWizard calls = %d, want 1", fullCalls)
	}
	if chooserCalls != 0 {
		t.Fatalf("ChooseSetupAction calls = %d, want 0", chooserCalls)
	}
	if strings.Contains(stdout, "What would you like to do?") {
		t.Fatalf("existing bare setup rendered Gormes chooser:\n%s", stdout)
	}
}

func TestSetupEntryMode_ReconfigureMatchesExistingBare(t *testing.T) {
	fullCalls := 0
	chooserCalls := 0
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: "openai-codex", Model: "gpt-5.5"},
	}
	seams := fake.seams()
	seams.ChooseSetupAction = func(*cobra.Command, []setupMenuOption, int) (setupAction, error) {
		chooserCalls++
		return setupActionExit, nil
	}
	seams.RunFullWizard = func(_ *cobra.Command, nonInteractive bool) error {
		fullCalls++
		if nonInteractive {
			t.Fatal("existing reconfigure ran full wizard in non-interactive mode")
		}
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--reconfigure")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if fullCalls != 1 || chooserCalls != 0 {
		t.Fatalf("fullCalls=%d chooserCalls=%d, want full=1 chooser=0", fullCalls, chooserCalls)
	}
}

func TestSetupEntryMode_QuickExistingRunsMissingItemsOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(`
[hermes]
provider = "anthropic"
endpoint = "https://api.anthropic.com/v1"
model = "claude-sonnet-4"
api_key = "test-api-key"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	fullCalls := 0
	chooserCalls := 0
	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: "anthropic", Model: "claude-sonnet-4"},
	}
	seams := fake.seams()
	seams.ChooseSetupAction = func(*cobra.Command, []setupMenuOption, int) (setupAction, error) {
		chooserCalls++
		return setupActionExit, nil
	}
	seams.RunFullWizard = func(*cobra.Command, bool) error {
		fullCalls++
		return nil
	}
	seams.ChooseSetupTarget = func(*cobra.Command, []cli.SetupTargetOption, int) (cli.SetupTargetID, error) {
		events = append(events, "target")
		return cli.SetupTargetTerminal, nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		events = append(events, "live-test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		events = append(events, "chat")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if fullCalls != 0 || chooserCalls != 0 {
		t.Fatalf("fullCalls=%d chooserCalls=%d, want both 0", fullCalls, chooserCalls)
	}
	for _, want := range []string{"Quick Setup - configure missing items only", "Current model/provider: claude-sonnet-4 via anthropic", "No missing core setup items detected."} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if got, want := strings.Join(events, ","), "target,live-test,chat"; got != want {
		t.Fatalf("events = %s, want %s", got, want)
	}
}

func TestSetupEntryMode_FreshInstallPromptsQuickVsFull(t *testing.T) {
	for _, args := range [][]string{nil, []string{"--reconfigure"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			chooserCalls := 0
			fullCalls := 0
			var captured []setupMenuOption
			fake := &setupCommandFakeSeams{isTTY: true, freshInstall: true}
			seams := fake.seams()
			seams.ChooseSetupAction = func(_ *cobra.Command, options []setupMenuOption, defaultOption int) (setupAction, error) {
				chooserCalls++
				captured = append([]setupMenuOption(nil), options...)
				if defaultOption != 0 || len(options) != 2 || options[0].Action != setupActionQuick || options[1].Action != setupActionFull {
					t.Fatalf("fresh-install options=%#v default=%d, want quick/full default quick", options, defaultOption)
				}
				return setupActionExit, nil
			}
			seams.RunFullWizard = func(*cobra.Command, bool) error {
				fullCalls++
				return nil
			}

			stdout, stderr, err := runSetupTestCommand(t, seams, args...)
			if err != nil {
				t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
			}
			if chooserCalls != 1 || fullCalls != 0 {
				t.Fatalf("chooserCalls=%d fullCalls=%d, want chooser=1 full=0", chooserCalls, fullCalls)
			}
			for _, want := range []string{"No existing Gormes configuration was found.", "Gormes Agent Setup Wizard"} {
				if !strings.Contains(stdout, want) {
					t.Fatalf("stdout missing %q:\n%s", want, stdout)
				}
			}
			if captured[0].Label != "Quick setup — provider, model & messaging (recommended)" || captured[1].Label != "Full setup — configure everything" {
				t.Fatalf("fresh-install labels = %#v, want Hermes-style Quick/Full wording", captured)
			}
		})
	}
}

func TestSetupQuickPromptsTargetBeforeProviderWork(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: " ", Model: " "},
	}
	seams := fake.seams()
	seams.ChooseSetupTarget = func(_ *cobra.Command, targets []cli.SetupTargetOption, defaultOption int) (cli.SetupTargetID, error) {
		events = append(events, "target")
		if defaultOption != 5 || len(targets) < 6 || targets[5].ID != cli.SetupTargetNavivox {
			t.Fatalf("targets=%#v default=%d, want navivox default", targets, defaultOption)
		}
		return cli.SetupTargetTerminal, nil
	}
	seams.RunSetupProvider = func(*cobra.Command, bool) error {
		events = append(events, "provider")
		fake.current = cli.ProviderModel{Provider: "openai-codex", Model: " "}
		return nil
	}
	seams.RunModelPicker = func(*cobra.Command) error {
		events = append(events, "model")
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

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got, want := strings.Join(events, ","), "target,provider,model,live-test,chat"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
	}
}

func TestSetupQuickNavivoxTargetDoesNotForceProviderSetup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: " ", Model: " "},
	}
	seams := fake.seams()
	seams.ChooseSetupTarget = func(_ *cobra.Command, targets []cli.SetupTargetOption, defaultOption int) (cli.SetupTargetID, error) {
		events = append(events, "target")
		if defaultOption != 5 || len(targets) < 6 || targets[5].ID != cli.SetupTargetNavivox {
			t.Fatalf("targets=%#v default=%d, want navivox default", targets, defaultOption)
		}
		return cli.SetupTargetNavivox, nil
	}
	seams.RunGatewayPlatform = func(_ *cobra.Command, platform string) error {
		events = append(events, "channel:"+platform)
		if platform != string(cli.SetupTargetNavivox) {
			t.Fatalf("RunGatewayPlatform platform = %q, want navivox", platform)
		}
		return nil
	}
	seams.RunSetupProvider = func(*cobra.Command, bool) error {
		t.Fatal("navivox quick target forced provider setup")
		return nil
	}
	seams.RunModelPicker = func(*cobra.Command) error {
		t.Fatal("navivox quick target forced model setup")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		t.Fatal("navivox quick target ran provider live test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("navivox quick setup launched terminal chat")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got, want := strings.Join(events, ","), "target,channel:navivox"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
	}
	for _, want := range []string{
		"Navivox channel setup checked.",
		"Provider/model setup is still required before `gormes gateway` can answer Navivox.",
		"Next setup command: gormes setup provider",
		"After that, start gateway: gormes gateway",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestSetupQuickNavivoxTargetRunsNavivoxEvenWhenAlreadyEnabled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	writeSetupGatewayFixtureConfig(t, `
[navivox]
enabled = true
bind_host = "127.0.0.1"
port = 8765
exposure_mode = "local"
auth_mode = "pairing_token"
token = "nvbx_test_token"
`)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: " ", Model: " "},
	}
	seams := fake.seams()
	seams.RunGatewayPlatform = func(_ *cobra.Command, platform string) error {
		events = append(events, "channel:"+platform)
		if platform != string(cli.SetupTargetNavivox) {
			t.Fatalf("RunGatewayPlatform platform = %q, want navivox", platform)
		}
		return nil
	}
	seams.RunSetupProvider = func(*cobra.Command, bool) error {
		t.Fatal("navivox quick target forced provider setup")
		return nil
	}
	seams.RunModelPicker = func(*cobra.Command) error {
		t.Fatal("navivox quick target forced model setup")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		t.Fatal("navivox quick target ran provider live test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("navivox quick setup launched terminal chat")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--target", "navivox")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got, want := strings.Join(events, ","), "channel:navivox"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
	}
	for _, want := range []string{
		"Navivox channel setup checked.",
		"Provider/model setup is still required before `gormes gateway` can answer Navivox.",
		"After that, start gateway: gormes gateway",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestSetupQuickNonInteractivePrintsTargetCommands(t *testing.T) {
	prompted := false
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: " ", Model: " "},
	}
	seams := fake.seams()
	seams.ChooseSetupTarget = func(*cobra.Command, []cli.SetupTargetOption, int) (cli.SetupTargetID, error) {
		prompted = true
		return cli.SetupTargetTerminal, nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if prompted {
		t.Fatalf("non-interactive quick setup prompted")
	}
	for _, want := range []string{
		"Quick setup targets:",
		"gormes setup --quick --target terminal",
		"gormes setup --quick --target telegram",
		"gormes whatsapp --plan",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestSetupQuickLiveTestFailureRedactsSecretsAndDoesNotLaunchChat(t *testing.T) {
	home := t.TempDir()
	secret := "sk-live-test-secret"
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", secret)
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(`
[hermes]
provider = "openai"
endpoint = "https://api.openai.com/v1"
model = "gpt-4o-mini"
api_key = "`+secret+`"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(config.EnvPath(), []byte("GORMES_API_KEY="+secret+"\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: "openai", Model: "gpt-4o-mini"},
	}
	seams := fake.seams()
	seams.ChooseSetupTarget = func(*cobra.Command, []cli.SetupTargetOption, int) (cli.SetupTargetID, error) {
		return cli.SetupTargetTerminal, nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		return fmt.Errorf("provider rejected credential %s", secret)
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("chat launched after live-test failure")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick")
	if err == nil {
		t.Fatalf("Execute() error = nil, want live-test failure stdout=%s stderr=%s", stdout, stderr)
	}
	for _, want := range []string{"Provider live test failed. Chat was not opened.", "Repair:"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, output := range []string{stdout, stderr, err.Error()} {
		if strings.Contains(output, secret) {
			t.Fatalf("live-test failure leaked secret in %q\nstdout=%s\nstderr=%s\nerr=%v", output, stdout, stderr, err)
		}
	}
}

func TestSetupQuickNonInteractiveTerminalTargetPrintsHandoffWithoutLaunchingChat(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: "openai-codex", Model: "gpt-5.5"},
	}
	seams := fake.seams()
	seams.RunSetupProvider = func(*cobra.Command, bool) error {
		events = append(events, "provider")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		events = append(events, "live-test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("non-interactive terminal quick setup launched chat")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--non-interactive", "--target", "terminal")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Terminal chat ready. Start chatting with: gormes") {
		t.Fatalf("stdout missing terminal handoff:\n%s", stdout)
	}
	if got, want := strings.Join(events, ","), "provider,live-test"; got != want {
		t.Fatalf("events = %s, want %s", got, want)
	}
}

func TestSetupQuickInvalidExplicitTargetFailsFast(t *testing.T) {
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: "openai-codex", Model: "gpt-5.5"},
	}
	seams := fake.seams()
	seams.RunSetupProvider = func(*cobra.Command, bool) error {
		t.Fatal("invalid target ran provider setup")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		t.Fatal("invalid target ran live test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("invalid target launched chat")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--target", "telegrm")
	if err == nil {
		t.Fatalf("Execute() error = nil, want invalid target stdout=%s stderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code != 2 {
		t.Fatalf("exit code = %d, want 2 err=%v stdout=%s stderr=%s", code, err, stdout, stderr)
	}
	if !strings.Contains(err.Error(), "setup_target_invalid_selection: telegrm") {
		t.Fatalf("err = %v, want invalid target detail", err)
	}
}

func TestSetupFirstRunRouterShowsConditionalMigrations(t *testing.T) {
	home := t.TempDir()
	hermes := filepath.Join(home, ".hermes")
	openclaw := filepath.Join(home, ".openclaw")
	var captured []setupMenuOption
	fake := &setupCommandFakeSeams{
		isTTY:        true,
		freshInstall: true,
		detectHermes: func() string {
			return hermes
		},
		detectOpenClaw: func() string {
			return openclaw
		},
	}
	seams := fake.seams()
	seams.ChooseSetupAction = func(_ *cobra.Command, options []setupMenuOption, defaultOption int) (setupAction, error) {
		captured = append([]setupMenuOption(nil), options...)
		if defaultOption != 0 {
			t.Fatalf("defaultOption=%d, want quick default", defaultOption)
		}
		return setupActionExit, nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams)
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if len(captured) != 4 {
		t.Fatalf("captured options=%#v, want quick/full/hermes/openclaw", captured)
	}
	wantActions := []setupAction{setupActionQuick, setupActionFull, setupActionMigrateHermes, setupActionMigrateOpenClaw}
	for i, want := range wantActions {
		if captured[i].Action != want {
			t.Fatalf("option %d action=%s, want %s (options=%#v)", i, captured[i].Action, want, captured)
		}
	}
	for _, want := range []string{"No existing Gormes configuration was found.", "Gormes Agent Setup Wizard"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, want := range []setupAction{setupActionMigrateHermes, setupActionMigrateOpenClaw} {
		found := false
		for _, option := range captured {
			if option.Action == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("captured options missing %s: %#v", want, captured)
		}
	}
}

func TestSetupEntryMode_ResetWritesDefaultsBeforeHeadlessGuidance(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "")
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte("[hermes]\nprovider = 'openai'\nendpoint = 'https://provider.example/v1'\nmodel = 'custom-model'\n"), 0o600); err != nil {
		t.Fatalf("write old config: %v", err)
	}
	fake := &setupCommandFakeSeams{isTTY: false}

	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "--reset", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Configuration reset to defaults.") || !strings.Contains(stdout, "Available setup sections:") {
		t.Fatalf("stdout missing reset + headless guidance:\n%s", stdout)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("load reset config: %v", err)
	}
	if cfg.Hermes.Provider != "" || cfg.Hermes.Endpoint != "" || cfg.Hermes.Model != "hermes-agent" {
		t.Fatalf("reset cfg Hermes = %+v, want built-in defaults", cfg.Hermes)
	}
}

func TestSetupEntryMode_NoTTYAndNonInteractiveNeverPrompt(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		tty  bool
	}{
		{name: "no tty", tty: false},
		{name: "non interactive flag", args: []string{"--non-interactive"}, tty: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &setupCommandFakeSeams{isTTY: tc.tty}
			seams := fake.seams()
			seams.ChooseSetupAction = func(*cobra.Command, []setupMenuOption, int) (setupAction, error) {
				t.Fatal("headless setup prompted")
				return setupActionExit, nil
			}

			stdout, stderr, err := runSetupTestCommand(t, seams, tc.args...)
			if err != nil {
				t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
			}
			if !strings.Contains(stdout, "Available setup sections:") {
				t.Fatalf("stdout missing headless guidance:\n%s", stdout)
			}
		})
	}
}

func TestSetupSections_HermesOwnedVsGormesOwned(t *testing.T) {
	for _, section := range []string{"model", "tts", "terminal", "gateway", "telegram", "tools", "agent"} {
		if got := setupSectionOwnership(section); got != "hermes_owned" {
			t.Fatalf("setupSectionOwnership(%q) = %q, want hermes_owned", section, got)
		}
	}
	for _, section := range []string{"provider", "workspace", "bindings"} {
		if got := setupSectionOwnership(section); got != "gormes_owned_extension" {
			t.Fatalf("setupSectionOwnership(%q) = %q, want gormes_owned_extension", section, got)
		}
	}
	if got := setupSectionOwnership("bogus"); got != "unknown" {
		t.Fatalf("setupSectionOwnership(bogus) = %q, want unknown", got)
	}
}

func TestSetupQuickWhatsAppTargetRoutesThroughWhatsAppSetupSeam(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "sk-whatsapp-quick-test")
	writeSetupGatewayFixtureConfig(t, `
[hermes]
provider = "openai"
endpoint = "https://api.openai.com/v1"
model = "gpt-4o-mini"
`)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: "openai", Model: "gpt-4o-mini"},
	}
	seams := fake.seams()
	seams.RunWhatsAppSetup = func(*cobra.Command) error {
		events = append(events, "whatsapp")
		return nil
	}
	seams.RunGatewayPlatform = func(*cobra.Command, string) error {
		t.Fatal("quick WhatsApp target used generic gateway platform setup")
		return nil
	}
	seams.RunSetupGateway = func(*cobra.Command, bool) error {
		t.Fatal("quick WhatsApp target started gateway setup section")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		events = append(events, "live-test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("quick WhatsApp target launched chat/TUI")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--target", "whatsapp")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got, want := strings.Join(events, ","), "whatsapp,live-test"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
	}
	if !strings.Contains(stdout, "Channel setup checked. Start messaging with: gormes gateway") {
		t.Fatalf("stdout missing channel handoff:\n%s", stdout)
	}
	for _, path := range []string{
		config.SessionDBPath(),
		config.MemoryDBPath(),
		config.GatewayRuntimeStatusPath(),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("quick WhatsApp target opened runtime/startup artifact %s\nstdout=%s", path, stdout)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat runtime/startup artifact %s: %v", path, err)
		}
	}
}

func TestSetupQuickNonInteractiveWhatsAppTargetPrintsCommandWithoutPairing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "sk-whatsapp-noninteractive-test")
	writeSetupGatewayFixtureConfig(t, `
[hermes]
provider = "openai"
endpoint = "https://api.openai.com/v1"
model = "gpt-4o-mini"
`)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   false,
		current: cli.ProviderModel{Provider: "openai", Model: "gpt-4o-mini"},
	}
	seams := fake.seams()
	seams.RunWhatsAppSetup = func(*cobra.Command) error {
		t.Fatal("non-interactive quick WhatsApp launched pairing setup")
		return nil
	}
	seams.RunGatewayPlatform = func(*cobra.Command, string) error {
		t.Fatal("non-interactive quick WhatsApp used generic gateway platform setup")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		events = append(events, "live-test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("non-interactive quick WhatsApp launched chat/TUI")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--non-interactive", "--target", "whatsapp")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"WhatsApp setup command: gormes whatsapp --plan",
		"Channel setup checked. Start messaging with: gormes gateway",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if got, want := strings.Join(events, ","), "live-test"; got != want {
		t.Fatalf("events = %s, want %s", got, want)
	}
}

func TestSetupQuickTelegramTargetStartsChannelSetupBeforeProviderOnFreshInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearSetupGatewayTelegramEnv(t)
	t.Setenv("GORMES_API_KEY", "")

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: " ", Model: " "},
	}
	seams := fake.seams()
	seams.RunTelegramGatewayWizard = func(_ *cobra.Command, _ config.TelegramCfg) (setupTelegramGatewayAnswers, error) {
		events = append(events, "telegram-bubbletea")
		return setupTelegramGatewayAnswers{
			Token:        "123456:abcdefghijklmnopqrstuvwxyzABCDE",
			AccessPolicy: "pairing",
			Apply:        true,
		}, nil
	}
	seams.RunGatewayPlatform = func(_ *cobra.Command, platform string) error {
		t.Fatalf("quick Telegram target used legacy gateway platform setup for %q", platform)
		return nil
	}
	seams.RunSetupProvider = func(*cobra.Command, bool) error {
		t.Fatal("quick Telegram target sent operator to provider before channel setup")
		return nil
	}
	seams.RunModelPicker = func(*cobra.Command) error {
		t.Fatal("quick Telegram target sent operator to model before channel setup")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		t.Fatal("quick Telegram target ran provider live test before provider setup")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("quick Telegram target launched chat/TUI")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--target", "telegram")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got, want := strings.Join(events, ","), "telegram-bubbletea"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
	}
	for _, want := range []string{
		"Review Telegram gateway changes",
		"GORMES_TELEGRAM_BOT_TOKEN=[REDACTED]",
		"Telegram gateway channel configured.",
		"Telegram channel setup checked.",
		"Provider/model setup is still required before `gormes gateway` can answer Telegram.",
		"Next setup command: gormes setup provider",
		"After that, start gateway: gormes gateway",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{
		"Quick Setup - configure missing items only",
		"Provider endpoint or auth is missing.",
		"Model/provider defaults are missing.",
		"Provider live test failed",
		"Terminal chat ready",
	} {
		if strings.Contains(stdout, forbidden) || strings.Contains(stderr, forbidden) {
			t.Fatalf("fresh Telegram quick setup printed forbidden %q\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
		}
	}
}

func TestSetupQuickNonInteractiveTelegramTargetPrintsCommandWithoutPrompting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "sk-telegram-noninteractive-test")
	clearSetupGatewayTelegramEnv(t)
	t.Setenv("GORMES_TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_CHAT_ID", "")
	t.Setenv("TELEGRAM_HOME_CHANNEL", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	writeSetupGatewayFixtureConfig(t, `
[hermes]
provider = "openai"
endpoint = "https://api.openai.com/v1"
model = "gpt-4o-mini"
`)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   false,
		current: cli.ProviderModel{Provider: "openai", Model: "gpt-4o-mini"},
	}
	seams := fake.seams()
	seams.RunGatewayPlatform = func(*cobra.Command, string) error {
		t.Fatal("non-interactive quick Telegram prompted for platform setup")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		events = append(events, "live-test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("non-interactive quick Telegram launched chat/TUI")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--non-interactive", "--target", "telegram")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Channel setup command: gormes setup gateway",
		"Channel setup checked. Start messaging with: gormes gateway",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if got, want := strings.Join(events, ","), "live-test"; got != want {
		t.Fatalf("events = %s, want %s", got, want)
	}
}

func TestSetupQuickTelegramTargetStillRunsChannelSetupWhenCoreMissingButTokenExists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearSetupGatewayTelegramEnv(t)
	writeSetupGatewayFixtureConfig(t, `
[telegram]
bot_token = "123456:ready-token-ready-token-ready-token"
allowed_chat_id = 4242
`)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: " ", Model: " "},
	}
	seams := fake.seams()
	seams.RunTelegramGatewayWizard = func(_ *cobra.Command, cfg config.TelegramCfg) (setupTelegramGatewayAnswers, error) {
		events = append(events, "telegram-bubbletea")
		if cfg.BotToken == "" || cfg.AllowedChatID != 4242 {
			t.Fatalf("Telegram cfg = %+v, want existing token/chat carried into Bubble Tea setup", cfg)
		}
		return setupTelegramGatewayAnswers{
			AccessPolicy: "pairing",
			Apply:        true,
		}, nil
	}
	seams.RunGatewayPlatform = func(_ *cobra.Command, platform string) error {
		t.Fatalf("quick Telegram target used legacy gateway platform setup for %q", platform)
		return nil
	}
	seams.RunSetupProvider = func(*cobra.Command, bool) error {
		t.Fatal("configured Telegram target sent operator to provider before channel setup")
		return nil
	}
	seams.RunModelPicker = func(*cobra.Command) error {
		t.Fatal("configured Telegram target sent operator to model before channel setup")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		t.Fatal("configured Telegram target ran provider live test before provider setup")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("configured Telegram target launched chat/TUI")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--target", "telegram")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got, want := strings.Join(events, ","), "telegram-bubbletea"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
	}
	for _, want := range []string{
		"Review Telegram gateway changes",
		"Telegram gateway channel configured.",
		"Telegram channel setup checked.",
		"Provider/model setup is still required before `gormes gateway` can answer Telegram.",
		"Next setup command: gormes setup provider",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{"Quick Setup - configure missing items only", "Provider endpoint or auth is missing.", "Model/provider defaults are missing."} {
		if strings.Contains(stdout, forbidden) || strings.Contains(stderr, forbidden) {
			t.Fatalf("configured Telegram quick setup printed forbidden %q\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
		}
	}
}

func TestSetupQuickTelegramTargetSkipsConfiguredChannelSetup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "sk-telegram-ready-test")
	clearSetupGatewayTelegramEnv(t)
	t.Setenv("GORMES_TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_CHAT_ID", "")
	t.Setenv("TELEGRAM_HOME_CHANNEL", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	writeSetupGatewayFixtureConfig(t, `
[hermes]
provider = "openai"
endpoint = "https://api.openai.com/v1"
model = "gpt-4o-mini"

[telegram]
bot_token = "123456:ready-token"
allowed_chat_id = 4242
`)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: "openai", Model: "gpt-4o-mini"},
	}
	seams := fake.seams()
	seams.RunGatewayPlatform = func(*cobra.Command, string) error {
		t.Fatal("quick Telegram target prompted for already configured platform setup")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		events = append(events, "live-test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("quick Telegram target launched chat/TUI")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--target", "telegram")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout, "Channel setup command:") {
		t.Fatalf("stdout included channel setup guidance for configured Telegram:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Channel setup checked. Start messaging with: gormes gateway") {
		t.Fatalf("stdout missing channel handoff:\n%s", stdout)
	}
	if got, want := strings.Join(events, ","), "live-test"; got != want {
		t.Fatalf("events = %s, want %s", got, want)
	}
}

func TestSetupGatewaySectionRoutesThroughGatewaySeam(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	gatewayCalls := 0
	fake := &setupCommandFakeSeams{isTTY: true}
	fake.runSetupGateway = func(cmd *cobra.Command, nonInteractive bool) error {
		gatewayCalls++
		if nonInteractive {
			t.Fatal("interactive gateway setup was marked non-interactive")
		}
		cmd.Println("gateway seam reached")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if gatewayCalls != 1 {
		t.Fatalf("RunSetupGateway calls = %d, want 1", gatewayCalls)
	}
	if !strings.Contains(stdout, "gateway seam reached") {
		t.Fatalf("stdout missing gateway seam output:\n%s", stdout)
	}
}
