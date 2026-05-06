package main

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

func TestUpdateCommandRegistersNativeUpdate(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "update", "--help")
	if err != nil {
		t.Fatalf("update --help: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	for _, want := range []string{"--branch", "--check", "--yes", "--restart-gateway", "--kill-stale-dashboard"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("update help missing %q:\n%s", want, stdout)
		}
	}
}

func TestUpdateCommandUsesInjectedLifecycle(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	var got cli.UpdateLifecycleOptions
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(ctx context.Context, opts cli.UpdateLifecycleOptions) cli.UpdateReport {
			got = opts
			if ctx == nil {
				t.Fatal("RunLifecycle got nil context")
			}
			return cli.UpdateReport{
				Branch: "development",
				Evidence: []cli.UpdateEvidence{
					{Kind: cli.UpdateEvidenceAutostashRestored, Detail: "ok"},
				},
			}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--branch", "development", "--restart-gateway", "always")
	if err != nil {
		t.Fatalf("update command: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	if got.CheckoutDir != "/repo/gormes" || got.Branch != "development" || got.RestartGateway != "always" {
		t.Fatalf("options = %+v, want checkout/branch/restart flags", got)
	}
	if !strings.Contains(stdout, "update branch: development") || !strings.Contains(stdout, "update_autostash_restored") {
		t.Fatalf("stdout missing summary/evidence:\n%s", stdout)
	}
}

func TestUpdateCommandFailureReturnsOperatorEvidence(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport {
			return cli.UpdateReport{
				Branch: "development",
				Failed: true,
				Evidence: []cli.UpdateEvidence{
					{Kind: cli.UpdateEvidenceNetworkError, Detail: "could not fetch origin/development"},
					{Kind: cli.UpdateEvidenceAutostashPreserved, Detail: "stash preserved"},
				},
				OperatorRecovery: "Restore manually with: git stash apply stash-commit-safe",
			}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--branch", "development")
	if err == nil {
		t.Fatalf("update failure returned nil error; stdout=%s stderr=%s", stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	for _, want := range []string{"update_network_error", "update_autostash_preserved", "git stash apply"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("combined output missing %q:\nstdout=%s\nstderr=%s\nerr=%v", want, stdout, stderr, err)
		}
	}
}

func TestUpdateCommandCheckModeSkipsMutation(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	called := false
	command := newUpdateCommandWithSeams(updateCommandSeams{
		CheckoutDir: func() (string, error) { return "/repo/gormes", nil },
		RunLifecycle: func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport {
			called = true
			return cli.UpdateReport{}
		},
	})

	stdout, stderr, err := executeRootCommandForTest(command, "--check", "--branch", "development")
	if err != nil {
		t.Fatalf("update --check: %v stderr=%s", err, stderr)
	}
	if !called {
		t.Fatal("RunLifecycle was not called")
	}
	if !strings.Contains(stdout, "update_check") {
		t.Fatalf("stdout missing check evidence:\n%s", stdout)
	}
}
