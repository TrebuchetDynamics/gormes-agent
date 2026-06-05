package tuiadapter

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"unsafe"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestLocalSessionBundleInjectsSessionDirectoryAndResumeAdapters(t *testing.T) {
	seedLocalSessionBundleDB(t, []localSessionSeed{
		{id: "sess-alpha", title: "Alpha Work", role: "user", content: "preview alpha", ts: 100},
		{id: "sess-beta", title: "Beta Work", role: "user", content: "preview beta", ts: 200, turnKey: "beta-user"},
		{id: "sess-beta", title: "Beta Work", role: "user", content: "preview beta", ts: 200, chatID: "user"},
		{id: "sess-resume", role: "user", content: "resume question", ts: 300},
		{id: "sess-resume", role: "assistant", content: "resume answer", ts: 301},
	})
	writeLocalSessionBundleMirrorIndex(t, "sess-beta", "telegram")

	var resumedSession string
	model := localSessionBundleModelForTest(t, LocalSessionBundleOptions{
		RootContext: context.Background(),
		Resume: func(sessionID string, _ []llm.Message) error {
			resumedSession = sessionID
			return nil
		},
	})

	directory := capturedTUISessionDirectory(t, model)
	if directory == nil {
		t.Fatal("local TUI SessionDirectory = nil, want memory-backed sessions adapter")
	}
	entries, err := directory(1)
	if err != nil {
		t.Fatalf("SessionDirectory: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "sess-resume" || entries[0].MessageCount != 2 {
		t.Fatalf("SessionDirectory entries = %+v, want newest resume session with two messages", entries)
	}

	entries, err = directory(2)
	if err != nil {
		t.Fatalf("SessionDirectory: %v", err)
	}
	if len(entries) != 2 || entries[1].ID != "sess-beta" || entries[1].Title != "Beta Work" || entries[1].Preview != "preview beta" || entries[1].Source != "telegram" || entries[1].MessageCount != 1 {
		t.Fatalf("SessionDirectory entries = %+v, want deduped Telegram Beta Work as second entry", entries)
	}

	resume := capturedTUISessionResume(t, model)
	if resume == nil {
		t.Fatal("local TUI SessionResume = nil, want memory-backed resume adapter")
	}
	result, err := resume(context.Background(), "sess-res")
	if err != nil {
		t.Fatalf("SessionResume: %v", err)
	}
	if result.SessionID != "sess-resume" || resumedSession != "sess-resume" {
		t.Fatalf("SessionResume session = result:%q callback:%q, want sess-resume", result.SessionID, resumedSession)
	}
	if len(result.History) != 2 || result.History[0].Content != "resume question" || result.History[1].Content != "resume answer" {
		t.Fatalf("SessionResume History = %+v, want replayed resume transcript", result.History)
	}
}

func TestLocalSessionBundleInjectsBranchAndTreeAdapters(t *testing.T) {
	seedLocalSessionBundleDB(t, []localSessionSeed{
		{id: "sess-root", title: "Root Plan", role: "user", content: "root prompt", ts: 100},
		{id: "sess-root", role: "assistant", content: "root answer", ts: 101},
		{id: "sess-fork", role: "user", content: "fork prompt", ts: 120},
	})
	boltMap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	defer boltMap.Close()
	ctx := context.Background()
	if err := boltMap.PutMetadata(ctx, session.Metadata{SessionID: "sess-root", Title: "Root Plan", LineageKind: session.LineageKindPrimary, CreatedAt: 100, UpdatedAt: 110, Labels: []string{"pinned"}}); err != nil {
		t.Fatalf("put root metadata: %v", err)
	}
	if err := boltMap.PutMetadata(ctx, session.Metadata{SessionID: "sess-fork", Title: "Branch", ParentSessionID: "sess-root", LineageKind: session.LineageKindFork, CreatedAt: 120, UpdatedAt: 130}); err != nil {
		t.Fatalf("put fork metadata: %v", err)
	}

	model := localSessionBundleModelForTest(t, LocalSessionBundleOptions{
		RootContext: context.Background(),
		Metadata:    boltMap,
		Resume:      func(string, []llm.Message) error { return nil },
		Reset:       func() error { return nil },
	})
	if branch := capturedTUISessionBranch(t, model); branch == nil {
		t.Fatal("local TUI SessionBranch = nil, want memory-backed branch adapter")
	}
	tree := capturedTUISessionTree(t, model)
	label := capturedTUISessionTreeLabel(t, model)
	restore := capturedTUISessionTreeRestore(t, model)
	if tree == nil || label == nil || restore == nil {
		t.Fatalf("tree adapters available: tree=%v label=%v restore=%v, want all true", tree != nil, label != nil, restore != nil)
	}

	treeResult, err := tree(context.Background(), tui.SessionTreeRequest{Filter: "all-equivalent", ActiveSessionID: "sess-fork"})
	if err != nil {
		t.Fatalf("SessionTree: %v", err)
	}
	labelResult, err := label(context.Background(), tui.SessionTreeLabelRequest{SessionID: "sess-root", Action: "set", Label: "review"})
	if err != nil {
		t.Fatalf("SessionTreeLabel: %v", err)
	}
	restoreResult, err := restore(context.Background(), tui.SessionTreeRestoreRequest{SessionID: "sess-root", MessageID: 1})
	if err != nil {
		t.Fatalf("SessionTreeRestore: %v", err)
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

func TestPlainTUIModelHasNoLocalSessionAdapters(t *testing.T) {
	model := tui.NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, tui.Options{})
	if directory := capturedTUISessionDirectory(t, model); directory != nil {
		t.Fatal("plain/remote TUI SessionDirectory is non-nil; only local startup should inject sessions adapter")
	}
	if resume := capturedTUISessionResume(t, model); resume != nil {
		t.Fatal("plain/remote TUI SessionResume is non-nil; only local startup should inject resume adapter")
	}
	if branch := capturedTUISessionBranch(t, model); branch != nil {
		t.Fatal("plain/remote TUI SessionBranch is non-nil; only local startup should inject branch adapter")
	}
	if tree := capturedTUISessionTree(t, model); tree != nil {
		t.Fatal("plain/remote TUI SessionTree is non-nil; only local startup should inject tree adapter")
	}
}

func localSessionBundleModelForTest(t *testing.T, bundleOpts LocalSessionBundleOptions) tui.Model {
	t.Helper()
	opts := tui.Options{}
	NewLocalSessionBundle(bundleOpts).Apply(&opts)
	return tui.NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, opts)
}

func capturedTUISessionDirectory(t *testing.T, model tui.Model) tui.SessionDirectoryFunc {
	t.Helper()
	field := capturedTUIModelField(t, model, "sessionDirectory")
	if field.IsNil() {
		return nil
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.SessionDirectoryFunc)
}

func capturedTUISessionResume(t *testing.T, model tui.Model) tui.SessionResumeFunc {
	t.Helper()
	field := capturedTUIModelField(t, model, "sessionResume")
	if field.IsNil() {
		return nil
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.SessionResumeFunc)
}

func capturedTUISessionBranch(t *testing.T, model tui.Model) tui.SessionBranchFunc {
	t.Helper()
	field := capturedTUIModelField(t, model, "sessionBranch")
	if field.IsNil() {
		return nil
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.SessionBranchFunc)
}

func capturedTUISessionTree(t *testing.T, model tui.Model) tui.SessionTreeFunc {
	t.Helper()
	field := capturedTUIModelField(t, model, "sessionTree")
	if field.IsNil() {
		return nil
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.SessionTreeFunc)
}

func capturedTUISessionTreeLabel(t *testing.T, model tui.Model) tui.SessionTreeLabelFunc {
	t.Helper()
	field := capturedTUIModelField(t, model, "sessionTreeLabel")
	if field.IsNil() {
		return nil
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.SessionTreeLabelFunc)
}

func capturedTUISessionTreeRestore(t *testing.T, model tui.Model) tui.SessionTreeRestoreFunc {
	t.Helper()
	field := capturedTUIModelField(t, model, "sessionTreeRestore")
	if field.IsNil() {
		return nil
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.SessionTreeRestoreFunc)
}

func capturedTUIModelField(t *testing.T, model tui.Model, name string) reflect.Value {
	t.Helper()
	field := reflect.ValueOf(&model).Elem().FieldByName(name)
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

type localSessionSeed struct {
	id      string
	title   string
	role    string
	content string
	ts      int64
	chatID  string
	turnKey string
}

func seedLocalSessionBundleDB(t *testing.T, seeds []localSessionSeed) {
	t.Helper()
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dataHome, "config"))
	t.Setenv("GORMES_HOME", filepath.Join(dataHome, "gormes"))

	store, err := memory.OpenSqlite(config.MemoryDBPath(), 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	for _, seed := range seeds {
		meta := ""
		if seed.title != "" {
			meta = `{"title":"` + seed.title + `"}`
		}
		if _, err := store.DB().Exec(
			`INSERT INTO turns(session_id, role, content, ts_unix, chat_id, meta_json, turn_key) VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))`,
			seed.id, seed.role, seed.content, seed.ts, seed.chatID, meta, seed.turnKey,
		); err != nil {
			t.Fatalf("seed session %s: %v", seed.id, err)
		}
	}
}

func writeLocalSessionBundleMirrorIndex(t *testing.T, sessionID, source string) {
	t.Helper()
	path := config.SessionIndexMirrorPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create session mirror dir: %v", err)
	}
	body := "# Auto-generated session index\nsessions:\n  " + source + ":42: " + sessionID + "\nlineage:\n  " + sessionID + ":\n    lineage_kind: primary\n    lineage_status: ok\n    source: " + source + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write session mirror: %v", err)
	}
}
