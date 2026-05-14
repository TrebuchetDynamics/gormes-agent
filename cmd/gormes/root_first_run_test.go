package main

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
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

func TestRootFirstRunDoesNotAffectOneshot(t *testing.T) {
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
		newOneshotClient: func(context.Context, config.Config, oneshotInvocation) (hermes.Client, error) {
			t.Fatal("newOneshotClient should not be called when runOneshot is injected")
			return nil, nil
		},
		runOneshot: func(_ *cobra.Command, invocation oneshotInvocation) error {
			gotPrompt = invocation.Prompt
			return nil
		},
	})

	stdout, stderr, err := executeRootCommandForTest(cmd, "--oneshot", "hello")
	if err != nil {
		t.Fatalf("root oneshot: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if gotPrompt != "hello" {
		t.Fatalf("oneshot prompt = %q, want hello", gotPrompt)
	}
	if setupCalls != 0 {
		t.Fatalf("runFirstRunSetup calls = %d, want 0", setupCalls)
	}
	if tuiCalls != 0 {
		t.Fatalf("runResolvedTUI calls = %d, want 0", tuiCalls)
	}
}
