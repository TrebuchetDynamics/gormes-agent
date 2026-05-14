package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
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
			fake := &setupCommandFakeSeams{isTTY: true, freshInstall: true}
			seams := fake.seams()
			seams.ChooseSetupAction = func(_ *cobra.Command, options []setupMenuOption, defaultOption int) (setupAction, error) {
				chooserCalls++
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
			for _, want := range []string{"No existing Gormes configuration was found.", "Quick setup - provider, model, and messaging", "Full setup - configure everything"} {
				if !strings.Contains(stdout, want) {
					t.Fatalf("stdout missing %q:\n%s", want, stdout)
				}
			}
		})
	}
}

func TestSetupQuickPromptsTargetBeforeProviderWork(t *testing.T) {
	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: " ", Model: " "},
	}
	seams := fake.seams()
	seams.ChooseSetupTarget = func(_ *cobra.Command, targets []cli.SetupTargetOption, defaultOption int) (cli.SetupTargetID, error) {
		events = append(events, "target")
		if defaultOption != 0 || len(targets) == 0 || targets[0].ID != cli.SetupTargetTerminal {
			t.Fatalf("targets=%#v default=%d, want terminal default", targets, defaultOption)
		}
		return cli.SetupTargetTerminal, nil
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
	if got, want := strings.Join(events, ","), "target,model,live-test,chat"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
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
		"gormes whatsapp",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
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
	for _, want := range []string{"Migrate Hermes", "Migrate OpenClaw"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
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
	for _, section := range []string{"model", "tts", "terminal", "gateway", "tools", "agent"} {
		if got := setupSectionOwnership(section); got != "hermes_owned" {
			t.Fatalf("setupSectionOwnership(%q) = %q, want hermes_owned", section, got)
		}
	}
	for _, section := range []string{"provider", "workspace", "bindings", "onboard"} {
		if got := setupSectionOwnership(section); got != "gormes_owned_extension" {
			t.Fatalf("setupSectionOwnership(%q) = %q, want gormes_owned_extension", section, got)
		}
	}
	if got := setupSectionOwnership("bogus"); got != "unknown" {
		t.Fatalf("setupSectionOwnership(bogus) = %q, want unknown", got)
	}
}
