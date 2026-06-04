package main

import (
	"reflect"
	"testing"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUIBranchSlashBindingLocalModelReceivesSessionBranchAdapter(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "sess-parent", role: "user", content: "parent question", ts: 100},
		{id: "sess-parent", role: "assistant", content: "parent answer", ts: 101},
	})
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}

	var sawBranch bool
	err = runResolvedTUIWithRuntime(cmd, tuiInvocation{Config: cfg}, rootRuntime{
		tuiProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) tuiProgram {
			return fakeTUIProgram{run: func() {
				branch := capturedTUISessionBranch(t, model)
				if branch == nil {
					return
				}
				sawBranch = true
			}}
		},
	})
	if err != nil {
		t.Fatalf("runResolvedTUIWithRuntime: %v", err)
	}
	if !sawBranch {
		t.Fatal("local TUI SessionBranch = nil, want memory-backed branch adapter")
	}
}

func TestTUIBranchSlashBindingRemoteTUIUnchanged(t *testing.T) {
	model := tui.NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, tui.Options{})
	if branch := capturedTUISessionBranch(t, model); branch != nil {
		t.Fatal("plain/remote TUI SessionBranch is non-nil; only local startup should inject branch adapter")
	}
}

func capturedTUISessionBranch(t *testing.T, model tea.Model) tui.SessionBranchFunc {
	t.Helper()

	m, ok := model.(tui.Model)
	if !ok {
		t.Fatalf("captured model type = %T, want tui.Model", model)
	}

	field := reflect.ValueOf(&m).Elem().FieldByName("sessionBranch")
	if !field.IsValid() {
		t.Fatal("tui.Model missing sessionBranch field")
	}
	if field.IsNil() {
		return nil
	}

	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.SessionBranchFunc)
}
