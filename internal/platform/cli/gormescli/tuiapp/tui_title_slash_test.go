package tuiapp

import (
	"context"
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUITitleSlashBindingLocalModelPersistsManualTitle(t *testing.T) {
	cfg := loadNativeTUITestConfig(t)
	if err := os.MkdirAll(config.GormesHome(), 0o755); err != nil {
		t.Fatalf("mkdir GORMES_HOME: %v", err)
	}
	var sawTitle bool
	var setRes tui.SessionTitleResult
	var setErr error
	var getRes tui.SessionTitleResult
	var getErr error
	runOfflineTUIForTest(t, cfg, func(model tea.Model) {
		titleFn := capturedTUITitle(t, model)
		if titleFn == nil {
			return
		}
		sawTitle = true
		setRes, setErr = titleFn("sess-tui-title", "Operator Title")
		getRes, getErr = titleFn("sess-tui-title", "")
	})
	if !sawTitle {
		t.Fatal("local TUI SessionTitle = nil, want metadata-backed title adapter")
	}
	if setErr != nil {
		t.Fatalf("SessionTitle set: %v", setErr)
	}
	if setRes.Title != "Operator Title" {
		t.Fatalf("SessionTitle set result = %+v, want Operator Title", setRes)
	}
	if getErr != nil {
		t.Fatalf("SessionTitle get: %v", getErr)
	}
	if getRes.Title != "Operator Title" {
		t.Fatalf("SessionTitle get result = %+v, want Operator Title", getRes)
	}

	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		t.Fatalf("OpenBolt(%s): %v", config.SessionDBPath(), err)
	}
	defer smap.Close()
	meta, ok, err := smap.GetMetadata(context.Background(), "sess-tui-title")
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if !ok || meta.Title != "Operator Title" || !meta.TitleManuallySet {
		t.Fatalf("metadata = %+v ok=%v, want manual Operator Title", meta, ok)
	}
}

func TestTUITitleSlashBindingRemoteTUIUnchanged(t *testing.T) {
	model := newPlainRemoteTUIModel()
	if titleFn := capturedTUITitle(t, model); titleFn != nil {
		t.Fatal("plain/remote TUI SessionTitle is non-nil; only local startup should inject title adapter")
	}
}

func capturedTUITitle(t *testing.T, model tea.Model) tui.SessionTitleFunc {
	t.Helper()
	return capturedOptionalTUIModelField[tui.SessionTitleFunc](t, model, "sessionTitle")
}
