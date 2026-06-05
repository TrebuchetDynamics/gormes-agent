package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/spf13/cobra"
)

// TestOnboardCommand_JSONEmitsStructuredFirstRunStatus proves the internal
// onboarding readiness helper returns a parseable `{build, home, config_path,
// skills_root, skills_local, skills_bundled, provider_configured, provider,
// endpoint, model, auth_configured, agents: [...], bindings: [...]}` document
// without scraping the multi-line "Home: / Config: / Provider:" prose.
// Build provenance leads — same convention as the rest of the `--json`
// arc. Secrets stay out: API keys live in the env file, not in this
// report; only their *presence* is signalled via `auth_configured`.
func TestOnboardCommand_JSONEmitsStructuredFirstRunStatus(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")

	writeOneshotFlagConfig(t, []byte(`
[hermes]
provider = "anthropic"
endpoint = "https://api.anthropic.com"
model = "claude-sonnet-4-5"
api_key = "sk-ant-fixture-token"
`))

	stdout, stderr, err := executeOnboardCommandWithSeams(t, onboardCommandSeams{}, "--json")
	if err != nil {
		t.Fatalf("onboard --json: %v\nstderr=%s", err, stderr)
	}

	// Raw API key MUST never leak into stdout.
	if strings.Contains(stdout, "sk-ant-fixture-token") {
		t.Fatalf("onboard --json LEAKED the api key:\nstdout=%s", stdout)
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Home               string `json:"home"`
		ConfigPath         string `json:"config_path"`
		SkillsRoot         string `json:"skills_root"`
		SkillsLocal        int    `json:"skills_local"`
		SkillsBundled      int    `json:"skills_bundled"`
		ProviderConfigured bool   `json:"provider_configured"`
		Provider           string `json:"provider"`
		Endpoint           string `json:"endpoint"`
		Model              string `json:"model"`
		AuthConfigured     bool   `json:"auth_configured"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("onboard --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Home != config.GormesHome() {
		t.Errorf("home = %q, want %q", got.Home, config.GormesHome())
	}
	if !got.ProviderConfigured {
		t.Errorf("provider_configured = false, want true (endpoint+model set)")
	}
	if got.Provider != "anthropic" {
		t.Errorf("provider = %q, want anthropic", got.Provider)
	}
	if got.Model != "claude-sonnet-4-5" {
		t.Errorf("model = %q, want claude-sonnet-4-5", got.Model)
	}
	if !got.AuthConfigured {
		t.Errorf("auth_configured = false, want true (api_key was set)")
	}
}

func TestOnboardStatusPrintsFirstRunNextCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")

	cmd := newOnboardCommandWithSeams(onboardCommandSeams{})
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("onboard: %v\nstderr=%s", err, stderr.String())
	}

	for _, want := range []string{
		"First-run readiness: setup needed",
		"not ready: provider endpoint is not configured",
		"Provider: provider endpoint is not configured",
		"Next: gormes setup --quick --target terminal",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestOnboardWizardInteractivePromptsForStepActions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")

	var prompted []string
	var defaults []string
	seams := onboardCommandSeams{
		IsTTY: func() bool { return true },
		PromptAction: func(_ *cobra.Command, step cli.OnboardStep, defaultAction string) (string, error) {
			prompted = append(prompted, step.ID)
			defaults = append(defaults, defaultAction)
			return "skip", nil
		},
		RunAction: func(_ *cobra.Command, step cli.OnboardStep) error {
			t.Fatalf("RunAction called for skipped step %q", step.ID)
			return nil
		},
	}

	stdout, stderr, err := executeOnboardCommandWithSeams(t, seams, "--wizard")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}

	wantIDs := []string{
		cli.OnboardStepModel,
		cli.OnboardStepProvider,
		cli.OnboardStepAuth,
		cli.OnboardStepGateway,
		cli.OnboardStepBrowser,
		cli.OnboardStepSkills,
		cli.OnboardStepDashboard,
	}
	if !reflect.DeepEqual(prompted, wantIDs) {
		t.Fatalf("prompted steps = %v, want %v\nstdout=%s", prompted, wantIDs, stdout)
	}
	if defaults[1] != "run" {
		t.Fatalf("provider default action = %q, want run; defaults=%v", defaults[1], defaults)
	}
	for _, want := range []string{
		"Gormes first-run wizard",
		"Mode: interactive action runner",
		"1. Model:",
		"2. Provider: missing",
		"3. Auth: missing",
		"4. Gateway: missing",
		"5. Browser/CDP: missing",
		"6. Skills: available",
		"7. Dashboard: available",
		"Action for Model",
		"Action for Dashboard",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestOnboardWizardSkipWarningsBeforeSkip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")

	seams := onboardCommandSeams{
		IsTTY: func() bool { return true },
		PromptAction: func(_ *cobra.Command, _ cli.OnboardStep, _ string) (string, error) {
			return "skip", nil
		},
	}

	stdout, stderr, err := executeOnboardCommandWithSeams(t, seams, "--wizard")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	warningLine := "Skip warning: Skipping provider setup means agents cannot make provider-backed turns until endpoint and model settings are added."
	skipLine := "skip_warning: step=provider Skipping provider setup means agents cannot make provider-backed turns until endpoint and model settings are added."
	warningAt := strings.Index(stdout, warningLine)
	skipAt := strings.Index(stdout, skipLine)
	if warningAt < 0 || skipAt < 0 {
		t.Fatalf("stdout missing provider skip warning/order evidence:\n%s", stdout)
	}
	if warningAt > skipAt {
		t.Fatalf("provider skip warning printed after skip action:\n%s", stdout)
	}
}

func TestOnboardWizardConfiguredStepsArePrefilled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")
	writeOnboardWizardConfig(t, `
[hermes]
provider = "groq"
endpoint = "https://api.groq.com/openai/v1"
model = "llama-3.3-70b-versatile"

[telegram]
bot_token = "123456:test-token"
allowed_chat_id = 4242

[browser]
cdp_url = "http://127.0.0.1:9222"
`)
	if err := os.WriteFile(filepath.Join(home, ".env"), []byte("GORMES_API_KEY=sk-onboard-test\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	defaults := map[string]string{}
	seams := onboardCommandSeams{
		IsTTY: func() bool { return true },
		PromptAction: func(_ *cobra.Command, step cli.OnboardStep, defaultAction string) (string, error) {
			defaults[step.ID] = defaultAction
			return "review", nil
		},
	}

	stdout, stderr, err := executeOnboardCommandWithSeams(t, seams, "--wizard")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, id := range []string{cli.OnboardStepModel, cli.OnboardStepProvider, cli.OnboardStepAuth, cli.OnboardStepGateway, cli.OnboardStepBrowser} {
		if defaults[id] != "review" {
			t.Fatalf("default action for %s = %q, want review; defaults=%v", id, defaults[id], defaults)
		}
	}
	for _, want := range []string{
		"Model: configured",
		"Provider: configured",
		"Auth: configured",
		"Gateway: configured",
		"Browser/CDP: configured",
		"review: model current=configured command=\"gormes setup model\"",
		"review: gateway current=configured command=\"gormes setup gateway\"",
		"llama-3.3-70b-versatile",
		"groq",
		"telegram",
		"http://127.0.0.1:9222",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestOnboardWizardDelegatesSelectedActions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")

	var called []string
	seams := onboardCommandSeams{
		IsTTY: func() bool { return true },
		PromptAction: func(_ *cobra.Command, _ cli.OnboardStep, _ string) (string, error) {
			return "run", nil
		},
		RunAction: func(cmd *cobra.Command, step cli.OnboardStep) error {
			called = append(called, step.ID)
			cmd.Printf("onboard_action_delegated: step=%s command=%q\n", step.ID, step.NextCommand)
			return nil
		},
	}

	stdout, stderr, err := executeOnboardCommandWithSeams(t, seams, "--wizard")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	wantIDs := []string{
		cli.OnboardStepModel,
		cli.OnboardStepProvider,
		cli.OnboardStepAuth,
		cli.OnboardStepGateway,
		cli.OnboardStepBrowser,
		cli.OnboardStepSkills,
		cli.OnboardStepDashboard,
	}
	if !reflect.DeepEqual(called, wantIDs) {
		t.Fatalf("called steps = %v, want %v\nstdout=%s", called, wantIDs, stdout)
	}
	for _, want := range []string{
		"onboard_action_delegated: step=model command=\"gormes setup model\"",
		"onboard_action_delegated: step=provider command=\"gormes setup provider\"",
		"onboard_action_delegated: step=auth command=\"gormes auth add <provider>\"",
		"onboard_action_delegated: step=gateway command=\"gormes setup gateway\"",
		"onboard_action_delegated: step=browser command=\"gormes doctor --offline\"",
		"onboard_action_delegated: step=skills command=\"gormes skills list\"",
		"onboard_action_delegated: step=dashboard command=\"gormes dashboard\"",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestOnboardWizardRowBackedActionEvidence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")

	seams := onboardCommandSeams{
		IsTTY: func() bool { return true },
		PromptAction: func(_ *cobra.Command, _ cli.OnboardStep, _ string) (string, error) {
			return "run", nil
		},
	}

	stdout, stderr, err := executeOnboardCommandWithSeams(t, seams, "--wizard")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"onboard_action_row_backed: step=model recommended_command=\"gormes setup model\"",
		"onboard_action_row_backed: step=provider recommended_command=\"gormes setup provider\"",
		"onboard_action_row_backed: step=auth recommended_command=\"gormes auth add <provider>\"",
		"onboard_action_row_backed: step=gateway recommended_command=\"gormes setup gateway\"",
		"onboard_action_row_backed: step=browser recommended_command=\"gormes doctor --offline\"",
		"onboard_action_row_backed: step=skills recommended_command=\"gormes skills list\"",
		"onboard_action_row_backed: step=dashboard recommended_command=\"gormes dashboard\"",
		"Onboarding wizard finished.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestOnboardWizardNonInteractiveStillDoesNotPrompt(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		tty  bool
	}{
		{name: "flag", args: []string{"--wizard", "--non-interactive"}, tty: true},
		{name: "non tty", args: []string{"--wizard"}, tty: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("GORMES_HOME", home)
			t.Setenv("GORMES_SKILLS_ROOT", "")
			t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")

			seams := onboardCommandSeams{
				IsTTY: func() bool { return tc.tty },
				PromptAction: func(_ *cobra.Command, step cli.OnboardStep, _ string) (string, error) {
					t.Fatalf("PromptAction called for noninteractive step %q", step.ID)
					return "", nil
				},
				RunAction: func(_ *cobra.Command, step cli.OnboardStep) error {
					t.Fatalf("RunAction called for noninteractive step %q", step.ID)
					return nil
				},
			}

			stdout, stderr, err := executeOnboardCommandWithSeams(t, seams, tc.args...)
			if err != nil {
				t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
			}
			for _, want := range []string{
				"Mode: non-interactive plan",
				"Interactive action prompting is available in a TTY",
			} {
				if !strings.Contains(stdout, want) {
					t.Fatalf("stdout missing %q:\n%s", want, stdout)
				}
			}
			for _, forbidden := range []string{
				"Mode: interactive action runner",
				"Action for Model",
				"onboard_action_row_backed",
			} {
				if strings.Contains(stdout, forbidden) {
					t.Fatalf("stdout contains interactive text %q:\n%s", forbidden, stdout)
				}
			}
		})
	}
}

func executeOnboardCommandWithSeams(t *testing.T, seams onboardCommandSeams, args ...string) (string, string, error) {
	t.Helper()
	cmd := newOnboardCommandWithSeams(seams)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func writeOnboardWizardConfig(t *testing.T, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config home: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
