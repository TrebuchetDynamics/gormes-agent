package main

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUIKanbanSlashBindingLocalModelReceivesRunner(t *testing.T) {
	setupNativeTUITestEnv(t)

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}

	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}

	var sawRunner bool
	var initOut, listOut string
	var initErr, listErr error
	err = runResolvedTUIWithRuntime(cmd, tuiInvocation{Config: cfg}, rootRuntime{
		tuiProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) tuiProgram {
			return fakeTUIProgram{run: func() {
				runKanban := capturedTUIKanbanSlash(t, model)
				if runKanban == nil {
					return
				}
				sawRunner = true
				initOut, initErr = runKanban("/kanban init")
				listOut, listErr = runKanban("/kanban list")
			}}
		},
	})
	if err != nil {
		t.Fatalf("runResolvedTUIWithRuntime: %v", err)
	}

	if !sawRunner {
		t.Fatal("local TUI KanbanSlash = nil, want CLI-backed runner")
	}
	if initErr != nil {
		t.Fatalf("KanbanSlash(/kanban init): %v\nout=%s", initErr, initOut)
	}
	if !strings.Contains(initOut, "kanban initialized at") {
		t.Fatalf("KanbanSlash output = %q, want init output", initOut)
	}
	if listErr != nil {
		t.Fatalf("KanbanSlash(/kanban list): %v\nout=%s", listErr, listOut)
	}
	if !strings.Contains(listOut, "No Kanban tasks.") {
		t.Fatalf("KanbanSlash list output = %q, want empty-board output", listOut)
	}
}

func TestTUIKanbanSlashBindingRemoteTUIUnchanged(t *testing.T) {
	model := tui.NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, tui.Options{})
	if runKanban := capturedTUIKanbanSlash(t, model); runKanban != nil {
		t.Fatal("plain/remote TUI KanbanSlash is non-nil; only local startup should inject command runner")
	}
}

func TestRunTUIKanbanSlashCommandSurfacesErrors(t *testing.T) {
	setupNativeTUITestEnv(t)

	out, err := runTUIKanbanSlashCommand(context.Background(), "/kanban show missing-task")
	if err == nil {
		t.Fatalf("runTUIKanbanSlashCommand missing-task error = nil\nout=%s", out)
	}
	if !strings.Contains(out, "not found") && !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing-task output/error missing evidence:\nout=%s\nerr=%v", out, err)
	}
}

func capturedTUIKanbanSlash(t *testing.T, model tea.Model) tui.KanbanSlashFunc {
	t.Helper()

	m, ok := model.(tui.Model)
	if !ok {
		t.Fatalf("captured model type = %T, want tui.Model", model)
	}

	field := reflect.ValueOf(&m).Elem().FieldByName("kanbanSlash")
	if !field.IsValid() {
		t.Fatal("tui.Model missing kanbanSlash field")
	}
	if field.IsNil() {
		return nil
	}

	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.KanbanSlashFunc)
}
