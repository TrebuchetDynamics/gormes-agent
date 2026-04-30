package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTodoToolValidateAndDedupe(t *testing.T) {
	store := NewTodoStore(t.TempDir())

	result := store.Write([]TodoItem{
		{ID: "1", Content: "first version", Status: TodoStatusPending},
		{ID: "2", Content: "", Status: TodoStatus("not-a-status")},
		{ID: "1", Content: "latest version", Status: TodoStatusInProgress},
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
	if got, want := result.Todos[0].Status, TodoStatusPending; got != want {
		t.Fatalf("todos[0].Status = %q, want %q", got, want)
	}
	if got, want := result.Todos[1], (TodoItem{ID: "1", Content: "latest version", Status: TodoStatusInProgress}); got != want {
		t.Fatalf("todos[1] = %+v, want %+v", got, want)
	}
}

func TestTodoToolReplaceAndMerge(t *testing.T) {
	store := NewTodoStore(t.TempDir())

	store.Write([]TodoItem{
		{ID: "1", Content: "original", Status: TodoStatusPending},
		{ID: "2", Content: "second", Status: TodoStatusPending},
	}, false)

	merged := store.Write([]TodoItem{
		{ID: "1", Status: TodoStatusCompleted},
		{ID: "3", Content: "third", Status: TodoStatusPending},
	}, true)

	if got, want := merged.Todos, []TodoItem{
		{ID: "1", Content: "original", Status: TodoStatusCompleted},
		{ID: "2", Content: "second", Status: TodoStatusPending},
		{ID: "3", Content: "third", Status: TodoStatusPending},
	}; !equalTodoItems(got, want) {
		t.Fatalf("merge todos = %+v, want %+v", got, want)
	}

	replaced := store.Write([]TodoItem{
		{ID: "4", Content: "replacement", Status: TodoStatusInProgress},
	}, false)

	if got, want := replaced.Todos, []TodoItem{
		{ID: "4", Content: "replacement", Status: TodoStatusInProgress},
	}; !equalTodoItems(got, want) {
		t.Fatalf("replace todos = %+v, want %+v", got, want)
	}
}

func TestTodoToolFormatForInjectionOnlyActive(t *testing.T) {
	store := NewTodoStore(t.TempDir())
	store.Write([]TodoItem{
		{ID: "1", Content: "done", Status: TodoStatusCompleted},
		{ID: "2", Content: "next", Status: TodoStatusPending},
		{ID: "3", Content: "working", Status: TodoStatusInProgress},
		{ID: "4", Content: "cancelled", Status: TodoStatusCancelled},
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

	store.Write([]TodoItem{{ID: "5", Content: "also done", Status: TodoStatusCompleted}}, false)
	if text := store.FormatForInjection(); text != "" {
		t.Fatalf("inactive-only injection = %q, want empty", text)
	}
}

func TestTodoToolSummaryCounts(t *testing.T) {
	tool := NewTodoTool(TodoToolConfig{Store: NewTodoStore(t.TempDir())})
	var _ Tool = tool

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

	var result TodoToolResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal result: %v\n%s", err, raw)
	}
	if got, want := result.Summary, (TodoSummary{Total: 4, Pending: 1, InProgress: 1, Completed: 1, Cancelled: 1}); got != want {
		t.Fatalf("summary = %+v, want %+v", got, want)
	}
	if got, want := len(result.Todos), 4; got != want {
		t.Fatalf("todos len = %d, want %d", got, want)
	}

	if err := json.Unmarshal(tool.Schema(), new(map[string]any)); err != nil {
		t.Fatalf("tool schema invalid JSON: %v", err)
	}

	reg := NewRegistry()
	reg.MustRegister(tool)
	registered, ok := reg.Get(TodoToolName)
	if !ok {
		t.Fatal("registered todo tool not found")
	}
	if registered.Name() != TodoToolName {
		t.Fatalf("registered tool name = %q, want %q", registered.Name(), TodoToolName)
	}
}

func TestTodoToolStoreCorruptionDegrades(t *testing.T) {
	store := NewTodoStore(t.TempDir())
	path := store.Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir todo store: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"todos":`), 0o640); err != nil {
		t.Fatalf("write corrupt todo store: %v", err)
	}

	result := store.Read()
	if got, want := result.Evidence, TodoEvidenceStoreCorrupt; got != want {
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

func equalTodoItems(got []TodoItem, want []TodoItem) bool {
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
