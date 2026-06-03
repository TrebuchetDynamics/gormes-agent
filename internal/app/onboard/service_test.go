package onboard

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

func TestBuildStatusReportDoesNotExposeSecrets(t *testing.T) {
	cfg := config.Config{}
	cfg.Hermes.Provider = "anthropic"
	cfg.Hermes.Endpoint = "https://api.anthropic.com"
	cfg.Hermes.Model = "claude-sonnet-4-5"
	cfg.Hermes.APIKey = "sk-ant-fixture-token"

	report := BuildStatusReport(cfg, Runtime{
		Build:      BuildProvenance{Version: "test", GitCommit: "abc"},
		Home:       "/tmp/gormes-home",
		ConfigPath: "/tmp/gormes-home/config.toml",
	})

	if !report.ProviderConfigured {
		t.Fatalf("ProviderConfigured = false, want true")
	}
	if !report.AuthConfigured {
		t.Fatalf("AuthConfigured = false, want true")
	}
	if report.Provider != "anthropic" || report.Model != "claude-sonnet-4-5" {
		t.Fatalf("provider/model = %q/%q", report.Provider, report.Model)
	}
	if strings.Contains(report.Endpoint+report.Model+report.Provider, "sk-ant-fixture-token") {
		t.Fatalf("status report leaked API key")
	}
}

func TestPrintFirstRunReadinessPreservesGuidanceText(t *testing.T) {
	plan := cli.FirstRunPlan{
		Ready:       false,
		Summary:     "not ready: provider endpoint is not configured",
		NextCommand: "gormes setup --quick --target terminal",
		MissingSteps: []cli.FirstRunStep{{
			ID:     cli.FirstRunStepProvider,
			Label:  "Provider",
			Detail: "provider endpoint is not configured",
		}},
	}
	var out strings.Builder
	PrintFirstRunReadiness(&out, plan, Runtime{})
	for _, want := range []string{"First-run readiness: setup needed", "Provider: provider endpoint is not configured", "Next: gormes setup --quick --target terminal"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestNormalizeActionAliases(t *testing.T) {
	for input, want := range map[string]string{"r": "run", "s": "skip", "v": "review", "": "run"} {
		if got := NormalizeAction(Runtime{}, input, "run"); got != want {
			t.Fatalf("NormalizeAction(%q) = %q, want %q", input, got, want)
		}
	}
}
