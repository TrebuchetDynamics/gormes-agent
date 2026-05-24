package providers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

func TestModelCommandUsesInjectedSeamsAndNormalizesSelection(t *testing.T) {
	var persisted []cli.Selection
	cmd := NewModelCommandWithSeams(ModelCommandSeams{
		IsTTY: func() bool { return true },
		LoadCurrent: func() (cli.ProviderModel, error) {
			return cli.ProviderModel{Provider: "ollama-cloud", Model: "kimi-k2.6"}, nil
		},
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

	if err := cmd.Execute(); err != nil {
		t.Fatalf("model command: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if len(persisted) != 1 || persisted[0].Provider != "ollama-cloud" || persisted[0].Model != "qwen3-coder:480b" {
		t.Fatalf("persisted = %#v, want normalized Ollama Cloud model", persisted)
	}
	if strings.Contains(stdout.String(), "qwen3-coder:480b-cloud") {
		t.Fatalf("stdout leaked suffixed model:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "provider=ollama-cloud") || !strings.Contains(stdout.String(), "model=qwen3-coder:480b") {
		t.Fatalf("stdout missing saved model evidence:\n%s", stdout.String())
	}
}
