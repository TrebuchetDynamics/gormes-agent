package main

import (
	"strings"
	"testing"
)

func TestUsageCommand_RendersUnsupportedProviderWithoutStartingTUI(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "usage", "--provider", "fixture-provider")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	for _, want := range []string{"Provider: fixture-provider", "Usage unavailable: account usage is not supported for provider fixture-provider"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q\nstdout=%s\nstderr=%s", want, stdout, stderr)
		}
	}
	if strings.Contains(stderr, "api_server") {
		t.Fatalf("stderr contains api_server health output:\n%s", stderr)
	}
}
