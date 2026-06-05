package gormescli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

type modelCommandFakeSeams struct {
	isTTY bool

	current   cli.ProviderModel
	providers []cli.ProviderMenuEntry
	chooseIdx int
	model     string

	persisted []cli.Selection
}

func (f *modelCommandFakeSeams) seams() ModelCommandSeams {
	if f.providers == nil {
		f.providers = []cli.ProviderMenuEntry{{ID: "openai-codex", Label: "OpenAI Codex"}}
	}
	if f.model == "" {
		f.model = "gpt-5.5"
	}
	return ModelCommandSeams{
		IsTTY:       func() bool { return f.isTTY },
		LoadCurrent: func() (cli.ProviderModel, error) { return f.current, nil },
		ListProviders: func() ([]cli.ProviderMenuEntry, error) {
			return append([]cli.ProviderMenuEntry(nil), f.providers...), nil
		},
		ChooseProvider: func(entries []cli.ProviderMenuEntry, defaultIndex int) (int, error) {
			return f.chooseIdx, nil
		},
		ChooseModel: func(provider string, current string) (string, error) { return f.model, nil },
		PersistSelection: func(selection cli.Selection) error {
			f.persisted = append(f.persisted, selection)
			return nil
		},
	}
}

func runModelTestCommand(t *testing.T, seams ModelCommandSeams, args ...string) (string, string, error) {
	t.Helper()
	cmd := NewModelCommandWithSeams(seams)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestModelPickerRequiresTTY(t *testing.T) {
	fake := &modelCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runModelTestCommand(t, fake.seams())
	if !errors.Is(err, cli.ErrModelPickerRequiresTTY) {
		t.Fatalf("Execute error = %v, want ErrModelPickerRequiresTTY stdout=%s stderr=%s", err, stdout, stderr)
	}
	if len(fake.persisted) != 0 {
		t.Fatalf("persisted on non-TTY: %#v", fake.persisted)
	}
	if !strings.Contains(stdout+stderr+err.Error(), "model_picker_requires_tty") {
		t.Fatalf("missing typed evidence: stdout=%s stderr=%s err=%v", stdout, stderr, err)
	}
}

func TestModelPickerPersistsModelAndProvider(t *testing.T) {
	fake := &modelCommandFakeSeams{isTTY: true, current: cli.ProviderModel{Provider: "anthropic", Model: "claude-sonnet-4"}, providers: []cli.ProviderMenuEntry{{ID: "openai-codex", Label: "OpenAI Codex"}}, model: "gpt-5.5"}
	stdout, stderr, err := runModelTestCommand(t, fake.seams())
	if err != nil {
		t.Fatalf("Execute: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if len(fake.persisted) != 1 || fake.persisted[0].Provider != "openai-codex" || fake.persisted[0].Model != "gpt-5.5" {
		t.Fatalf("persisted = %#v", fake.persisted)
	}
	if !strings.Contains(stdout, "provider=openai-codex") || !strings.Contains(stdout, "model=gpt-5.5") {
		t.Fatalf("stdout missing redacted selection evidence:\n%s", stdout)
	}
}

func TestModelPickerOllamaCloudNormalizesSuffixedSelection(t *testing.T) {
	fake := &modelCommandFakeSeams{
		isTTY:     true,
		current:   cli.ProviderModel{Provider: "ollama-cloud", Model: "kimi-k2.6"},
		providers: []cli.ProviderMenuEntry{{ID: "ollama-cloud", Label: "Ollama Cloud"}},
		model:     "qwen3-coder:480b-cloud",
	}
	stdout, stderr, err := runModelTestCommand(t, fake.seams())
	if err != nil {
		t.Fatalf("Execute: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if len(fake.persisted) != 1 {
		t.Fatalf("persisted = %#v, want one selection", fake.persisted)
	}
	if got := fake.persisted[0].Model; got != "qwen3-coder:480b" {
		t.Fatalf("persisted model = %q, want qwen3-coder:480b", got)
	}
	if strings.Contains(stdout, "qwen3-coder:480b-cloud") {
		t.Fatalf("stdout shows suffixed Ollama Cloud model:\n%s", stdout)
	}
	if !strings.Contains(stdout, "model=qwen3-coder:480b") {
		t.Fatalf("stdout missing normalized model evidence:\n%s", stdout)
	}
}

func TestDefaultModelProviderEntriesUseHermesProviderCatalog(t *testing.T) {
	entries, err := DefaultModelProviderEntries()
	if err != nil {
		t.Fatalf("DefaultModelProviderEntries() error = %v", err)
	}
	if len(entries) != 37 {
		t.Fatalf("provider entries = %d, want 37", len(entries))
	}
	for _, want := range []struct {
		index int
		id    string
		label string
	}{
		{0, "nous", "Nous Portal (Nous Research subscription)"},
		{5, "openai-codex", "OpenAI Codex"},
		{36, "custom", "custom (direct API)"},
	} {
		got := entries[want.index]
		if got.ID != want.id || got.Label != want.label {
			t.Fatalf("entry[%d] = %#v, want id=%q label=%q", want.index, got, want.id, want.label)
		}
	}
	for _, entry := range entries {
		if strings.Contains(entry.Label, "(oauth_external)") || strings.Contains(entry.Label, "(api_key)") {
			t.Fatalf("provider entry leaked raw auth taxonomy: %#v", entry)
		}
		if entry.ID == cli.ProviderCatalogAuxConfig || entry.ID == cli.ProviderCatalogLeaveUnchanged || entry.ID == "custom-endpoint" {
			t.Fatalf("provider entry contains setup-only action: %#v", entry)
		}
	}
}

func setupModelCommandConfigTestEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("HOME", filepath.Join(root, "home"))
}

func TestModelCommandDefaultPersistWritesConfig(t *testing.T) {
	setupModelCommandConfigTestEnv(t)
	selection := cli.Selection{Provider: "openai-codex", Model: "gpt-5.5"}
	if err := PersistModelSelectionToConfig(selection); err != nil {
		t.Fatalf("PersistModelSelectionToConfig: %v", err)
	}
	body, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	text := string(body)
	for _, want := range []string{`model = 'gpt-5.5'`, `provider = 'openai-codex'`} {
		if !strings.Contains(text, want) {
			t.Fatalf("config missing %s:\n%s", want, text)
		}
	}
	if strings.Contains(text, "access_token") || strings.Contains(text, "refresh_token") {
		t.Fatalf("config leaked credential-looking field:\n%s", text)
	}
	if filepath.Dir(config.ConfigPath()) == t.TempDir() {
		t.Fatal("unreachable path guard")
	}
}
