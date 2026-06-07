package tuiapp

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUISkinSlashBindingLocalModelReceivesSkinConfig(t *testing.T) {
	setupNativeTUITestEnv(t)
	if err := config.WriteTOMLValue(config.ConfigPath(), "tui.theme", "ares"); err != nil {
		t.Fatalf("write tui.theme: %v", err)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	var sawConfig bool
	var initialSkin string
	var status, changed tui.SkinConfigResult
	var invalidErr error
	runOfflineTUIForTest(t, cfg, func(model tea.Model) {
		initialSkin = capturedTUIActiveSkinName(t, model)
		configure := capturedTUISkinConfig(t, model)
		if configure == nil {
			return
		}
		sawConfig = true
		status, err = configure(tui.SkinConfigRequest{SessionID: "sess-skin"})
		if err != nil {
			return
		}
		changed, err = configure(tui.SkinConfigRequest{Name: "mono", SessionID: "sess-skin"})
		if err != nil {
			return
		}
		_, invalidErr = configure(tui.SkinConfigRequest{Name: "zeus", SessionID: "sess-skin"})
	})
	if initialSkin != "ares" {
		t.Fatalf("initial active skin = %q, want ares from tui.theme", initialSkin)
	}
	if !sawConfig {
		t.Fatal("local TUI SkinConfig = nil, want config-backed /skin adapter")
	}
	if err != nil {
		t.Fatalf("SkinConfig: %v", err)
	}
	if status.Name != "ares" {
		t.Fatalf("status skin = %+v, want ares", status)
	}
	if changed.Name != "mono" {
		t.Fatalf("changed skin = %+v, want mono", changed)
	}
	if invalidErr == nil || !strings.Contains(invalidErr.Error(), "unknown skin") {
		t.Fatalf("invalid skin error = %v, want unknown skin", invalidErr)
	}
	reloaded, err := config.Load(nil)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if reloaded.TUI.Theme != "mono" {
		t.Fatalf("persisted tui.theme = %q, want mono", reloaded.TUI.Theme)
	}
}

func TestTUISkinSlashBindingRemoteTUIUnchanged(t *testing.T) {
	model := newPlainRemoteTUIModel()
	if configure := capturedTUISkinConfig(t, model); configure != nil {
		t.Fatal("plain/remote TUI SkinConfig is non-nil; only local startup should inject /skin adapter")
	}
}

func capturedTUISkinConfig(t *testing.T, model tea.Model) tui.SkinConfigFunc {
	t.Helper()
	return capturedOptionalTUIModelField[tui.SkinConfigFunc](t, model, "skinConfig")
}

func capturedTUIActiveSkinName(t *testing.T, model tea.Model) string {
	t.Helper()
	return capturedRequiredTUIModelField[string](t, model, "activeSkinName")
}
