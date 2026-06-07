package tuiapp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUISaveBinding_LocalModelReceivesSessionExport(t *testing.T) {
	cfg := loadNativeTUITestConfig(t)
	seedTUISaveTranscriptDB(t, "sess-binding", "discord:binding")

	var captured tea.Model
	runOfflineTUIForTest(t, cfg, func(model tea.Model) {
		captured = model
	})

	exportFn := capturedTUISessionExport(t, captured)
	if exportFn == nil {
		t.Fatal("local TUI SessionExport = nil, want XDG-backed export helper")
	}

	path, err := exportFn(context.Background(), "sess-binding")
	if err != nil {
		t.Fatalf("SessionExportFunc: %v", err)
	}
	wantDir := filepath.Join(config.GormesHome(), "sessions", "exports")
	if filepath.Dir(path) != wantDir {
		t.Fatalf("export dir = %q, want %q", filepath.Dir(path), wantDir)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Stat(export path): %v", err)
	}
}

func TestTUISaveBinding_RemoteTUIUnchanged(t *testing.T) {
	captured := runRemoteTUIForTest(t, "remote-binding")

	if exportFn := capturedTUISessionExport(t, captured); exportFn != nil {
		t.Fatal("remote TUI SessionExport is non-nil; remote startup must not receive local /save binding")
	}
}

func capturedTUISessionExport(t *testing.T, model tea.Model) tui.SessionExportFunc {
	t.Helper()
	return capturedOptionalTUIModelField[tui.SessionExportFunc](t, model, "sessionExport")
}
