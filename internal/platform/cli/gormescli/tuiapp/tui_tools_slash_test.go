package tuiapp

import (
	"reflect"
	"testing"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
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
	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}

	var sawConfigure bool
	var result tui.ToolsConfigureResult
	err = RunResolved(cmd, Invocation{Config: cfg}, Runtime{
		ProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) Program {
			return fakeTUIProgram{run: func() {
				configure := capturedTUIToolsConfigure(t, model)
				if configure == nil {
					return
				}
				sawConfigure = true
				result, err = configure(tui.ToolsConfigureRequest{Action: "enable", Names: []string{"web"}, SessionID: "sess-tools"})
			}}
		},
	})
	if err != nil {
		t.Fatalf("RunResolved: %v", err)
	}
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
	model := tui.NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, tui.Options{})
	if configure := capturedTUIToolsConfigure(t, model); configure != nil {
		t.Fatal("plain/remote TUI ToolsConfigure is non-nil; only local startup should inject /tools adapter")
	}
}

func capturedTUIToolsConfigure(t *testing.T, model tea.Model) tui.ToolsConfigureFunc {
	t.Helper()
	m, ok := model.(tui.Model)
	if !ok {
		t.Fatalf("captured model type = %T, want tui.Model", model)
	}
	field := reflect.ValueOf(&m).Elem().FieldByName("toolsConfigure")
	if !field.IsValid() {
		t.Fatal("tui.Model missing toolsConfigure field")
	}
	if field.IsNil() {
		return nil
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.ToolsConfigureFunc)
}
