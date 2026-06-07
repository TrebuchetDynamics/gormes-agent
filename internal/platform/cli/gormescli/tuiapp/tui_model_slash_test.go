package tuiapp

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUIModelSlashBindingLocalModelReceivesSessionModelSeamAndCatalog(t *testing.T) {
	cfg := loadNativeTUITestConfig(t)
	cfg.Hermes.Provider = "anthropic"
	cfg.Hermes.Model = "claude-sonnet-test"

	var captured tea.Model
	runOfflineTUIForTest(t, cfg, func(model tea.Model) {
		captured = model
		setModel := capturedTUISetSessionModel(t, model)
		if setModel == nil {
			t.Fatal("local TUI SetSessionModelFunc = nil, want kernel-backed model switch seam")
		}
		if err := setModel("anthropic", "claude-opus-test"); err != nil {
			t.Fatalf("SetSessionModelFunc: %v", err)
		}
	})

	if got := capturedTUIModelProvider(t, captured); got != "anthropic" {
		t.Fatalf("captured model provider = %q, want anthropic", got)
	}
	catalog := capturedTUIModelPickerCatalog(t, captured)
	if catalog == nil {
		t.Fatal("local TUI ModelPickerCatalog = nil, want provider/model catalog seam")
	}
	providers, err := catalog()
	if err != nil {
		t.Fatalf("ModelPickerCatalog(): %v", err)
	}
	if len(providers) == 0 {
		t.Fatal("ModelPickerCatalog returned no providers")
	}
}

func TestTUIModelSlashBindingRemoteTUIUnchanged(t *testing.T) {
	captured := runRemoteTUIForTest(t, "remote-model")

	if setModel := capturedTUISetSessionModel(t, captured); setModel != nil {
		t.Fatal("remote TUI SetSessionModelFunc is non-nil; remote startup must not receive local kernel seam")
	}
	if catalog := capturedTUIModelPickerCatalog(t, captured); catalog != nil {
		t.Fatal("remote TUI ModelPickerCatalog is non-nil; remote startup must not receive local model catalog")
	}
}

func capturedTUISetSessionModel(t *testing.T, model tea.Model) tui.SetSessionModelFunc {
	t.Helper()
	return capturedOptionalTUIModelField[tui.SetSessionModelFunc](t, model, "setSessionModel")
}

func capturedTUIModelPickerCatalog(t *testing.T, model tea.Model) tui.ModelPickerCatalogFunc {
	t.Helper()
	return capturedOptionalTUIModelField[tui.ModelPickerCatalogFunc](t, model, "modelPickerCatalog")
}

func capturedTUIModelProvider(t *testing.T, model tea.Model) string {
	t.Helper()
	return capturedRequiredTUIModelField[string](t, model, "modelProvider")
}
