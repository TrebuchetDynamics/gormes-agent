package main

import (
	"context"
	"reflect"
	"testing"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
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

func TestTUIBranchSlashAdapterForksTranscriptAndResumesKernelSession(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "sess-parent", role: "user", content: "persisted question", ts: 100},
		{id: "sess-parent", role: "assistant", content: "persisted answer", ts: 101},
	})
	boltMap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	defer boltMap.Close()

	var resumedSession string
	var resumedHistory []hermes.Message
	branch := newTUIBranchFuncWithID(context.Background(), boltMap, func(sessionID string, history []hermes.Message) error {
		resumedSession = sessionID
		resumedHistory = append([]hermes.Message(nil), history...)
		return nil
	}, func() string { return "sess-child" })

	result, err := branch(context.Background(), tui.BranchRequest{
		ParentSessionID: "sess-parent",
		Title:           "branch title",
		History: []hermes.Message{
			{Role: "user", Content: "visible question"},
			{Role: "assistant", Content: "visible answer"},
		},
	})
	if err != nil {
		t.Fatalf("SessionBranch: %v", err)
	}
	if result.SessionID != "sess-child" || result.ParentSessionID != "sess-parent" || result.Title != "branch title" || result.TranscriptCopied != 2 {
		t.Fatalf("BranchResult = %+v, want child/parent/title/copied evidence", result)
	}
	if resumedSession != "sess-child" {
		t.Fatalf("resumed session = %q, want sess-child", resumedSession)
	}
	if len(resumedHistory) != 2 || resumedHistory[0].Content != "visible question" || resumedHistory[1].Content != "visible answer" {
		t.Fatalf("resumed history = %+v, want visible frame history", resumedHistory)
	}
	assertSessionCommandTurnCount(t, "sess-child", 2)

	meta, ok, err := boltMap.GetMetadata(context.Background(), "sess-child")
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if !ok || meta.ParentSessionID != "sess-parent" || meta.LineageKind != session.LineageKindFork {
		t.Fatalf("metadata = %+v ok=%v, want fork child metadata", meta, ok)
	}
}

func TestTUIBranchSlashAdapterFallsBackToCopiedTranscriptWhenVisibleHistoryEmpty(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "sess-parent", role: "user", content: "persisted question", ts: 100},
		{id: "sess-parent", role: "assistant", content: "persisted answer", ts: 101},
	})
	boltMap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	defer boltMap.Close()

	var resumedHistory []hermes.Message
	branch := newTUIBranchFuncWithID(context.Background(), boltMap, func(_ string, history []hermes.Message) error {
		resumedHistory = append([]hermes.Message(nil), history...)
		return nil
	}, func() string { return "sess-child" })

	if _, err := branch(context.Background(), tui.BranchRequest{ParentSessionID: "sess-parent"}); err != nil {
		t.Fatalf("SessionBranch: %v", err)
	}
	if len(resumedHistory) != 2 || resumedHistory[0].Content != "persisted question" || resumedHistory[1].Content != "persisted answer" {
		t.Fatalf("resumed history = %+v, want copied persisted transcript", resumedHistory)
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
