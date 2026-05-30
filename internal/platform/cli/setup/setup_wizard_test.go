package setup

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestSetupWizard_PromptDefault(t *testing.T) {
	input := "test-value\n"
	w := &SetupWizard{in: bufio.NewReader(strings.NewReader(input)), out: os.Stdout}
	result := w.prompt("Label", "default")
	if result != "test-value" {
		t.Fatalf("prompt = %q, want test-value", result)
	}
}

func TestSetupWizard_PromptEmptyReturnsDefault(t *testing.T) {
	w := &SetupWizard{in: bufio.NewReader(strings.NewReader("\n")), out: os.Stdout}
	result := w.prompt("Label", "default-value")
	if result != "default-value" {
		t.Fatalf("prompt = %q, want default-value", result)
	}
}

func TestSetupWizard_PromptEmptyNoDefault(t *testing.T) {
	w := &SetupWizard{in: bufio.NewReader(strings.NewReader("\n")), out: os.Stdout}
	result := w.prompt("Label", "")
	if result != "" {
		t.Fatalf("prompt = %q, want empty", result)
	}
}

func TestSetupWizard_PromptTrimsWhitespace(t *testing.T) {
	w := &SetupWizard{in: bufio.NewReader(strings.NewReader("  spaced  \n")), out: os.Stdout}
	result := w.prompt("Label", "")
	if result != "spaced" {
		t.Fatalf("prompt = %q, want 'spaced'", result)
	}
}

func TestSetupWizard_NonInteractiveRun(t *testing.T) {
	input := "\n\n\n"
	w := &SetupWizard{in: bufio.NewReader(strings.NewReader(input)), out: os.Stderr}
	if err := w.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestProviderAPIEnvKey(t *testing.T) {
	tests := map[string]string{
		"openai":     "OPENAI_API_KEY",
		"anthropic":  "ANTHROPIC_API_KEY",
		"openrouter": "OPENROUTER_API_KEY",
		"deepseek":   "DEEPSEEK_API_KEY",
		"unknown":    "OPENAI_API_KEY",
	}
	for provider, want := range tests {
		if got := providerAPIEnvKey(provider); got != want {
			t.Fatalf("providerAPIEnvKey(%q) = %q, want %q", provider, got, want)
		}
	}
}
