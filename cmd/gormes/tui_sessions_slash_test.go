package main

import (
	"context"
	"reflect"
	"testing"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUISessionsSlashBindingLocalModelReceivesSessionDirectory(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "sess-alpha", title: "Alpha Work", role: "user", content: "preview alpha", ts: 100},
		{id: "sess-beta", title: "Beta Work", role: "user", content: "preview beta", ts: 200},
	})
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}

	var sawDirectory bool
	var entries []tui.SessionDirectoryEntry
	var dirErr error
	err = runResolvedTUIWithRuntime(cmd, tuiInvocation{Config: cfg}, rootRuntime{
		tuiProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) tuiProgram {
			return fakeTUIProgram{run: func() {
				directory := capturedTUISessionDirectory(t, model)
				if directory == nil {
					return
				}
				sawDirectory = true
				entries, dirErr = directory(1)
			}}
		},
	})
	if err != nil {
		t.Fatalf("runResolvedTUIWithRuntime: %v", err)
	}
	if !sawDirectory {
		t.Fatal("local TUI SessionDirectory = nil, want memory-backed sessions adapter")
	}
	if dirErr != nil {
		t.Fatalf("SessionDirectory: %v", dirErr)
	}
	if len(entries) != 1 {
		t.Fatalf("SessionDirectory returned %d entries, want limit 1: %+v", len(entries), entries)
	}
	if entries[0].ID != "sess-beta" || entries[0].Title != "Beta Work" || entries[0].Preview != "preview beta" || entries[0].MessageCount != 1 {
		t.Fatalf("SessionDirectory entry = %+v, want newest Beta Work entry", entries[0])
	}
}

func TestTUIResumeSlashBindingLocalModelReceivesSessionResumeAdapter(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "sess-alpha", role: "user", content: "alpha question", ts: 100},
		{id: "sess-alpha", role: "assistant", content: "alpha answer", ts: 101},
	})
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}

	var sawResume bool
	var result tui.SessionResumeResult
	var resumeErr error
	err = runResolvedTUIWithRuntime(cmd, tuiInvocation{Config: cfg}, rootRuntime{
		tuiProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) tuiProgram {
			return fakeTUIProgram{run: func() {
				resume := capturedTUISessionResume(t, model)
				if resume == nil {
					return
				}
				sawResume = true
				result, resumeErr = resume(context.Background(), "sess-al")
			}}
		},
	})
	if err != nil {
		t.Fatalf("runResolvedTUIWithRuntime: %v", err)
	}
	if !sawResume {
		t.Fatal("local TUI SessionResume = nil, want memory-backed resume adapter")
	}
	if resumeErr != nil {
		t.Fatalf("SessionResume: %v", resumeErr)
	}
	if result.SessionID != "sess-alpha" {
		t.Fatalf("SessionResume SessionID = %q, want sess-alpha", result.SessionID)
	}
	if len(result.History) != 2 || result.History[0].Content != "alpha question" || result.History[1].Content != "alpha answer" {
		t.Fatalf("SessionResume History = %+v, want replayed alpha transcript", result.History)
	}
}

func TestTUISessionsSlashBindingRemoteTUIUnchanged(t *testing.T) {
	model := tui.NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, tui.Options{})
	if directory := capturedTUISessionDirectory(t, model); directory != nil {
		t.Fatal("plain/remote TUI SessionDirectory is non-nil; only local startup should inject sessions adapter")
	}
	if resume := capturedTUISessionResume(t, model); resume != nil {
		t.Fatal("plain/remote TUI SessionResume is non-nil; only local startup should inject resume adapter")
	}
}

func capturedTUISessionDirectory(t *testing.T, model tea.Model) tui.SessionDirectoryFunc {
	t.Helper()

	m, ok := model.(tui.Model)
	if !ok {
		t.Fatalf("captured model type = %T, want tui.Model", model)
	}

	field := reflect.ValueOf(&m).Elem().FieldByName("sessionDirectory")
	if !field.IsValid() {
		t.Fatal("tui.Model missing sessionDirectory field")
	}
	if field.IsNil() {
		return nil
	}

	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.SessionDirectoryFunc)
}

func capturedTUISessionResume(t *testing.T, model tea.Model) tui.SessionResumeFunc {
	t.Helper()

	m, ok := model.(tui.Model)
	if !ok {
		t.Fatalf("captured model type = %T, want tui.Model", model)
	}

	field := reflect.ValueOf(&m).Elem().FieldByName("sessionResume")
	if !field.IsValid() {
		t.Fatal("tui.Model missing sessionResume field")
	}
	if field.IsNil() {
		return nil
	}

	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.SessionResumeFunc)
}
