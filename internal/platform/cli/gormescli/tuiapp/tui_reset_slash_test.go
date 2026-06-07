package tuiapp

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUIResetSlashBindingLocalModelReceivesSessionResetSeam(t *testing.T) {
	cfg := loadNativeTUITestConfig(t)

	var captured tea.Model
	runOfflineTUIForTest(t, cfg, func(model tea.Model) {
		captured = model
		reset := capturedTUISessionReset(t, model)
		if reset == nil {
			t.Fatal("local TUI SessionReset = nil, want kernel-backed reset seam")
		}
		if err := reset(); err != nil {
			t.Fatalf("SessionResetFunc: %v", err)
		}
	})
	if captured == nil {
		t.Fatal("did not capture local TUI model")
	}
}

func TestTUIResetSlashBindingRemoteTUIUnchanged(t *testing.T) {
	captured := runRemoteTUIForTest(t, "remote-model")

	if reset := capturedTUISessionReset(t, captured); reset != nil {
		t.Fatal("remote TUI SessionReset is non-nil; remote startup must not receive local kernel seam")
	}
}

func capturedTUISessionReset(t *testing.T, model tea.Model) tui.SessionResetFunc {
	t.Helper()
	return capturedOptionalTUIModelField[tui.SessionResetFunc](t, model, "sessionReset")
}
