package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

func TestRootFreshInteractiveLaunchRoutesToFirstRunSetup(t *testing.T) {
	freshInstallE2EHome(t)

	var setupCalls int
	var tuiCalls int
	cmd := newRootCommandWithRuntime(rootRuntime{
		isTTY: func() bool { return true },
		runFirstRunSetup: func(*cobra.Command) error {
			setupCalls++
			return nil
		},
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
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
	freshInstallE2EHome(t)

	var setupCalls int
	var tuiCalls int
	cmd := newRootCommandWithRuntime(rootRuntime{
		isTTY: func() bool { return false },
		runFirstRunSetup: func(*cobra.Command) error {
			setupCalls++
			return nil
		},
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
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
	freshInstallE2EHome(t)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
provider = "openai"
endpoint = "https://api.openai.com/v1"
model = "gpt-4o-mini"
api_key = "sk-test-ready"
`))

	var setupCalls int
	var tuiCalls int
	cmd := newRootCommandWithRuntime(rootRuntime{
		isTTY: func() bool { return true },
		runFirstRunSetup: func(*cobra.Command) error {
			setupCalls++
			return nil
		},
		runResolvedTUI: func(_ *cobra.Command, invocation tuiInvocation) error {
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
			freshInstallE2EHome(t)
			var setupCalls int
			var tuiCalls int
			cmd := newRootCommandWithRuntime(rootRuntime{
				isTTY: func() bool { return true },
				runFirstRunSetup: func(*cobra.Command) error {
					setupCalls++
					return nil
				},
				runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
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

func TestRootFirstRunDoesNotAffectScriptedChat(t *testing.T) {
	freshInstallE2EHome(t)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
provider = "openai"
endpoint = "https://api.openai.com/v1"
model = "gpt-4o-mini"
api_key = "sk-test-oneshot"
`))

	var setupCalls int
	var tuiCalls int
	var gotPrompt string
	cmd := newRootCommandWithRuntime(rootRuntime{
		isTTY: func() bool { return true },
		runFirstRunSetup: func(*cobra.Command) error {
			setupCalls++
			return nil
		},
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
			tuiCalls++
			return nil
		},
		newOneshotClient: func(context.Context, config.Config, oneshotInvocation) (llm.Client, error) {
			t.Fatal("newOneshotClient should not be called when runOneshot is injected")
			return nil, nil
		},
		runOneshot: func(_ *cobra.Command, invocation oneshotInvocation) error {
			gotPrompt = invocation.Prompt
			return nil
		},
	})

	stdout, stderr, err := executeRootCommandForTest(cmd, "chat", "-q", "hello")
	if err != nil {
		t.Fatalf("chat -q: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if gotPrompt != "hello" {
		t.Fatalf("scripted chat prompt = %q, want hello", gotPrompt)
	}
	if setupCalls != 0 {
		t.Fatalf("runFirstRunSetup calls = %d, want 0", setupCalls)
	}
	if tuiCalls != 0 {
		t.Fatalf("runResolvedTUI calls = %d, want 0", tuiCalls)
	}
}

func TestRootFirstRunSetupCommandUsesCurrentSetupEntrypoint(t *testing.T) {
	freshInstallE2EHome(t)

	cmd := &cobra.Command{Use: "gormes"}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := runFirstRunSetupCommand(cmd)
	if err != nil {
		t.Fatalf("runFirstRunSetupCommand: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Available setup sections:") {
		t.Fatalf("stdout = %q, want setup command with no args to render current setup entrypoint", stdout.String())
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

	if got := detectHermesMigrationSource(); got != envHermes {
		t.Fatalf("detectHermesMigrationSource() = %q, want HERMES_HOME path %q", got, envHermes)
	}

	plan := buildFirstRunPlanFromConfig(config.Config{}, cli.SetupTargetTerminal, false)
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
