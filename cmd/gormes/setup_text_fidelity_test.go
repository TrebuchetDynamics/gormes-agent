package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSetupTextFidelity_TopLevelTTYMenuAndLineFallback(t *testing.T) {
	fullCalls := 0
	fake := &setupCommandFakeSeams{isTTY: true, freshInstall: true}
	fake.runFullWizard = func(_ *cobra.Command, nonInteractive bool) error {
		fullCalls++
		if nonInteractive {
			t.Fatal("interactive full setup selection was marked non-interactive")
		}
		return nil
	}

	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "\x1b[B\x1b[A2\n")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"No existing Gormes configuration was found.",
		"How would you like to set up Gormes?",
		"Quick setup - provider, model, and messaging",
		"Full setup - configure everything",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if fullCalls != 1 {
		t.Fatalf("full setup calls = %d, want 1 after escape-noise option 2", fullCalls)
	}
	if strings.Contains(stdout, "\u2191\u2193 navigate") {
		t.Fatalf("line-input fallback advertised unsupported arrow navigation:\n%s", stdout)
	}
}

func TestSetupTextFidelity_RootHelpSetupProviderGatewayPath(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "--help")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"gormes onboard",
		"gormes setup provider",
		"gormes setup model",
		"gormes --oneshot \"hello\"",
		"gormes gateway status",
		"gormes whatsapp",
		"gormes telegram",
		"gormes auth add <provider>",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("root help missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{
		"hermes setup",
		"hermes config edit",
		".hermes",
		"config.yaml",
	} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("root help contains stale Hermes setup/path text %q:\n%s", forbidden, stdout)
		}
	}
}

func TestSetupTextFidelity_UnknownSectionGuidance(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "browser")
	if err == nil {
		t.Fatalf("Execute() error = nil, want unsupported-section guidance stdout=%s stderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code != 2 {
		t.Fatalf("exit code = %d, want 2 (err=%v)", code, err)
	}
	artifact := stdout + stderr + err.Error()
	for _, want := range []string{
		"setup_section_unsupported: section=browser",
		"available=provider|model|agent|workspace|bindings|tts|terminal|gateway|tools",
		"recommended_command=\"gormes setup\"",
		"setup_section_row_backed",
	} {
		if !strings.Contains(artifact, want) {
			t.Fatalf("artifact missing %q:\nstdout=%s\nstderr=%s\nerr=%v", want, stdout, stderr, err)
		}
	}
	if strings.Contains(artifact, `unknown command "browser"`) {
		t.Fatalf("artifact leaked raw Cobra unknown-command text:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
}
