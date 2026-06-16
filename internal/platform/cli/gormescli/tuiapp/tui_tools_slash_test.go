package tuiapp

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUIToolsSlashBindingLocalModelReceivesToolsConfigure(t *testing.T) {
	setupNativeTUITestEnv(t)
	writeSetupToolsFixtureConfig(t, `
platform_toolsets = { cli = ["terminal"] }
`)
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	var sawConfigure bool
	var result tui.ToolsConfigureResult
	runOfflineTUIForTest(t, cfg, func(model tea.Model) {
		configure := capturedTUIToolsConfigure(t, model)
		if configure == nil {
			return
		}
		sawConfigure = true
		result, err = configure(tui.ToolsConfigureRequest{Action: "enable", Names: []string{"web"}, SessionID: "sess-tools"})
	})
	if !sawConfigure {
		t.Fatal("local TUI ToolsConfigure = nil, want config-backed /tools adapter")
	}
	if err != nil {
		t.Fatalf("ToolsConfigure: %v", err)
	}
	if !containsString(result.Changed, "web") || !result.Reset {
		t.Fatalf("ToolsConfigure result = %+v, want changed web with reset evidence", result)
	}
	got := readCLIPlatformToolsets(t)
	if !containsString(got, "web") || !containsString(got, "terminal") {
		t.Fatalf("persisted CLI toolsets = %v, want existing terminal plus enabled web", got)
	}
}

func TestTUIToolsSlashBindingRemoteTUIUnchanged(t *testing.T) {
	model := newPlainRemoteTUIModel()
	if configure := capturedTUIToolsConfigure(t, model); configure != nil {
		t.Fatal("plain/remote TUI ToolsConfigure is non-nil; only local startup should inject /tools adapter")
	}
}

func capturedTUIToolsConfigure(t *testing.T, model tea.Model) tui.ToolsConfigureFunc {
	t.Helper()
	return capturedOptionalTUIModelField[tui.ToolsConfigureFunc](t, model, "toolsConfigure")
}
