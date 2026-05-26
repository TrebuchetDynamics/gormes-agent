package providers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

func TestProvidersSetupSubcommandPrintsSetupGuidance(t *testing.T) {
	cmd := NewProvidersCommand(Options{})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"setup", "openrouter"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("providers setup openrouter: %v\nstdout=%s", err, stdout.String())
	}
	for _, want := range []string{
		"Provider setup commands for OpenRouter (openrouter):",
		"Status: implemented (openai_chat, api_key)",
		"Interactive: gormes setup provider",
		"GORMES_INFERENCE_PROVIDER=openrouter",
		"OPENROUTER_API_KEY=<token>",
		"gormes auth add openrouter --type api-key --api-key <token>",
		"--inference-url https://openrouter.ai/api/v1",
		"Select model: gormes model",
		"Verify auth: gormes auth status openrouter",
		"Verify runtime: gormes doctor --offline",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestProvidersSetupSubcommandSupportsEveryManifestProvider(t *testing.T) {
	for _, entry := range hermes.HermesProviderRegistryManifest() {
		t.Run(entry.ID, func(t *testing.T) {
			cmd := NewProvidersCommand(Options{})
			var stdout bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			cmd.SetArgs([]string{"setup", entry.ID})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("providers setup %s: %v\nstdout=%s", entry.ID, err, stdout.String())
			}
			for _, want := range []string{"Provider setup commands for", "gormes setup provider", "gormes model", "gormes doctor --offline"} {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
				}
			}
			if entry.AuthType == "api_key" && !strings.Contains(stdout.String(), "--type api-key") {
				t.Fatalf("api-key provider %s should advertise api-key credential command:\n%s", entry.ID, stdout.String())
			}
			if entry.AuthType != "api_key" && strings.Contains(stdout.String(), "--type api-key") {
				t.Fatalf("non-api-key provider %s should not advertise api-key credential command:\n%s", entry.ID, stdout.String())
			}
		})
	}
}

func TestProvidersSetupSubcommandListsAllSetupEntrypoints(t *testing.T) {
	cmd := NewProvidersCommand(Options{})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"setup"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("providers setup: %v\nstdout=%s", err, stdout.String())
	}
	for _, want := range []string{
		"Provider setup commands:",
		"Interactive: gormes setup provider",
		"Model picker: gormes model",
		"Credential pool: gormes auth add <provider>",
		"Run `gormes providers setup <provider>` for provider-specific commands.",
		"Known manifest providers:",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	for _, entry := range hermes.HermesProviderRegistryManifest() {
		if !strings.Contains(stdout.String(), entry.ID) {
			t.Fatalf("known manifest providers missing %s:\n%s", entry.ID, stdout.String())
		}
	}
}

func TestProvidersSetupSubcommandDoesNotAdvertiseUnsupportedOAuthAdapters(t *testing.T) {
	cmd := NewProvidersCommand(Options{})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"setup", "minimax-oauth"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("providers setup minimax-oauth: %v\nstdout=%s", err, stdout.String())
	}
	if strings.Contains(stdout.String(), "gormes auth add minimax-oauth --type oauth") {
		t.Fatalf("minimax-oauth should not advertise unimplemented OAuth auth add:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "MiniMax OAuth is row-backed") {
		t.Fatalf("minimax-oauth should explain row-backed auth guidance:\n%s", stdout.String())
	}
}
