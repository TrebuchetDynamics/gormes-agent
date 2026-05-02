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

func TestSetupNoSectionPrintsSectionList(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams())
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Available setup sections:", "provider", "model", "agent", "workspace", "bindings", "tts", "terminal", "gateway", "tools", "gormes setup provider"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if fake.modelPickerCalls != 0 || fake.loadedCurrent != 0 {
		t.Fatalf("setup without section invoked work: picker=%d loadCurrent=%d", fake.modelPickerCalls, fake.loadedCurrent)
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

func TestSetupOtherSectionsReturnUnsupported(t *testing.T) {
	for _, section := range []string{"tts", "terminal", "gateway", "tools", "unknown"} {
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

func TestSetupAgentWorkspaceBindingsSectionsAreImplemented(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	for _, tc := range []struct {
		section string
		want    []string
	}{
		{section: "agent", want: []string{"Agent setup in non-interactive mode", "gormes agent reset"}},
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

func TestSetupResetReconfigureRejected(t *testing.T) {
	for _, args := range [][]string{{"--reset"}, {"--reconfigure"}, {"model", "--reset"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			fake := &setupCommandFakeSeams{isTTY: true}
			stdout, stderr, err := runSetupTestCommand(t, fake.seams(), args...)
			if err == nil {
				t.Fatalf("Execute() error = nil, want full wizard unsupported stdout=%s stderr=%s", stdout, stderr)
			}
			if code := exitCodeFromError(err); code != 2 {
				t.Fatalf("exit code = %d, want 2 (err=%v)", code, err)
			}
			for _, want := range []string{"setup_full_wizard_unsupported", "gormes config edit", "gormes auth add"} {
				if !strings.Contains(stdout+stderr+err.Error(), want) {
					t.Fatalf("missing %q stdout=%s stderr=%s err=%v", want, stdout, stderr, err)
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
