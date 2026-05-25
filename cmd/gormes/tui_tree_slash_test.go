package main

import (
	"context"
	"reflect"
	"testing"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUISessionTreeBindingLocalModelReceivesSessionTreeAdapters(t *testing.T) {
	seedSessionsCommandDB(t, []sessionCommandSeed{
		{id: "sess-root", title: "Root Plan", role: "user", content: "root prompt", ts: 100},
		{id: "sess-root", role: "assistant", content: "root answer", ts: 101},
		{id: "sess-fork", role: "user", content: "fork prompt", ts: 120},
	})
	boltMap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	ctx := context.Background()
	if err := boltMap.PutMetadata(ctx, session.Metadata{SessionID: "sess-root", Title: "Root Plan", LineageKind: session.LineageKindPrimary, CreatedAt: 100, UpdatedAt: 110, Labels: []string{"pinned"}}); err != nil {
		t.Fatalf("put root metadata: %v", err)
	}
	if err := boltMap.PutMetadata(ctx, session.Metadata{SessionID: "sess-fork", Title: "Branch", ParentSessionID: "sess-root", LineageKind: session.LineageKindFork, CreatedAt: 120, UpdatedAt: 130}); err != nil {
		t.Fatalf("put fork metadata: %v", err)
	}
	if err := boltMap.Close(); err != nil {
		t.Fatalf("close seeded bolt map: %v", err)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}

	var sawTree, sawLabel, sawRestore bool
	var treeResult tui.SessionTreeResult
	var labelResult tui.SessionTreeLabelResult
	var restoreResult tui.SessionTreeRestoreResult
	err = runResolvedTUIWithRuntime(cmd, tuiInvocation{Config: cfg}, rootRuntime{
		tuiProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) tuiProgram {
			return fakeTUIProgram{run: func() {
				tree := capturedTUISessionTree(t, model)
				label := capturedTUISessionTreeLabel(t, model)
				restore := capturedTUISessionTreeRestore(t, model)
				if tree != nil {
					sawTree = true
					treeResult, err = tree(context.Background(), tui.SessionTreeRequest{Filter: "all-equivalent", ActiveSessionID: "sess-fork"})
					if err != nil {
						t.Fatalf("SessionTree: %v", err)
					}
				}
				if label != nil {
					sawLabel = true
					labelResult, err = label(context.Background(), tui.SessionTreeLabelRequest{SessionID: "sess-root", Action: "set", Label: "review"})
					if err != nil {
						t.Fatalf("SessionTreeLabel: %v", err)
					}
				}
				if restore != nil {
					sawRestore = true
					restoreResult, err = restore(context.Background(), tui.SessionTreeRestoreRequest{SessionID: "sess-root", MessageID: 1})
					if err != nil {
						t.Fatalf("SessionTreeRestore: %v", err)
					}
				}
			}}
		},
	})
	if err != nil {
		t.Fatalf("runResolvedTUIWithRuntime: %v", err)
	}
	if !sawTree || !sawLabel || !sawRestore {
		t.Fatalf("tree adapters available: tree=%v label=%v restore=%v, want all true", sawTree, sawLabel, sawRestore)
	}
	if gotIDs := tuiTreeIDs(treeResult.Entries); !reflect.DeepEqual(gotIDs, []string{"sess-root", "sess-fork"}) {
		t.Fatalf("tree IDs = %v, want root/fork", gotIDs)
	}
	if !treeResult.Entries[1].Active || treeResult.Entries[1].ParentID != "sess-root" || treeResult.Entries[1].LineageKind != session.LineageKindFork {
		t.Fatalf("fork entry = %+v, want active fork child", treeResult.Entries[1])
	}
	if len(treeResult.Entries[0].Labels) == 0 || treeResult.Entries[0].Labels[0] != "pinned" {
		t.Fatalf("root labels = %v, want pinned", treeResult.Entries[0].Labels)
	}
	if roles := tuiTreeMessageRoles(treeResult.Entries[0].Messages); !reflect.DeepEqual(roles, []string{"user", "assistant"}) {
		t.Fatalf("root message roles = %v, want all-equivalent user+assistant", roles)
	}
	if labelResult.SessionID != "sess-root" || !reflect.DeepEqual(labelResult.Labels, []string{"pinned", "review"}) {
		t.Fatalf("label result = %+v, want pinned+review", labelResult)
	}
	if !restoreResult.Editable || restoreResult.Text != "root prompt" || restoreResult.Evidence != "" {
		t.Fatalf("restore result = %+v, want editable root prompt", restoreResult)
	}
}

func TestTUISessionTreeBindingRemoteTUIUnchanged(t *testing.T) {
	model := tui.NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, tui.Options{})
	if tree := capturedTUISessionTree(t, model); tree != nil {
		t.Fatal("plain/remote TUI SessionTree is non-nil; only local startup should inject tree adapter")
	}
}

func capturedTUISessionTree(t *testing.T, model tea.Model) tui.SessionTreeFunc {
	t.Helper()
	field := capturedTUIModelField(t, model, "sessionTree")
	if field.IsNil() {
		return nil
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.SessionTreeFunc)
}

func capturedTUISessionTreeLabel(t *testing.T, model tea.Model) tui.SessionTreeLabelFunc {
	t.Helper()
	field := capturedTUIModelField(t, model, "sessionTreeLabel")
	if field.IsNil() {
		return nil
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.SessionTreeLabelFunc)
}

func capturedTUISessionTreeRestore(t *testing.T, model tea.Model) tui.SessionTreeRestoreFunc {
	t.Helper()
	field := capturedTUIModelField(t, model, "sessionTreeRestore")
	if field.IsNil() {
		return nil
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.SessionTreeRestoreFunc)
}

func capturedTUIModelField(t *testing.T, model tea.Model, name string) reflect.Value {
	t.Helper()
	m, ok := model.(tui.Model)
	if !ok {
		t.Fatalf("captured model type = %T, want tui.Model", model)
	}
	field := reflect.ValueOf(&m).Elem().FieldByName(name)
	if !field.IsValid() {
		t.Fatalf("tui.Model missing %s field", name)
	}
	return field
}

func tuiTreeIDs(entries []tui.SessionTreeEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids
}

func tuiTreeMessageRoles(messages []tui.SessionTreeMessage) []string {
	roles := make([]string, 0, len(messages))
	for _, msg := range messages {
		roles = append(roles, msg.Role)
	}
	return roles
}
