package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
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
	cmd := newSetupCommandWithSeams(seams)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
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
	for _, want := range []string{"Available setup sections:", "model", "tts", "terminal", "gateway", "tools", "agent"} {
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

func TestSetupOtherSectionsReturnUnsupported(t *testing.T) {
	for _, section := range []string{"tts", "terminal", "gateway", "tools", "agent", "unknown"} {
		t.Run(section, func(t *testing.T) {
			fake := &setupCommandFakeSeams{isTTY: true}
			stdout, stderr, err := runSetupTestCommand(t, fake.seams(), section)
			if err == nil {
				t.Fatalf("Execute() error = nil, want unsupported section stdout=%s stderr=%s", stdout, stderr)
			}
			if code := exitCodeFromError(err); code != 2 {
				t.Fatalf("exit code = %d, want 2 (err=%v)", code, err)
			}
			for _, want := range []string{"setup_section_unsupported", "model|tts|terminal|gateway|tools|agent"} {
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
