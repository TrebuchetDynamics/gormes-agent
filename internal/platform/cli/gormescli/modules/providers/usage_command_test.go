package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func TestUsageCommandUsesInjectedSeamsAndBuildProvenance(t *testing.T) {
	var requests []hermes.AccountUsageFetchRequest
	cmd := NewUsageCommandWithSeams(UsageSeams{
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				Hermes: config.HermesCfg{
					APIKey: "configured-secret",
				},
			}, nil
		},
		FetchAccountUsage: func(_ context.Context, req hermes.AccountUsageFetchRequest) (hermes.AccountUsageSnapshot, error) {
			requests = append(requests, req)
			return hermes.AccountUsageSnapshot{
				Provider:  req.Provider,
				AccountID: "acct-fixture",
				Plan:      "Pro",
				Source:    "fixture",
				FetchedAt: time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC),
				Windows: []hermes.AccountUsageWindow{{
					Label:       "Session",
					UsedPercent: floatPtr(25),
				}},
			}, nil
		},
	}, Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			return gormescli.BuildProvenance{Version: "test-version", GitCommit: "test-sha"}
		},
	})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--provider", "openrouter", "--api-key", "flag-secret", "--account-id", "acct-123", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("usage --json: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}
	if requests[0].Provider != "openrouter" || requests[0].APIKey != "flag-secret" || requests[0].AccountID != "acct-123" {
		t.Fatalf("request = %+v, want injected provider/key/account", requests[0])
	}
	if bytes.Contains(stdout.Bytes(), []byte("flag-secret")) || bytes.Contains(stdout.Bytes(), []byte("configured-secret")) {
		t.Fatalf("usage JSON leaked a credential:\n%s", stdout.String())
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Provider string `json:"provider"`
		Plan     string `json:"plan"`
		Windows  []struct {
			Label       string   `json:"label"`
			UsedPercent *float64 `json:"used_percent"`
		} `json:"windows"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("usage stdout must be JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Build.Version != "test-version" || got.Build.GitCommit != "test-sha" {
		t.Fatalf("build provenance = %+v, want injected test values", got.Build)
	}
	if got.Provider != "openrouter" || got.Plan != "Pro" {
		t.Fatalf("usage JSON = %+v, want openrouter Pro", got)
	}
	if len(got.Windows) != 1 || got.Windows[0].Label != "Session" || got.Windows[0].UsedPercent == nil || *got.Windows[0].UsedPercent != 25 {
		t.Fatalf("windows = %+v, want Session 25%% used", got.Windows)
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
