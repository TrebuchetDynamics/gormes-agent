package main

import (
	"strings"
	"testing"
)

func TestToolsCommandListShowsRuntimeToolsets(t *testing.T) {
	setupNativeTUITestEnv(t)
	writeSetupToolsFixtureConfig(t, `
platform_toolsets = { cli = ["terminal", "web"] }
`)

	stdout, stderr, err := executeRootCommandForTest(newRootCommand(), "tools", "list")
	if err != nil {
		t.Fatalf("tools list error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Tools for CLI", "web", "enabled", "terminal"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("tools list stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "hermes_command_unavailable") || strings.Contains(stdout, "row-backed") {
		t.Fatalf("tools list still reports row-backed unavailable:\n%s", stdout)
	}
}

func TestToolsCommandEnableDisablePersistsCLISelection(t *testing.T) {
	setupNativeTUITestEnv(t)
	writeSetupToolsFixtureConfig(t, `
platform_toolsets = { cli = ["terminal"] }
`)

	stdout, stderr, err := executeRootCommandForTest(newRootCommand(), "tools", "enable", "web")
	if err != nil {
		t.Fatalf("tools enable error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "enabled: web") || !strings.Contains(stdout, "session_reset_required=true") {
		t.Fatalf("tools enable stdout = %q, want enabled web with reset evidence", stdout)
	}
	got := readCLIPlatformToolsets(t)
	if !containsString(got, "terminal") || !containsString(got, "web") {
		t.Fatalf("toolsets after enable = %v, want terminal and web", got)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommand(), "tools", "disable", "terminal")
	if err != nil {
		t.Fatalf("tools disable error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "disabled: terminal") || !strings.Contains(stdout, "session_reset_required=true") {
		t.Fatalf("tools disable stdout = %q, want disabled terminal with reset evidence", stdout)
	}
	got = readCLIPlatformToolsets(t)
	if containsString(got, "terminal") || !containsString(got, "web") {
		t.Fatalf("toolsets after disable = %v, want web only", got)
	}
}
