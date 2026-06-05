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
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
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

	out, err := gormescli.RunTUIKanbanSlashCommand(context.Background(), "/kanban show missing-task", kanbanCommandOptions())
	if err == nil {
		t.Fatalf("runTUIKanbanSlashCommand missing-task error = nil\nout=%s", out)
	}
	if !strings.Contains(out, "not found") && !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing-task output/error missing evidence:\nout=%s\nerr=%v", out, err)
	}
}

func TestRunTUIKanbanSlashCommandCuratedHelpAliases(t *testing.T) {
	setupNativeTUITestEnv(t)

	bare, err := gormescli.RunTUIKanbanSlashCommand(context.Background(), "/kanban", kanbanCommandOptions())
	if err != nil {
		t.Fatalf("runTUIKanbanSlashCommand(/kanban): %v\nout=%s", err, bare)
	}
	for _, want := range []string{"/kanban", "Common subcommands", "`list`", "`show <id>`"} {
		if !strings.Contains(bare, want) {
			t.Fatalf("curated help missing %q:\n%s", want, bare)
		}
	}
	if len(bare) >= 2000 {
		t.Fatalf("curated help length = %d, want chat-sized output", len(bare))
	}
	for _, banned := range []string{"Usage:", "gormes kanban", "(usage error:"} {
		if strings.Contains(bare, banned) {
			t.Fatalf("curated help leaked %q:\n%s", banned, bare)
		}
	}

	for _, alias := range []string{"help", "--help", "-h", "?"} {
		out, err := gormescli.RunTUIKanbanSlashCommand(context.Background(), "/kanban "+alias, kanbanCommandOptions())
		if err != nil {
			t.Fatalf("runTUIKanbanSlashCommand(/kanban %s): %v\nout=%s", alias, err, out)
		}
		if out != bare {
			t.Fatalf("/kanban %s help differs from bare help:\n--- bare ---\n%s\n--- alias ---\n%s", alias, bare, out)
		}
	}
}

func TestRunTUIKanbanSlashCommandSubcommandHelpUsesSlashProg(t *testing.T) {
	setupNativeTUITestEnv(t)

	out, err := gormescli.RunTUIKanbanSlashCommand(context.Background(), "/kanban show -h", kanbanCommandOptions())
	if err != nil {
		t.Fatalf("runTUIKanbanSlashCommand(/kanban show -h): %v\nout=%s", err, out)
	}
	for _, want := range []string{"/kanban show", "<task-id>"} {
		if !strings.Contains(out, want) {
			t.Fatalf("subcommand help missing %q:\n%s", want, out)
		}
	}
	for _, banned := range []string{"⚠", "gormes kanban", "kanban kanban", "(usage error:"} {
		if strings.Contains(out, banned) {
			t.Fatalf("subcommand help leaked %q:\n%s", banned, out)
		}
	}
}

func TestRunTUIKanbanSlashCommandUsageErrorsAreFriendly(t *testing.T) {
	setupNativeTUITestEnv(t)

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "unknown action",
			input: "/kanban frobnicate",
			want:  []string{"⚠ /kanban usage error", "frobnicate", "/kanban"},
		},
		{
			name:  "missing required arg",
			input: "/kanban show",
			want:  []string{"⚠ /kanban usage error", "/kanban show", "<task-id>"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := gormescli.RunTUIKanbanSlashCommand(context.Background(), tt.input, kanbanCommandOptions())
			if err != nil {
				t.Fatalf("runTUIKanbanSlashCommand(%q) err = %v\nout=%s", tt.input, err, out)
			}
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Fatalf("usage output missing %q:\n%s", want, out)
				}
			}
			for _, banned := range []string{"gormes kanban", "kanban kanban", "(usage error:"} {
				if strings.Contains(out, banned) {
					t.Fatalf("usage output leaked %q:\n%s", banned, out)
				}
			}
		})
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
