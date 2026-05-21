package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/spf13/cobra"
)

type setupCommandFakeSeams struct {
	isTTY        bool
	freshInstall bool
	current      cli.ProviderModel

	modelPickerCalls               int
	activeProviderModelPickerCalls int
	activeProviderModelPickerInput cli.ProviderModel
	loadedCurrent                  int

	chooseSetupAction              func(*cobra.Command, []setupMenuOption, int) (setupAction, error)
	chooseSetupTarget              func(*cobra.Command, []cli.SetupTargetOption, int) (cli.SetupTargetID, error)
	chooseSetupProvider            func(*cobra.Command, []cli.ProviderMenuEntry, int) (int, error)
	chooseProviderCredentialAction func(*cobra.Command, setupProviderCredentialPrompt) (setupProviderCredentialAction, error)
	runSetupProvider               func(*cobra.Command, bool) error
	runProviderLiveTest            func(*cobra.Command) error
	runProviderAuth                func(*cobra.Command, string) error
	runActiveProviderModelPicker   func(*cobra.Command, cli.ProviderModel) error
	runFullWizard                  func(*cobra.Command, bool) error
	runSetupGateway                func(*cobra.Command, bool) error
	runGatewaySetupWizard          func(*cobra.Command, config.Config) (setupGatewayWizardResult, error)
	runGatewayPlatform             func(*cobra.Command, string) error
	runWhatsAppSetup               func(*cobra.Command) error
	providerAuthStatus             cli.ProviderAuthStatus
	detectHermes                   func() string
	detectOpenClaw                 func() string
}

func (f *setupCommandFakeSeams) seams() setupCommandSeams {
	if f.current.Provider == "" {
		f.current.Provider = "openai-codex"
	}
	if f.current.Model == "" {
		f.current.Model = "gpt-5.5"
	}
	existingInstall := !f.freshInstall
	detectHermes := f.detectHermes
	if detectHermes == nil {
		detectHermes = func() string { return "" }
	}
	detectOpenClaw := f.detectOpenClaw
	if detectOpenClaw == nil {
		detectOpenClaw = func() string { return "" }
	}
	return setupCommandSeams{
		IsTTY: func() bool { return f.isTTY },
		HasExistingInstall: func() (bool, error) {
			return existingInstall, nil
		},
		RunModelPicker: func(cmd *cobra.Command) error {
			f.modelPickerCalls++
			return nil
		},
		RunActiveProviderModelPicker: func(cmd *cobra.Command, current cli.ProviderModel) error {
			f.activeProviderModelPickerCalls++
			f.activeProviderModelPickerInput = current
			if f.runActiveProviderModelPicker != nil {
				return f.runActiveProviderModelPicker(cmd, current)
			}
			return nil
		},
		LoadCurrentModel: func() (cli.ProviderModel, error) {
			f.loadedCurrent++
			return f.current, nil
		},
		ChooseSetupAction:              f.chooseSetupAction,
		ChooseSetupTarget:              f.chooseSetupTarget,
		ChooseSetupProvider:            f.chooseSetupProvider,
		ChooseProviderCredentialAction: f.chooseProviderCredentialAction,
		RunSetupProvider:               f.runSetupProvider,
		RunProviderLiveTest:            f.runProviderLiveTest,
		RunProviderAuth:                f.runProviderAuth,
		RunFullWizard:                  f.runFullWizard,
		RunSetupGateway:                f.runSetupGateway,
		RunGatewaySetupWizard:          firstSetupGatewayWizardSeam(f.runGatewaySetupWizard),
		RunGatewayPlatform:             f.runGatewayPlatform,
		RunWhatsAppSetup:               f.runWhatsAppSetup,
		LoadProviderAuthStatus: func(_ string) (cli.ProviderAuthStatus, error) {
			status := f.providerAuthStatus
			if status.Provider == "" {
				status.Provider = f.current.Provider
			}
			if status.AuthType == "" {
				status.AuthType = "oauth_external"
			}
			if status.Status == "" {
				status.Status = cli.AuthStatusLoggedIn
				status.Authenticated = true
			}
			return status, nil
		},
		DetectHermesMigrationSource:   detectHermes,
		DetectOpenClawMigrationSource: detectOpenClaw,
	}
}

func runSetupTestCommand(t *testing.T, seams setupCommandSeams, args ...string) (string, string, error) {
	t.Helper()
	return runSetupTestCommandWithInput(t, seams, "", args...)
}

func runSetupTestCommandWithInput(t *testing.T, seams setupCommandSeams, input string, args ...string) (string, string, error) {
	t.Helper()
	cmd := newSetupCommandWithSeams(seams)
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

func firstSetupGatewayWizardSeam(fn func(*cobra.Command, config.Config) (setupGatewayWizardResult, error)) func(*cobra.Command, config.Config) (setupGatewayWizardResult, error) {
	if fn != nil {
		return fn
	}
	return func(cmd *cobra.Command, cfg config.Config) (setupGatewayWizardResult, error) {
		options := setupGatewayPlatformOptions(cfg)
		selection, err := promptString(cmd, "Messaging platforms (comma-separated numbers or ids, blank to keep current): ", "")
		if err != nil {
			return setupGatewayWizardResult{}, err
		}
		selected, keepCurrent, err := parseSetupGatewaySelection(selection, options)
		if err != nil {
			return setupGatewayWizardResult{}, err
		}
		if keepCurrent {
			return setupGatewayWizardResult{}, nil
		}
		return setupGatewayWizardResult{SelectedPlatforms: selected}, nil
	}
}

func TestSetupProviderChoiceTextIgnoresArrowEscapeNoise(t *testing.T) {
	cmd := &cobra.Command{}
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetIn(strings.NewReader("\x1b[B\n"))

	got, err := promptSetupProviderChoiceText(cmd, []cli.ProviderMenuEntry{
		{ID: "nous", Label: "Nous Portal"},
		{ID: "openai-codex", Label: "OpenAI Codex"},
	}, 1)
	if err != nil {
		t.Fatalf("promptSetupProviderChoiceText() error = %v stdout=%s", err, stdout.String())
	}
	if got != 1 {
		t.Fatalf("promptSetupProviderChoiceText() = %d, want default index 1", got)
	}
}

func TestSetupNoSectionNonTTYPrintsSectionList(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams())
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Available setup sections:", "provider", "model", "agent", "workspace", "bindings", "tts", "terminal", "gateway", "tools", "gormes setup provider"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "What would you like to do?") {
		t.Fatalf("non-TTY setup launched interactive menu:\n%s", stdout)
	}
	if fake.modelPickerCalls != 0 || fake.loadedCurrent != 0 {
		t.Fatalf("setup without section invoked work: picker=%d loadCurrent=%d", fake.modelPickerCalls, fake.loadedCurrent)
	}
}

func TestSetupNoSectionFreshInstallShowsQuickFullChoice(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	var captured []setupMenuOption
	var defaultIndex int
	fake := &setupCommandFakeSeams{isTTY: true, freshInstall: true}
	fake.chooseSetupAction = func(_ *cobra.Command, options []setupMenuOption, defaultOption int) (setupAction, error) {
		captured = append([]setupMenuOption(nil), options...)
		defaultIndex = defaultOption
		return setupActionExit, nil
	}

	stdout, stderr, err := runSetupTestCommand(t, fake.seams())
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if defaultIndex != 0 {
		t.Fatalf("default menu index = %d, want Quick Setup at 0", defaultIndex)
	}
	for _, want := range []string{
		"No existing Gormes configuration was found.",
		"Gormes Agent Setup Wizard",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if len(captured) != 2 || captured[0].Action != setupActionQuick || captured[1].Action != setupActionFull {
		t.Fatalf("captured menu = %#v, want Quick/Full menu", captured)
	}
	if captured[0].Label != "Quick setup — provider, model & messaging (recommended)" || captured[1].Label != "Full setup — configure everything" {
		t.Fatalf("captured labels = %#v, want Hermes-style Quick/Full wording", captured)
	}
	if _, err := os.Stat(config.ConfigPath()); !os.IsNotExist(err) {
		t.Fatalf("exit-only setup mutated config path %s: %v", config.ConfigPath(), err)
	}
	if fake.modelPickerCalls != 0 || fake.loadedCurrent != 0 {
		t.Fatalf("exit-only setup invoked work: picker=%d loadCurrent=%d", fake.modelPickerCalls, fake.loadedCurrent)
	}
}

func TestSetupTopLevelFullWizardRoutesThroughWizardAndSummary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fullCalls := 0
	fake := &setupCommandFakeSeams{isTTY: true}
	fake.chooseSetupAction = func(_ *cobra.Command, _ []setupMenuOption, _ int) (setupAction, error) {
		return setupActionFull, nil
	}
	fake.runFullWizard = func(cmd *cobra.Command, nonInteractive bool) error {
		fullCalls++
		if nonInteractive {
			t.Fatal("top-level interactive full wizard was marked non-interactive")
		}
		printSetupSummary(cmd)
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, fake.seams())
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if fullCalls != 1 {
		t.Fatalf("RunFullWizard calls = %d, want 1", fullCalls)
	}
	for _, want := range []string{
		"Setup Complete",
		"📁 All your files are in",
		"Settings:", config.ConfigPath(),
		"API Keys:", config.EnvPath(),
		"Data:", filepath.Join(home, "cron") + "/, sessions/, logs/",
		"📝 To edit your configuration:",
		"gormes setup          Re-run the full wizard",
		"gormes config set <key> <value>\n                          Set a specific value",
		"Or edit the files directly:",
		"nano " + config.ConfigPath(),
		"nano " + config.EnvPath(),
		"🚀 Ready to go!",
		"gormes doctor",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{".hermes", "config.yaml", "hermes setup", "hermes doctor"} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("stdout contains Hermes-owned summary text %q:\n%s", forbidden, stdout)
		}
	}
}

func TestSetupFullWizardOffersGormesLaunchPromptAfterSummary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	fake.chooseSetupAction = func(_ *cobra.Command, _ []setupMenuOption, _ int) (setupAction, error) {
		return setupActionFull, nil
	}
	input := strings.Repeat("\n", 10) + "n\n"
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), input)
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Setup Complete", "🚀 Ready to go!", "Launch gormes chat now? [Y/n]: "} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "Launch hermes chat now?") {
		t.Fatalf("stdout contains Hermes-owned launch prompt:\n%s", stdout)
	}
}

func TestSetupFullWizardPrintsReconfigureAndProviderPrelude(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_INSTALL_HOME", home)

	fake := &setupCommandFakeSeams{
		isTTY: true,
		current: cli.ProviderModel{
			Provider: "openai-codex",
			Model:    "gpt-5.5",
		},
	}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "40\n"+strings.Repeat("\n", 12)+"n\n")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"◆ Reconfigure",
		"✓ You already have Gormes configured.",
		"Running the full wizard - each prompt shows your current value.",
		"Tip: jump straight to a section with 'gormes setup model|fallback|",
		"◆ Configuration Location",
		"Config file:  " + config.ConfigPath(),
		"Secrets file: " + config.EnvPath(),
		"Data folder:  " + config.GormesHome(),
		"Install dir:  " + filepath.Join(home, "gormes-agent"),
		"You can edit these files directly or use 'gormes config edit'",
		"◆ Inference Provider",
		"Guide: https://hermes-agent.nousresearch.com/docs/integrations/providers",
		"Current model:    gpt-5.5",
		"Active provider:  OpenAI Codex",
		"No change.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, want := range []string{
		"Select provider:",
		"1. Nous Portal (Nous Research subscription)",
		"6. OpenAI Codex  ← currently active",
		"39. Configure auxiliary models...",
		"40. Leave unchanged",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing provider picker row %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "OpenAI Codex credentials:") {
		t.Fatalf("setup showed Codex credential submenu before provider selection:\n%s", stdout)
	}
	if fake.modelPickerCalls != 0 || fake.activeProviderModelPickerCalls != 0 {
		t.Fatalf("picker calls after Leave unchanged: general=%d active=%d, want none", fake.modelPickerCalls, fake.activeProviderModelPickerCalls)
	}
}

func TestSetupFullWizardReauthenticateRunsAuthBeforeModelPicker(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY: true,
		current: cli.ProviderModel{
			Provider: "openai-codex",
			Model:    "gpt-5.5",
		},
	}
	fake.chooseSetupProvider = func(_ *cobra.Command, entries []cli.ProviderMenuEntry, defaultIndex int) (int, error) {
		if defaultIndex != 5 {
			t.Fatalf("default index = %d, want 5", defaultIndex)
		}
		return 5, nil
	}
	fake.chooseProviderCredentialAction = func(_ *cobra.Command, _ setupProviderCredentialPrompt) (setupProviderCredentialAction, error) {
		return setupProviderCredentialReauthenticate, nil
	}
	fake.runProviderAuth = func(_ *cobra.Command, provider string) error {
		events = append(events, "auth:"+provider)
		return nil
	}
	seams := fake.seams()
	seams.RunActiveProviderModelPicker = func(_ *cobra.Command, current cli.ProviderModel) error {
		if current.Provider != "openai-codex" {
			t.Fatalf("active model picker provider = %q, want openai-codex", current.Provider)
		}
		events = append(events, "model")
		return nil
	}

	stdout, stderr, err := runSetupTestCommandWithInput(t, seams, strings.Repeat("\n", 12)+"n\n")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got, want := strings.Join(events, ","), "auth:openai-codex,model"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
	}
	if !strings.Contains(stdout, "Starting a fresh OpenAI Codex login...") {
		t.Fatalf("stdout missing reauth handoff:\n%s", stdout)
	}
}

func TestSetupFullWizardSwitchingProviderUsesSelectedProviderDefaultModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{
		isTTY: true,
		current: cli.ProviderModel{
			Provider: "openai-codex",
			Model:    "gpt-5.5",
		},
	}
	fake.chooseSetupProvider = func(_ *cobra.Command, entries []cli.ProviderMenuEntry, _ int) (int, error) {
		for i, entry := range entries {
			if entry.ID == "openrouter" {
				return i, nil
			}
		}
		t.Fatalf("provider menu missing openrouter: %#v", entries)
		return -1, nil
	}
	seams := fake.seams()
	seams.RunActiveProviderModelPicker = func(_ *cobra.Command, current cli.ProviderModel) error {
		if current.Provider != "openrouter" {
			t.Fatalf("active model picker provider = %q, want openrouter", current.Provider)
		}
		if current.Model != "moonshotai/kimi-k2.6" {
			t.Fatalf("active model picker model seed = %q, want OpenRouter default moonshotai/kimi-k2.6", current.Model)
		}
		return nil
	}

	stdout, stderr, err := runSetupTestCommandWithInput(t, seams, strings.Repeat("\n", 12)+"n\n")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Selected provider: OpenRouter") {
		t.Fatalf("stdout missing selected-provider confirmation:\n%s", stdout)
	}
}

func TestSetupActiveProviderModelPickerSkipsProviderList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := config.WriteTOMLValue(config.ConfigPath(), "hermes.provider", "openai-codex"); err != nil {
		t.Fatalf("write provider: %v", err)
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "hermes.model", "gpt-5.5"); err != nil {
		t.Fatalf("write model: %v", err)
	}

	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("\n"))

	err := runSetupActiveProviderModelPicker(cmd, cli.ProviderModel{Provider: "openai-codex", Model: "gpt-5.5"})
	if err != nil {
		t.Fatalf("runSetupActiveProviderModelPicker() error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	for _, want := range []string{"Select model for openai-codex:", "1. gpt-5.5", "Choice [1-8] (1), custom model, or q to cancel:", "model selection saved: provider=openai-codex model=gpt-5.5"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "Suggested models for openai-codex:") {
		t.Fatalf("stdout still uses comma-separated model suggestions:\n%s", stdout.String())
	}
	for _, forbidden := range []string{"Choose a provider", "github-models (api_key)", "openai-codex (oauth_external)"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("stdout contains provider-list artifact %q:\n%s", forbidden, stdout.String())
		}
	}
}

func TestSetupModelSectionDelegatesToPicker(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "model")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if fake.modelPickerCalls != 1 {
		t.Fatalf("model picker calls = %d, want 1", fake.modelPickerCalls)
	}
	if !strings.Contains(stdout, "Gormes Setup — Model") {
		t.Fatalf("stdout missing boxed model-section header:\n%s", stdout)
	}
}

func TestSetupModelSectionCancelIsClean(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: true}
	seams := fake.seams()
	seams.RunModelPicker = func(*cobra.Command) error {
		return fmt.Errorf("gormes model: %w", cli.ErrModelPickerCancelled)
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "model")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Setup cancelled.") {
		t.Fatalf("stdout missing clean cancel message:\n%s", stdout)
	}
	if strings.Contains(stdout+stderr, "model_picker_cancelled") {
		t.Fatalf("setup leaked internal cancel sentinel stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestSetupModelDefaultPickerDoesNotInheritParentSetupArg(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	oldArgs := os.Args
	os.Args = []string{"gormes", "setup"}
	t.Cleanup(func() { os.Args = oldArgs })

	stdout, stderr, err := runSetupTestCommand(t, setupCommandSeams{
		IsTTY: func() bool { return true },
	}, "model")
	if err == nil {
		t.Fatalf("Execute() error = nil, want model picker TTY failure in test process stdout=%s stderr=%s", stdout, stderr)
	}
	if strings.Contains(stdout+stderr+err.Error(), `unknown command "setup" for "model"`) {
		t.Fatalf("setup model leaked parent argv into model command stdout=%s stderr=%s err=%v", stdout, stderr, err)
	}
	if !strings.Contains(err.Error(), cli.ErrModelPickerRequiresTTY.Error()) {
		t.Fatalf("error = %v, want model picker TTY failure after argv isolation stdout=%s stderr=%s", err, stdout, stderr)
	}
}

func TestSetupMenuIgnoresArrowEscapeNoiseBeforeSelection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fullCalls := 0
	fake := &setupCommandFakeSeams{isTTY: true, freshInstall: true}
	fake.runFullWizard = func(_ *cobra.Command, nonInteractive bool) error {
		fullCalls++
		if nonInteractive {
			t.Fatal("interactive full setup selection was marked non-interactive")
		}
		return nil
	}

	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "\x1b[B\x1b[A2\n")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if fullCalls != 1 {
		t.Fatalf("full setup calls = %d, want 1 after selecting option 2", fullCalls)
	}
	if strings.Contains(stdout, "↑/↓ arrows") {
		// This is expected in the new interactive menu mode.
	}
}

func TestSetupProviderNonInteractiveWritesConfigAndDotenv(t *testing.T) {
	home := t.TempDir()
	secret := "sk-test-secret-7890"
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_ENDPOINT", "https://provider.example/v1")
	t.Setenv("GORMES_API_KEY", secret)
	t.Setenv("GORMES_MODEL", "provider-fixture-model")

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "provider", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Gormes Setup — Provider", "Provider configured.", config.ConfigPath(), config.EnvPath(), "provider-fixture-model", "API key:  stored (redacted)", "Test it:  gormes chat"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, leaked := range []string{secret, "sk-t***7890", "sk-t", "7890"} {
		if strings.Contains(stdout+stderr, leaked) {
			t.Fatalf("setup output leaked API key material %q:\nstdout=%s\nstderr=%s", leaked, stdout, stderr)
		}
	}
	configBody, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	for _, want := range []string{`endpoint = 'https://provider.example/v1'`, `model = 'provider-fixture-model'`} {
		if !strings.Contains(string(configBody), want) {
			t.Fatalf("config missing %q:\n%s", want, string(configBody))
		}
	}
	if strings.Contains(string(configBody), secret) || strings.Contains(string(configBody), "api_key") {
		t.Fatalf("config.toml leaked secret material:\n%s", string(configBody))
	}
	envBody, err := os.ReadFile(config.EnvPath())
	if err != nil {
		t.Fatalf("read env: %v", err)
	}
	if !strings.Contains(string(envBody), "GORMES_API_KEY="+secret) {
		t.Fatalf(".env missing API key entry:\n%s", string(envBody))
	}
}

func TestSetupProviderInteractiveWritesSelectedProvider(t *testing.T) {
	home := t.TempDir()
	secret := "sk-openrouter-secret-7890"
	t.Setenv("GORMES_HOME", home)
	withOpenRouterModelCatalogFetcherForTest(t, openRouterModelCatalogOfflineForTest)
	if err := config.WriteTOMLValue(config.ConfigPath(), "hermes.provider", "openrouter"); err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "hermes.model", "moonshotai/kimi-k2.6"); err != nil {
		t.Fatalf("seed model: %v", err)
	}

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "\n"+secret+"\n\n", "provider")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Select provider:",
		"OpenRouter (100+ models, pay-per-use)  ← currently active",
		"Select model for openrouter:",
		"1. anthropic/claude-opus-4.7",
		"4. moonshotai/kimi-k2.6",
		"31. inclusionai/ring-2.6-1t:free",
		"Provider: openrouter",
		"Endpoint: https://openrouter.ai/api/v1",
		"Model:    moonshotai/kimi-k2.6",
		"API key:  stored (redacted)",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "Provider [openai]") {
		t.Fatalf("setup provider used stale free-text provider prompt:\n%s", stdout)
	}
	for _, leaked := range []string{secret, "sk-o***7890", "sk-openrouter", "7890"} {
		if strings.Contains(stdout+stderr, leaked) {
			t.Fatalf("setup output leaked API key material %q:\nstdout=%s\nstderr=%s", leaked, stdout, stderr)
		}
	}
	configBody, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	for _, want := range []string{`provider = 'openrouter'`, `endpoint = 'https://openrouter.ai/api/v1'`, `model = 'moonshotai/kimi-k2.6'`} {
		if !strings.Contains(string(configBody), want) {
			t.Fatalf("config missing %q:\n%s", want, string(configBody))
		}
	}
	if strings.Contains(string(configBody), secret) || strings.Contains(string(configBody), "api_key") {
		t.Fatalf("config.toml leaked secret material:\n%s", string(configBody))
	}
	envBody, err := os.ReadFile(config.EnvPath())
	if err != nil {
		t.Fatalf("read env: %v", err)
	}
	if !strings.Contains(string(envBody), "GORMES_API_KEY="+secret) {
		t.Fatalf(".env missing API key entry:\n%s", string(envBody))
	}
}

func TestSetupProviderOpenAICodexUsesOAuthFlow(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY: true,
		current: cli.ProviderModel{
			Provider: "openrouter",
			Model:    "moonshotai/kimi-k2.6",
		},
		providerAuthStatus: cli.ProviderAuthStatus{
			Provider:      "openai-codex",
			AuthType:      "oauth_external",
			Status:        cli.AuthStatusLoggedOut,
			Authenticated: false,
		},
	}
	fake.chooseSetupProvider = func(_ *cobra.Command, entries []cli.ProviderMenuEntry, _ int) (int, error) {
		for i, entry := range entries {
			if entry.ID == "openai-codex" {
				return i, nil
			}
		}
		t.Fatalf("provider menu missing openai-codex: %#v", entries)
		return -1, nil
	}
	fake.runProviderAuth = func(_ *cobra.Command, provider string) error {
		events = append(events, "auth:"+provider)
		return nil
	}
	seams := fake.seams()
	seams.RunActiveProviderModelPicker = func(_ *cobra.Command, current cli.ProviderModel) error {
		events = append(events, "model:"+current.Provider)
		return nil
	}

	stdout, stderr, err := runSetupTestCommandWithInput(t, seams, "6\n", "provider")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got, want := strings.Join(events, ","), "auth:openai-codex,model:openai-codex"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
	}
	for _, want := range []string{
		"Selected provider: OpenAI Codex",
		"Not logged into OpenAI Codex. Starting login...",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "API key:") {
		t.Fatalf("OpenAI Codex setup must not ask for an API key:\n%s", stdout)
	}
	if _, err := os.Stat(config.EnvPath()); !os.IsNotExist(err) {
		t.Fatalf("Codex OAuth setup should not write GORMES_API_KEY env path %s: %v", config.EnvPath(), err)
	}
}

func TestSetupProviderAnthropicUsesOAuthFlow(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY: true,
		current: cli.ProviderModel{
			Provider: "openrouter",
			Model:    "moonshotai/kimi-k2.6",
		},
		providerAuthStatus: cli.ProviderAuthStatus{
			Provider:      config.AnthropicProvider,
			AuthType:      config.CredentialAuthOAuth,
			Status:        cli.AuthStatusLoggedOut,
			Authenticated: false,
		},
	}
	fake.chooseSetupProvider = func(_ *cobra.Command, entries []cli.ProviderMenuEntry, _ int) (int, error) {
		for i, entry := range entries {
			if entry.ID == config.AnthropicProvider {
				return i, nil
			}
		}
		t.Fatalf("provider menu missing anthropic: %#v", entries)
		return -1, nil
	}
	fake.runProviderAuth = func(_ *cobra.Command, provider string) error {
		events = append(events, "auth:"+provider)
		return nil
	}
	seams := fake.seams()
	seams.RunActiveProviderModelPicker = func(_ *cobra.Command, current cli.ProviderModel) error {
		events = append(events, "model:"+current.Provider)
		return nil
	}

	stdout, stderr, err := runSetupTestCommandWithInput(t, seams, "2\n", "provider")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got, want := strings.Join(events, ","), "auth:anthropic,model:anthropic"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
	}
	for _, want := range []string{
		"Selected provider: Anthropic",
		"Not logged into Anthropic. Starting login...",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "API key:") {
		t.Fatalf("Anthropic OAuth setup must not ask for an API key:\n%s", stdout)
	}
	if _, err := os.Stat(config.EnvPath()); !os.IsNotExist(err) {
		t.Fatalf("Anthropic OAuth setup should not write GORMES_API_KEY env path %s: %v", config.EnvPath(), err)
	}
}

func TestSetupProviderDoesNotPrintShortSecretMaterial(t *testing.T) {
	home := t.TempDir()
	secret := "abcd"
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_ENDPOINT", "https://provider.example/v1")
	t.Setenv("GORMES_API_KEY", secret)

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "provider", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "API key:  stored (redacted)") {
		t.Fatalf("stdout missing redacted key confirmation:\n%s", stdout)
	}
	for _, leaked := range []string{secret, "abcd***abcd", "abc", "bcd"} {
		if strings.Contains(stdout+stderr, leaked) {
			t.Fatalf("setup output leaked short API key material %q:\nstdout=%s\nstderr=%s", leaked, stdout, stderr)
		}
	}
	envBody, err := os.ReadFile(config.EnvPath())
	if err != nil {
		t.Fatalf("read env: %v", err)
	}
	if !strings.Contains(string(envBody), "GORMES_API_KEY="+secret) {
		t.Fatalf(".env missing API key entry:\n%s", string(envBody))
	}
}

func TestPromptSecretUsesNoEchoTerminalReader(t *testing.T) {
	secret := "sk-terminal-secret-7890"
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open devnull: %v", err)
	}
	defer stdin.Close()

	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetIn(stdin)
	cmd.SetOut(&stdout)

	oldReadPassword := setupReadPassword
	oldInputIsTerminal := setupInputIsTerminal
	setupReadPassword = func(fd int) ([]byte, error) {
		if fd != int(stdin.Fd()) {
			t.Fatalf("read password fd = %d, want %d", fd, stdin.Fd())
		}
		return []byte(secret), nil
	}
	setupInputIsTerminal = func(file *os.File) bool {
		return file == stdin
	}
	t.Cleanup(func() {
		setupReadPassword = oldReadPassword
		setupInputIsTerminal = oldInputIsTerminal
	})

	got, err := promptSecret(cmd, "API key: ")
	if err != nil {
		t.Fatalf("promptSecret: %v", err)
	}
	if got != secret {
		t.Fatalf("promptSecret returned %q, want secret", got)
	}
	if out := stdout.String(); !strings.Contains(out, "API key: ") || strings.Contains(out, secret) {
		t.Fatalf("stdout = %q, want prompt and no raw secret", out)
	}
}

func TestSetupProviderRequiresTTYWithoutNonInteractive(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "provider")
	if !errors.Is(err, errSetupRequiresTTY) {
		t.Fatalf("Execute() error = %v, want errSetupRequiresTTY stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stderr+err.Error(), "setup_requires_tty") || !strings.Contains(stderr, "gormes setup provider --non-interactive") {
		t.Fatalf("missing provider TTY guidance stdout=%s stderr=%s err=%v", stdout, stderr, err)
	}
	if fake.modelPickerCalls != 0 || fake.loadedCurrent != 0 {
		t.Fatalf("non-tty setup invoked work: picker=%d loadCurrent=%d", fake.modelPickerCalls, fake.loadedCurrent)
	}
}

func TestSetupUnknownSectionReturnsUnsupported(t *testing.T) {
	for _, section := range []string{"unknown"} {
		t.Run(section, func(t *testing.T) {
			fake := &setupCommandFakeSeams{isTTY: true}
			stdout, stderr, err := runSetupTestCommand(t, fake.seams(), section)
			if err == nil {
				t.Fatalf("Execute() error = nil, want unsupported section stdout=%s stderr=%s", stdout, stderr)
			}
			if code := exitCodeFromError(err); code != 2 {
				t.Fatalf("exit code = %d, want 2 (err=%v)", code, err)
			}
			for _, want := range []string{"setup_section_unsupported", setupSectionList()} {
				if !strings.Contains(stdout+stderr+err.Error(), want) {
					t.Fatalf("missing %q stdout=%s stderr=%s err=%v", want, stdout, stderr, err)
				}
			}
			if fake.modelPickerCalls != 0 {
				t.Fatalf("unsupported section invoked picker %d times", fake.modelPickerCalls)
			}
		})
	}
}

func TestSetupHermesParitySectionsAreImplementedNonInteractive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	for _, tc := range []struct {
		section string
		want    []string
	}{
		{section: "tts", want: []string{"Text-to-Speech Provider", "Edge TTS", "OpenAI TTS", "Keep current"}},
		{section: "terminal", want: []string{"Terminal Backend", "Local", "Docker", "Modal", "SSH", "Daytona", "Singularity/Apptainer", "Keep current"}},
		{section: "gateway", want: []string{"Messaging Platforms", "Telegram", "Discord", "Slack", "gormes gateway"}},
		{section: "tools", want: []string{"Tools for CLI", "Web Search & Scraping", "Browser Automation", "Terminal & Processes", "File Operations"}},
		{section: "agent", want: []string{"Agent Settings", "Max iterations", "Tool progress mode", "Compression threshold", "Session reset policy"}},
	} {
		t.Run(tc.section, func(t *testing.T) {
			fake := &setupCommandFakeSeams{isTTY: false}
			stdout, stderr, err := runSetupTestCommand(t, fake.seams(), tc.section, "--non-interactive")
			if err != nil {
				t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
			}
			for _, want := range tc.want {
				if !strings.Contains(stdout, want) {
					t.Fatalf("stdout missing %q:\n%s", want, stdout)
				}
			}
			if strings.Contains(stdout+stderr, "setup_section_unsupported") {
				t.Fatalf("implemented section returned unsupported evidence:\nstdout=%s\nstderr=%s", stdout, stderr)
			}
		})
	}
}

func TestSetupAgentSettingsInteractivePersistsRuntimeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "200\n4\n0.75\ndaily\n", "agent")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Max iterations set to 200",
		"Select tool progress mode:",
		"3. all",
		"4. verbose",
		"Choice [1-4] (3), or q to cancel:",
		"Tool progress set to: verbose",
		"Compression threshold set to 0.75",
		"Session reset policy set to: daily",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	body, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	for _, want := range []string{
		"max_tool_iterations = 200",
		"compression_threshold = 0.75",
		"session_reset_policy = 'daily'",
		"tool_progress = 'verbose'",
	} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("config missing %q:\n%s", want, string(body))
		}
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := configuredMaxToolIterations(cfg); got != 200 {
		t.Fatalf("configuredMaxToolIterations = %d, want 200", got)
	}
}

func TestSetupAgentSettingsToolProgressRejectsNonOption(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "200\nhi\n0.75\ndaily\n", "agent")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Select tool progress mode:",
		"Choice [1-4] (3), or q to cancel:",
		"Unknown tool progress mode \"hi\"; keeping all",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{
		"Tool progress set to: hi",
		"tool_progress = 'hi'",
		"setup_agent_value_ignored: tool_progress",
	} {
		if strings.Contains(stdout+stderr, forbidden) {
			t.Fatalf("output contains forbidden %q:\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
		}
	}
	body, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(body), "tool_progress = 'hi'") {
		t.Fatalf("config persisted invalid tool_progress:\n%s", string(body))
	}
}

func TestSetupAgentWorkspaceBindingsSectionsAreImplemented(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	for _, tc := range []struct {
		section string
		want    []string
	}{
		{section: "agent", want: []string{"Agent Settings", "Max iterations", "Tool progress mode"}},
		{section: "workspace", want: []string{"Workspace setup in non-interactive mode", home + "/workspace", "[agents.defaults] workspace"}},
		{section: "bindings", want: []string{"Bindings setup in non-interactive mode", "[[bindings]]", "channel = \"telegram\""}},
	} {
		t.Run(tc.section, func(t *testing.T) {
			fake := &setupCommandFakeSeams{isTTY: false}
			stdout, stderr, err := runSetupTestCommand(t, fake.seams(), tc.section, "--non-interactive")
			if err != nil {
				t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
			}
			for _, want := range tc.want {
				if !strings.Contains(stdout, want) {
					t.Fatalf("stdout missing %q:\n%s", want, stdout)
				}
			}
			if fake.modelPickerCalls != 0 || fake.loadedCurrent != 0 {
				t.Fatalf("guidance section invoked model work: picker=%d loadCurrent=%d", fake.modelPickerCalls, fake.loadedCurrent)
			}
		})
	}
}

func TestSetupReconfigureRunsFullWizard(t *testing.T) {
	fullCalls := 0
	fake := &setupCommandFakeSeams{isTTY: true}
	seams := fake.seams()
	seams.RunFullWizard = func(cmd *cobra.Command, nonInteractive bool) error {
		fullCalls++
		if nonInteractive {
			t.Fatal("interactive reconfigure was marked non-interactive")
		}
		printSetupSummary(cmd)
		return nil
	}
	stdout, stderr, err := runSetupTestCommand(t, seams, "--reconfigure")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if fullCalls != 1 {
		t.Fatalf("RunFullWizard calls = %d, want 1", fullCalls)
	}
	for _, want := range []string{"Setup Complete", "gormes setup", config.ConfigPath(), config.EnvPath()} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("missing %q stdout=%s stderr=%s", want, stdout, stderr)
		}
	}
}

func TestSetupNonInteractiveUsesDefaults(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: false, current: cli.ProviderModel{Provider: "anthropic", Model: "claude-sonnet-4"}}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "model", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if fake.modelPickerCalls != 0 {
		t.Fatalf("non-interactive setup invoked picker %d times", fake.modelPickerCalls)
	}
	if fake.loadedCurrent != 1 {
		t.Fatalf("LoadCurrentModel calls = %d, want 1", fake.loadedCurrent)
	}
	for _, want := range []string{"setup_model_defaults", "provider=anthropic", "model=claude-sonnet-4"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestSetupModelRequiresTTYWithoutNonInteractive(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "model")
	if !errors.Is(err, errSetupRequiresTTY) {
		t.Fatalf("Execute() error = %v, want errSetupRequiresTTY stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stderr+err.Error(), "setup_requires_tty") || !strings.Contains(stderr, "--non-interactive") {
		t.Fatalf("missing TTY guidance stdout=%s stderr=%s err=%v", stdout, stderr, err)
	}
	if fake.modelPickerCalls != 0 || fake.loadedCurrent != 0 {
		t.Fatalf("non-tty setup invoked work: picker=%d loadCurrent=%d", fake.modelPickerCalls, fake.loadedCurrent)
	}
}
