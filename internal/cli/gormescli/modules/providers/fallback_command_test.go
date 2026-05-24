package providers

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

func TestFallbackCommandUsesInjectedModelSeams(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]\nprovider = 'openai-codex'\nmodel = 'gpt-5.5'\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var persisted []cli.Selection
	cmd := NewFallbackCommandWithSeams(ModelCommandSeams{
		IsTTY:       func() bool { return true },
		LoadCurrent: func() (cli.ProviderModel, error) { return cli.ProviderModel{}, nil },
		ListProviders: func() ([]cli.ProviderMenuEntry, error) {
			return []cli.ProviderMenuEntry{{ID: "ollama-cloud", Label: "Ollama Cloud"}}, nil
		},
		ChooseProvider: func([]cli.ProviderMenuEntry, int) (int, error) { return 0, nil },
		ChooseModel:    func(provider string, current string) (string, error) { return "qwen3-coder:480b-cloud", nil },
		PersistSelection: func(selection cli.Selection) error {
			persisted = append(persisted, selection)
			return nil
		},
	})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"add"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("fallback add: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if len(persisted) != 0 {
		t.Fatalf("fallback command must override picker persistence instead of mutating primary model: %#v", persisted)
	}
	cfg, err := LoadFallbackConfig(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatalf("LoadFallbackConfig: %v", err)
	}
	if len(cfg.Chain) != 1 || cfg.Chain[0].Provider != "ollama-cloud" || cfg.Chain[0].Model != "qwen3-coder:480b" {
		t.Fatalf("fallback chain = %#v, want normalized Ollama Cloud fallback", cfg.Chain)
	}
	if !strings.Contains(stdout.String(), "Added fallback: qwen3-coder:480b  (via ollama-cloud)") {
		t.Fatalf("stdout missing added fallback evidence:\n%s", stdout.String())
	}
}
