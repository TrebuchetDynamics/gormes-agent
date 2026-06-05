package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/spf13/cobra"
)

func TestSetupModelPickerUsesActiveProviderPickerModelSet(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("6\n"))

	err := runSetupActiveProviderModelPicker(cmd, cli.ProviderModel{Provider: "openai-codex", Model: "gpt-5.5"})
	if err != nil {
		t.Fatalf("runSetupActiveProviderModelPicker() error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		"Select model for openai-codex:",
		"1. gpt-5.5",
		"6. gpt-5.2-codex",
		"Choice [1-8] (1), custom model, or q to cancel:",
		"model selection saved: provider=openai-codex model=gpt-5.2-codex",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "Suggested models for openai-codex:") {
		t.Fatalf("stdout still uses comma-separated suggestion line instead of selectable list:\n%s", stdout.String())
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

func TestSetupModelPickerUsesOpenRouterFullModelSet(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	withOpenRouterModelCatalogFetcherForTest(t, openRouterModelCatalogOfflineForTest)

	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("\n"))

	err := runSetupActiveProviderModelPicker(cmd, cli.ProviderModel{Provider: "openrouter", Model: "moonshotai/kimi-k2.6"})
	if err != nil {
		t.Fatalf("runSetupActiveProviderModelPicker() error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		"Select model for openrouter:",
		"1. deepseek/deepseek-chat-v3-0324:free",
		"4. qwen/qwen3-235b-a22b:free",
		"8. moonshotai/kimi-k2.6",
		"12. openai/gpt-5.5",
		"35. inclusionai/ring-2.6-1t:free",
		"Choice [1-35] (8), custom model, or q to cancel:",
		"model selection saved: provider=openrouter model=moonshotai/kimi-k2.6",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "Model for openrouter [gpt-5.5]") {
		t.Fatalf("stdout used stale free-text OpenRouter prompt:\n%s", stdout.String())
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
