package tuiapp

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUIUsageSlashBindingLocalModelReceivesAccountUsageAdapter(t *testing.T) {
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	cfg.Hermes.Provider = "custom-provider"
	var sawAccountUsage bool
	var snapshot llm.AccountUsageSnapshot
	var accountErr error
	runOfflineTUIForTest(t, cfg, func(model tea.Model) {
		accountUsage := capturedTUIAccountUsage(t, model)
		if accountUsage == nil {
			return
		}
		sawAccountUsage = true
		snapshot, accountErr = accountUsage(context.Background())
	})
	if !sawAccountUsage {
		t.Fatal("local TUI AccountUsage = nil, want provider-backed usage adapter")
	}
	if accountErr != nil {
		t.Fatalf("AccountUsage: %v", accountErr)
	}
	if snapshot.Provider != "custom-provider" {
		t.Fatalf("AccountUsage provider = %q, want custom-provider", snapshot.Provider)
	}
	if snapshot.Unavailable == nil || snapshot.Unavailable.Reason != llm.AccountUsageReasonUnsupportedProvider {
		t.Fatalf("AccountUsage unavailable = %+v, want unsupported provider evidence", snapshot.Unavailable)
	}
}

func TestTUIUsageSlashBindingRemoteTUIUnchanged(t *testing.T) {
	model := newPlainRemoteTUIModel()
	if accountUsage := capturedTUIAccountUsage(t, model); accountUsage != nil {
		t.Fatal("plain/remote TUI AccountUsage is non-nil; only local startup should inject provider adapter")
	}
}

func capturedTUIAccountUsage(t *testing.T, model tea.Model) tui.AccountUsageFunc {
	t.Helper()
	return capturedOptionalTUIModelField[tui.AccountUsageFunc](t, model, "accountUsage")
}
