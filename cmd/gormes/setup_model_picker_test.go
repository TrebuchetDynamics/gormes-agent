package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/spf13/cobra"
)

func TestSetupModelPickerUsesActiveProviderPickerModelSet(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("gpt-5.2-codex\n"))

	err := runSetupActiveProviderModelPicker(cmd, cli.ProviderModel{Provider: "openai-codex", Model: "gpt-5.5"})
	if err != nil {
		t.Fatalf("runSetupActiveProviderModelPicker() error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	suggestionLine := firstSetupModelPickerLineWithPrefix(stdout.String(), "Suggested models for openai-codex:")
	if !strings.Contains(suggestionLine, "gpt-5.2-codex") {
		t.Fatalf("suggestion line missing picker model beyond static prompt ceiling:\n%s", stdout.String())
	}
	for _, want := range []string{
		"Suggested models for openai-codex:",
		"Model for openai-codex [gpt-5.5]",
		"model selection saved: provider=openai-codex model=gpt-5.2-codex",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "Model for openrouter") {
		t.Fatalf("stdout used wrong provider prompt:\n%s", stdout.String())
	}

	body, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	for _, want := range []string{`provider = 'openai-codex'`, `model = 'gpt-5.2-codex'`} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("config missing %q:\n%s", want, string(body))
		}
	}
}

func TestSetupModelPickerCancelDoesNotPersist(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("q\n"))

	err := runSetupActiveProviderModelPicker(cmd, cli.ProviderModel{Provider: "openai-codex", Model: "gpt-5.5"})
	if !errors.Is(err, cli.ErrModelPickerCancelled) {
		t.Fatalf("runSetupActiveProviderModelPicker() error = %v, want ErrModelPickerCancelled stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	if _, statErr := os.Stat(config.ConfigPath()); !os.IsNotExist(statErr) {
		t.Fatalf("cancel mutated config path %s: %v", config.ConfigPath(), statErr)
	}
}

func TestSetupModelPickerReportsDegradedFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("custom-model\n"))

	err := runSetupActiveProviderModelPicker(cmd, cli.ProviderModel{Provider: "fixture-provider", Model: "current-model"})
	if err != nil {
		t.Fatalf("runSetupActiveProviderModelPicker() error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		"Model catalog degraded for fixture-provider: provider not in picker catalog; accepting free-text model.",
		"Model for fixture-provider [current-model]",
		"model selection saved: provider=fixture-provider model=custom-model",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func firstSetupModelPickerLineWithPrefix(text, prefix string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}
