package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestGatewayMutatingSubcommandsAreUnavailable(t *testing.T) {
	if len(gatewayMutatingUnavailableSubcommands) == 0 {
		t.Fatalf("no mutating subcommands registered")
	}
	for _, sub := range gatewayMutatingUnavailableSubcommands {
		t.Run(sub, func(t *testing.T) {
			setupGatewayStatusTestEnv(t)

			stdout, stderr, err := executeGatewayMutatingCommand(t, sub)
			if err == nil {
				t.Fatalf("expected error from `gateway %s`; got nil\nstdout=%s\nstderr=%s", sub, stdout, stderr)
			}

			wantMsg := "gateway: " + sub + " is not available; use the service_restart helper"
			if !strings.Contains(err.Error(), wantMsg) {
				t.Fatalf("error %q does not contain %q", err.Error(), wantMsg)
			}

			if code := exitCodeFromError(err); code == 0 {
				t.Fatalf("expected non-zero exit code from `gateway %s`, got %d", sub, code)
			}

			for _, path := range []string{
				config.GatewayRuntimeStatusPath(),
				config.SessionDBPath(),
				config.MemoryDBPath(),
			} {
				if _, statErr := os.Stat(path); statErr == nil {
					t.Fatalf("`gateway %s` opened or created runtime artifact %s", sub, path)
				} else if !os.IsNotExist(statErr) {
					t.Fatalf("stat runtime artifact %s: %v", path, statErr)
				}
			}
		})
	}
}

func TestGatewayMutatingExitCodeIsStable(t *testing.T) {
	codes := make(map[string]int, len(gatewayMutatingUnavailableSubcommands))
	for _, sub := range gatewayMutatingUnavailableSubcommands {
		t.Run(sub, func(t *testing.T) {
			setupGatewayStatusTestEnv(t)
			_, _, err := executeGatewayMutatingCommand(t, sub)
			if err == nil {
				t.Fatalf("expected error from `gateway %s`", sub)
			}
			codes[sub] = exitCodeFromError(err)
		})
	}

	first := gatewayMutatingUnavailableSubcommands[0]
	want := codes[first]
	for _, sub := range gatewayMutatingUnavailableSubcommands[1:] {
		if got := codes[sub]; got != want {
			t.Fatalf("exit code drift: gateway %s = %d but gateway %s = %d", first, want, sub, got)
		}
	}
}

func TestGatewayMutatingDoesNotShadowGatewayStatus(t *testing.T) {
	setupGatewayStatusTestEnv(t)

	stdout, stderr, err := executeGatewayStatusCommand(t)
	if err != nil {
		t.Fatalf("gateway status broken after mutating subcommands registered: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"runtime: missing",
		"channels: none configured",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("gateway status stdout missing %q after mutating subcommands registered:\n%s", want, stdout)
		}
	}
	assertGatewayStatusDidNotOpenRuntimeStores(t)
}

func executeGatewayMutatingCommand(t *testing.T, sub string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"gateway", sub})
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}
