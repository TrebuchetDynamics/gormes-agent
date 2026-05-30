package todo

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

func TestTodoToolValidateAndDedupe(t *testing.T) {
	store := NewStore(t.TempDir())

	result := store.Write([]Item{
		{ID: "1", Content: "first version", Status: StatusPending},
		{ID: "2", Content: "", Status: Status("not-a-status")},
		{ID: "1", Content: "latest version", Status: StatusInProgress},
	}, false)

	if result.Evidence != "" {
		t.Fatalf("Write Evidence = %q, want empty", result.Evidence)
	}
	if got, want := len(result.Todos), 2; got != want {
		t.Fatalf("todos len = %d, want %d: %+v", got, want, result.Todos)
	}
	if got, want := result.Todos[0].ID, "2"; got != want {
		t.Fatalf("todos[0].ID = %q, want %q", got, want)
	}
	if got, want := result.Todos[0].Content, "(no description)"; got != want {
		t.Fatalf("todos[0].Content = %q, want %q", got, want)
	}
	if got, want := result.Todos[0].Status, StatusPending; got != want {
		t.Fatalf("todos[0].Status = %q, want %q", got, want)
	}
	if got, want := result.Todos[1], (Item{ID: "1", Content: "latest version", Status: StatusInProgress}); got != want {
		t.Fatalf("todos[1] = %+v, want %+v", got, want)
	}
}

func TestTodoToolReplaceAndMerge(t *testing.T) {
	store := NewStore(t.TempDir())

	store.Write([]Item{
		{ID: "1", Content: "original", Status: StatusPending},
		{ID: "2", Content: "second", Status: StatusPending},
	}, false)

	merged := store.Write([]Item{
		{ID: "1", Status: StatusCompleted},
		{ID: "3", Content: "third", Status: StatusPending},
	}, true)

	if got, want := merged.Todos, []Item{
		{ID: "1", Content: "original", Status: StatusCompleted},
		{ID: "2", Content: "second", Status: StatusPending},
		{ID: "3", Content: "third", Status: StatusPending},
	}; !equalItems(got, want) {
		t.Fatalf("merge todos = %+v, want %+v", got, want)
	}

	replaced := store.Write([]Item{
		{ID: "4", Content: "replacement", Status: StatusInProgress},
	}, false)

	if got, want := replaced.Todos, []Item{
		{ID: "4", Content: "replacement", Status: StatusInProgress},
	}; !equalItems(got, want) {
		t.Fatalf("replace todos = %+v, want %+v", got, want)
	}
}

func TestTodoToolFormatForInjectionOnlyActive(t *testing.T) {
	store := NewStore(t.TempDir())
	store.Write([]Item{
		{ID: "1", Content: "done", Status: StatusCompleted},
		{ID: "2", Content: "next", Status: StatusPending},
		{ID: "3", Content: "working", Status: StatusInProgress},
		{ID: "4", Content: "cancelled", Status: StatusCancelled},
	}, false)

	text := store.FormatForInjection()
	if !strings.Contains(strings.ToLower(text), "context compression") {
		t.Fatalf("injection text = %q, want context compression header", text)
	}
	for _, want := range []string{"[ ] 2. next (pending)", "[>] 3. working (in_progress)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("injection text = %q, want %q", text, want)
		}
	}
	for _, forbidden := range []string{"done", "cancelled", "[x]", "[~]"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("injection text = %q, must omit %q", text, forbidden)
		}
	}

	store.Write([]Item{{ID: "5", Content: "also done", Status: StatusCompleted}}, false)
	if text := store.FormatForInjection(); text != "" {
		t.Fatalf("inactive-only injection = %q, want empty", text)
	}
}

func TestTodoToolSummaryCounts(t *testing.T) {
	tool := NewTool(Config{Store: NewStore(t.TempDir())})
	var _ toolkit.Tool = tool

	emptyRaw, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute empty read: %v", err)
	}
	if !strings.Contains(string(emptyRaw), `"todos":[]`) {
		t.Fatalf("empty read = %s, want todos array", emptyRaw)
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"todos":[
		{"id":"1","content":"pending","status":"pending"},
		{"id":"2","content":"working","status":"in_progress"},
		{"id":"3","content":"done","status":"completed"},
		{"id":"4","content":"cancelled","status":"cancelled"}
	]}`))
	if err != nil {
		t.Fatalf("Execute write: %v", err)
	}

	var result Result
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal result: %v\n%s", err, raw)
	}
	if got, want := result.Summary, (Summary{Total: 4, Pending: 1, InProgress: 1, Completed: 1, Cancelled: 1}); got != want {
		t.Fatalf("summary = %+v, want %+v", got, want)
	}
	if got, want := len(result.Todos), 4; got != want {
		t.Fatalf("todos len = %d, want %d", got, want)
	}

	if err := json.Unmarshal(tool.Schema(), new(map[string]any)); err != nil {
		t.Fatalf("tool schema invalid JSON: %v", err)
	}

	reg := toolkit.NewRegistry()
	reg.MustRegister(tool)
	registered, ok := reg.Get(ToolName)
	if !ok {
		t.Fatal("registered todo tool not found")
	}
	if registered.Name() != ToolName {
		t.Fatalf("registered tool name = %q, want %q", registered.Name(), ToolName)
	}
}

func TestTodoToolStoreCorruptionDegrades(t *testing.T) {
	store := NewStore(t.TempDir())
	path := store.Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir todo store: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"todos":`), 0o640); err != nil {
		t.Fatalf("write corrupt todo store: %v", err)
	}

	result := store.Read()
	if got, want := result.Evidence, EvidenceStoreCorrupt; got != want {
		t.Fatalf("Evidence = %q, want %q; result=%+v", got, want, result)
	}
	if result.Error == "" {
		t.Fatalf("Error is empty for corrupt store: %+v", result)
	}
	if got := len(result.Todos); got != 0 {
		t.Fatalf("corrupt read todos len = %d, want 0", got)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read corrupt todo store after degraded read: %v", err)
	}
	if got, want := string(raw), `{"todos":`; got != want {
		t.Fatalf("corrupt store content changed = %q, want %q", got, want)
	}
}

func equalItems(got []Item, want []Item) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
