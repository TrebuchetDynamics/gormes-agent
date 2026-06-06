package tuiapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

func TestRootFreshInteractiveLaunchRoutesToFirstRunSetup(t *testing.T) {
	setupFirstRunTestHome(t)

	var setupCalls int
	var tuiCalls int
	cmd := newRootCommandWithRuntime(Runtime{
		IsTTY: func() bool { return true },
		RunFirstRunSetup: func(*cobra.Command) error {
			setupCalls++
			return nil
		},
		RunResolvedTUI: func(*cobra.Command, Invocation) error {
			tuiCalls++
			return nil
		},
	})

	stdout, stderr, err := executeRootCommandForTest(cmd)
	if err != nil {
		t.Fatalf("plain root fresh interactive: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if setupCalls != 1 {
		t.Fatalf("runFirstRunSetup calls = %d, want 1", setupCalls)
	}
	if tuiCalls != 0 {
		t.Fatalf("runResolvedTUI calls = %d, want 0", tuiCalls)
	}
}

func TestRootFreshNonTTYPrintsSetupGuidance(t *testing.T) {
	setupFirstRunTestHome(t)

	var setupCalls int
	var tuiCalls int
	cmd := newRootCommandWithRuntime(Runtime{
		IsTTY: func() bool { return false },
		RunFirstRunSetup: func(*cobra.Command) error {
			setupCalls++
			return nil
		},
		RunResolvedTUI: func(*cobra.Command, Invocation) error {
			tuiCalls++
			return nil
		},
	})

	stdout, stderr, err := executeRootCommandForTest(cmd)
	if err != nil {
		t.Fatalf("plain root fresh non-tty: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if setupCalls != 0 {
		t.Fatalf("runFirstRunSetup calls = %d, want 0", setupCalls)
	}
	if tuiCalls != 0 {
		t.Fatalf("runResolvedTUI calls = %d, want 0", tuiCalls)
	}
	for _, want := range []string{
		"Gormes setup needed",
		"Next: gormes setup --quick --target terminal",
		"Non-interactive mode will not prompt.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\nstdout=%s\nstderr=%s", want, stdout, stderr)
		}
	}
}

func TestRootReadyConfigStillOpensTUI(t *testing.T) {
	setupFirstRunTestHome(t)
	writeSetupToolsFixtureConfig(t, `
[hermes]
provider = "openai"
endpoint = "https://api.openai.com/v1"
model = "gpt-4o-mini"
api_key = "sk-test-ready"
`)

	var setupCalls int
	var tuiCalls int
	cmd := newRootCommandWithRuntime(Runtime{
		IsTTY: func() bool { return true },
		RunFirstRunSetup: func(*cobra.Command) error {
			setupCalls++
			return nil
		},
		RunResolvedTUI: func(_ *cobra.Command, invocation Invocation) error {
			tuiCalls++
			if invocation.Config.Hermes.Provider != "openai" {
				t.Fatalf("provider = %q, want openai", invocation.Config.Hermes.Provider)
			}
			return nil
		},
	})

	stdout, stderr, err := executeRootCommandForTest(cmd)
	if err != nil {
		t.Fatalf("plain root ready config: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if setupCalls != 0 {
		t.Fatalf("runFirstRunSetup calls = %d, want 0", setupCalls)
	}
	if tuiCalls != 1 {
		t.Fatalf("runResolvedTUI calls = %d, want 1", tuiCalls)
	}
}

func TestRootFirstRunBypassOfflineAndRemote(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "offline", args: []string{"--offline"}},
		{name: "remote", args: []string{"--remote", "http://127.0.0.1:43827"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupFirstRunTestHome(t)
			var setupCalls int
			var tuiCalls int
			cmd := newRootCommandWithRuntime(Runtime{
				IsTTY: func() bool { return true },
				RunFirstRunSetup: func(*cobra.Command) error {
					setupCalls++
					return nil
				},
				RunResolvedTUI: func(*cobra.Command, Invocation) error {
					tuiCalls++
					return nil
				},
			})

			stdout, stderr, err := executeRootCommandForTest(cmd, tc.args...)
			if err != nil {
				t.Fatalf("root %s: %v\nstdout=%s\nstderr=%s", tc.name, err, stdout, stderr)
			}
			if setupCalls != 0 {
				t.Fatalf("runFirstRunSetup calls = %d, want 0", setupCalls)
			}
			if tuiCalls != 1 {
				t.Fatalf("runResolvedTUI calls = %d, want 1", tuiCalls)
			}
		})
	}
}

func TestDetectHermesMigrationSourcePrefersEnvOverHomeDotHermes(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	homeDotHermes := filepath.Join(home, ".hermes")
	envHermes := filepath.Join(root, "env-hermes")
	if err := os.MkdirAll(homeDotHermes, 0o755); err != nil {
		t.Fatalf("create home .hermes: %v", err)
	}
	if err := os.MkdirAll(envHermes, 0o755); err != nil {
		t.Fatalf("create env hermes: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("HERMES_HOME", envHermes)

	if got := DetectHermesMigrationSource(); got != envHermes {
		t.Fatalf("DetectHermesMigrationSource() = %q, want HERMES_HOME path %q", got, envHermes)
	}

	plan := BuildFirstRunPlanFromConfig(config.Config{}, cli.SetupTargetTerminal, false)
	wantCommand := "gormes migrate hermes --dry-run --source '" + envHermes + "'"
	var found bool
	for _, action := range plan.Actions {
		if action.ID == cli.FirstRunActionMigrateHermes {
			found = true
			if action.Command != wantCommand {
				t.Fatalf("Hermes migration command = %q, want %q", action.Command, wantCommand)
			}
		}
	}
	if !found {
		t.Fatalf("plan missing Hermes migration action: %+v", plan.Actions)
	}
}

func setupFirstRunTestHome(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes-home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes-home"))
	t.Setenv("CODEX_HOME", filepath.Join(root, "codex-home"))
	t.Setenv("GORMES_KANBAN_DB", "")
	t.Setenv("GORMES_KANBAN_HOME", "")
	t.Setenv("GORMES_KANBAN_TASK", "")
	t.Setenv("HERMES_KANBAN_BOARD", "")
	t.Setenv("HERMES_KANBAN_DB", "")
	t.Setenv("GORMES_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	return root
}
