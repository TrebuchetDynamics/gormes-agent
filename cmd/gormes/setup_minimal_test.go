package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/spf13/cobra"
)

type setupCommandFakeSeams struct {
	isTTY   bool
	current cli.ProviderModel

	modelPickerCalls int
	loadedCurrent    int

	chooseSetupAction  func(*cobra.Command, []setupMenuOption, int) (setupAction, error)
	runFullWizard      func(*cobra.Command, bool) error
	runSetupGateway    func(*cobra.Command, bool) error
	runGatewayPlatform func(*cobra.Command, string) error
}

func (f *setupCommandFakeSeams) seams() setupCommandSeams {
	if f.current.Provider == "" {
		f.current.Provider = "openai-codex"
	}
	if f.current.Model == "" {
		f.current.Model = "gpt-5.5"
	}
	return setupCommandSeams{
		IsTTY: func() bool { return f.isTTY },
		RunModelPicker: func(cmd *cobra.Command) error {
			f.modelPickerCalls++
			return nil
		},
		LoadCurrentModel: func() (cli.ProviderModel, error) {
			f.loadedCurrent++
			return f.current, nil
		},
		ChooseSetupAction:  f.chooseSetupAction,
		RunFullWizard:      f.runFullWizard,
		RunSetupGateway:    f.runSetupGateway,
		RunGatewayPlatform: f.runGatewayPlatform,
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

func TestSetupNoSectionInteractiveMenuShowsTopLevelChoices(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	var captured []setupMenuOption
	var defaultIndex int
	fake := &setupCommandFakeSeams{isTTY: true}
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
		"What would you like to do?",
		"Quick Setup - configure missing items only",
		"Full Setup - reconfigure everything",
		"Model & Provider",
		"Terminal Backend",
		"Messaging Platforms (Gateway)",
		"Tools",
		"Agent Settings",
		"Exit",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if len(captured) != 8 || captured[0].Action != setupActionQuick || captured[7].Action != setupActionExit {
		t.Fatalf("captured menu = %#v, want Quick..Exit eight-option menu", captured)
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
	for _, want := range []string{"Setup Complete", "All your files are in", "Settings:", config.ConfigPath(), "API Keys:", config.EnvPath(), "gormes setup", "gormes config edit", "gormes doctor"} {
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

func TestSetupModelSectionDelegatesToPicker(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "model")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if fake.modelPickerCalls != 1 {
		t.Fatalf("model picker calls = %d, want 1", fake.modelPickerCalls)
	}
	if !strings.Contains(stdout, "Setup section: model") {
		t.Fatalf("stdout missing model-section evidence:\n%s", stdout)
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

	gatewayCalls := 0
	fake := &setupCommandFakeSeams{isTTY: true}
	fake.runSetupGateway = func(_ *cobra.Command, nonInteractive bool) error {
		gatewayCalls++
		if nonInteractive {
			t.Fatal("interactive gateway menu selection was marked non-interactive")
		}
		return nil
	}

	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "\x1b[B\x1b[A5\n")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if gatewayCalls != 1 {
		t.Fatalf("gateway setup calls = %d, want 1 after selecting option 5", gatewayCalls)
	}
	if strings.Contains(stdout, "↑↓ navigate") {
		t.Fatalf("line-prompt setup menu advertised unsupported arrow navigation:\n%s", stdout)
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
	for _, want := range []string{"Setup section: provider", "Provider configured.", config.ConfigPath(), config.EnvPath(), "provider-fixture-model", "sk-t***7890"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout+stderr, secret) {
		t.Fatalf("setup output leaked raw API key:\nstdout=%s\nstderr=%s", stdout, stderr)
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
	secret := "sk-groq-secret-7890"
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "groq\n"+secret+"\n\n", "provider")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Provider: groq", "Endpoint: https://api.groq.com/openai/v1", "Model:    llama-3.3-70b-versatile", "sk-g***7890"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout+stderr, secret) {
		t.Fatalf("setup output leaked raw API key:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	configBody, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	for _, want := range []string{`provider = 'groq'`, `endpoint = 'https://api.groq.com/openai/v1'`, `model = 'llama-3.3-70b-versatile'`} {
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
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "200\nverbose\n0.75\ndaily\n", "agent")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Max iterations set to 200", "Tool progress set to: verbose", "Compression threshold set to 0.75", "Session reset policy set to: daily"} {
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

func TestSetupResetReconfigureRunsFullWizard(t *testing.T) {
	for _, args := range [][]string{{"--reset"}, {"--reconfigure"}, {"model", "--reset"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			fake := &setupCommandFakeSeams{isTTY: true}
			stdout, stderr, err := runSetupTestCommand(t, fake.seams(), args...)
			if err != nil {
				t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
			}
			for _, want := range []string{"Setup Complete", "gormes setup", config.ConfigPath(), config.EnvPath()} {
				if !strings.Contains(stdout, want) {
					t.Fatalf("missing %q stdout=%s stderr=%s", want, stdout, stderr)
				}
			}
			if fake.modelPickerCalls != 0 {
				t.Fatalf("reset/reconfigure invoked picker %d times", fake.modelPickerCalls)
			}
		})
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
