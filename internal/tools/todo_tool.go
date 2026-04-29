package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const TodoToolName = "todo"

type TodoStatus string

const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
	TodoStatusCancelled  TodoStatus = "cancelled"
)

const (
	TodoEvidenceInvalidArgs      = "todo_invalid_args"
	TodoEvidenceStoreUnavailable = "todo_store_unavailable"
	TodoEvidenceStoreCorrupt     = "todo_store_corrupt"
)

// TodoItem is one priority-ordered task in the session todo list.
type TodoItem struct {
	ID      string     `json:"id"`
	Content string     `json:"content"`
	Status  TodoStatus `json:"status"`
}

// TodoSummary mirrors the Hermes todo_tool summary payload.
type TodoSummary struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	InProgress int `json:"in_progress"`
	Completed  int `json:"completed"`
	Cancelled  int `json:"cancelled"`
}

// TodoToolResult is the JSON payload returned by TodoTool and TodoStore
// helpers. Evidence/Error are populated only for degraded-mode results.
type TodoToolResult struct {
	Todos    []TodoItem  `json:"todos"`
	Summary  TodoSummary `json:"summary"`
	Evidence string      `json:"evidence,omitempty"`
	Error    string      `json:"error,omitempty"`
}

// TodoStore persists a single session's todo list under an injected root.
type TodoStore struct {
	root string
	mu   sync.Mutex
}

func NewTodoStore(root string) *TodoStore {
	return &TodoStore{root: root}
}

func (s *TodoStore) Path() string {
	path, err := s.todoPath()
	if err != nil {
		return ""
	}
	return path
}

func (s *TodoStore) Read() TodoToolResult {
	if s == nil {
		return todoToolError(TodoEvidenceStoreUnavailable, "todo store is not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	items, evidence, message := s.loadLocked()
	if evidence != "" {
		return todoToolError(evidence, message)
	}
	return todoResult(items)
}

func (s *TodoStore) Write(todos []TodoItem, merge bool) TodoToolResult {
	if s == nil {
		return todoToolError(TodoEvidenceStoreUnavailable, "todo store is not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, evidence, message := s.loadLocked()
	if evidence != "" {
		return todoToolError(evidence, message)
	}

	var items []TodoItem
	if merge {
		items = mergeTodoItems(existing, todos)
	} else {
		items = validateTodoItems(dedupeTodoItems(todos))
	}

	if evidence, message := s.saveLocked(items); evidence != "" {
		return todoToolError(evidence, message)
	}
	return todoResult(items)
}

func (s *TodoStore) HasItems() bool {
	return len(s.Read().Todos) > 0
}

func (s *TodoStore) FormatForInjection() string {
	return FormatTodoItemsForInjection(s.Read().Todos)
}

func (s *TodoStore) todoPath() (string, error) {
	root := strings.TrimSpace(s.root)
	if root == "" {
		return "", errors.New("todo store root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve todo store root: %w", err)
	}
	return filepath.Join(absRoot, "todos.json"), nil
}

func (s *TodoStore) loadLocked() ([]TodoItem, string, string) {
	path, err := s.todoPath()
	if err != nil {
		return nil, TodoEvidenceStoreUnavailable, err.Error()
	}

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", ""
	}
	if err != nil {
		return nil, TodoEvidenceStoreUnavailable, fmt.Sprintf("read todo store: %v", err)
	}

	var file todoStoreFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, TodoEvidenceStoreCorrupt, fmt.Sprintf("decode todo store: %v", err)
	}
	return validateTodoItems(dedupeTodoItems(file.Todos)), "", ""
}

func (s *TodoStore) saveLocked(items []TodoItem) (string, string) {
	path, err := s.todoPath()
	if err != nil {
		return TodoEvidenceStoreUnavailable, err.Error()
	}
	root := filepath.Dir(path)
	if err := os.MkdirAll(root, 0o750); err != nil {
		return TodoEvidenceStoreUnavailable, fmt.Sprintf("create todo store: %v", err)
	}

	payload, err := json.MarshalIndent(todoStoreFile{Todos: cloneTodoItems(items)}, "", "  ")
	if err != nil {
		return TodoEvidenceInvalidArgs, fmt.Sprintf("encode todo store: %v", err)
	}

	tmp, err := os.CreateTemp(root, ".todos-*.json")
	if err != nil {
		return TodoEvidenceStoreUnavailable, fmt.Sprintf("create todo temp file: %v", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return TodoEvidenceStoreUnavailable, fmt.Sprintf("write todo temp file: %v", err)
	}
	if err := tmp.Close(); err != nil {
		return TodoEvidenceStoreUnavailable, fmt.Sprintf("close todo temp file: %v", err)
	}
	if err := os.Chmod(tmpName, 0o640); err != nil {
		return TodoEvidenceStoreUnavailable, fmt.Sprintf("chmod todo temp file: %v", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return TodoEvidenceStoreUnavailable, fmt.Sprintf("replace todo store: %v", err)
	}
	return "", ""
}

type todoStoreFile struct {
	Todos []TodoItem `json:"todos"`
}

// TodoToolConfig wires the stateful native todo tool to a per-session store.
type TodoToolConfig struct {
	Store *TodoStore
}

type TodoTool struct {
	cfg TodoToolConfig
}

func NewTodoTool(cfg TodoToolConfig) *TodoTool {
	return &TodoTool{cfg: cfg}
}

func (*TodoTool) Name() string { return TodoToolName }

func (*TodoTool) Description() string {
	return "Manage the current session task list. Use for complex tasks, preserve priority order, and return the full list."
}

func (*TodoTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"todos":{"type":"array","description":"Task items to write. Omit to read current list.","items":{"type":"object","properties":{"id":{"type":"string","description":"Unique item identifier"},"content":{"type":"string","description":"Task description"},"status":{"type":"string","enum":["pending","in_progress","completed","cancelled"],"description":"Current status"}},"required":["id","content","status"]}},"merge":{"type":"boolean","description":"true: update existing items by id, add new ones. false (default): replace the entire list.","default":false}},"required":[]}`)
}

func (*TodoTool) Timeout() time.Duration { return 0 }

func (t *TodoTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	_ = ctx
	if t == nil || t.cfg.Store == nil {
		return json.Marshal(todoToolError(TodoEvidenceStoreUnavailable, "todo store is not initialized"))
	}

	var in todoToolArgs
	if len(strings.TrimSpace(string(args))) == 0 {
		args = json.RawMessage(`{}`)
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return json.Marshal(todoToolError(TodoEvidenceInvalidArgs, "invalid todo args: "+err.Error()))
	}

	var result TodoToolResult
	if in.Todos != nil {
		result = t.cfg.Store.Write(*in.Todos, in.Merge)
	} else {
		result = t.cfg.Store.Read()
	}
	return json.Marshal(result)
}

type todoToolArgs struct {
	Todos *[]TodoItem `json:"todos"`
	Merge bool        `json:"merge"`
}

func validateTodoItems(items []TodoItem) []TodoItem {
	out := make([]TodoItem, 0, len(items))
	for _, item := range items {
		out = append(out, validateTodoItem(item))
	}
	return out
}

func validateTodoItem(item TodoItem) TodoItem {
	id := strings.TrimSpace(item.ID)
	if id == "" {
		id = "?"
	}

	content := strings.TrimSpace(item.Content)
	if content == "" {
		content = "(no description)"
	}

	status := normalizeTodoStatus(item.Status)
	return TodoItem{ID: id, Content: content, Status: status}
}

func normalizeTodoStatus(status TodoStatus) TodoStatus {
	switch TodoStatus(strings.ToLower(strings.TrimSpace(string(status)))) {
	case TodoStatusInProgress:
		return TodoStatusInProgress
	case TodoStatusCompleted:
		return TodoStatusCompleted
	case TodoStatusCancelled:
		return TodoStatusCancelled
	case TodoStatusPending:
		return TodoStatusPending
	default:
		return TodoStatusPending
	}
}

func validTodoStatus(status TodoStatus) (TodoStatus, bool) {
	normalized := TodoStatus(strings.ToLower(strings.TrimSpace(string(status))))
	switch normalized {
	case TodoStatusPending, TodoStatusInProgress, TodoStatusCompleted, TodoStatusCancelled:
		return normalized, true
	default:
		return "", false
	}
}

func dedupeTodoItems(items []TodoItem) []TodoItem {
	lastIndex := make(map[string]int, len(items))
	for i, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = "?"
		}
		lastIndex[id] = i
	}

	indexes := make([]int, 0, len(lastIndex))
	for _, idx := range lastIndex {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)

	out := make([]TodoItem, 0, len(indexes))
	for _, idx := range indexes {
		out = append(out, items[idx])
	}
	return out
}

func mergeTodoItems(existing []TodoItem, patches []TodoItem) []TodoItem {
	items := cloneTodoItems(validateTodoItems(existing))
	indexByID := make(map[string]int, len(items))
	for i, item := range items {
		indexByID[item.ID] = i
	}

	for _, patch := range dedupeTodoItems(patches) {
		id := strings.TrimSpace(patch.ID)
		if id == "" {
			continue
		}
		if idx, ok := indexByID[id]; ok {
			if content := strings.TrimSpace(patch.Content); content != "" {
				items[idx].Content = content
			}
			if status, ok := validTodoStatus(patch.Status); ok {
				items[idx].Status = status
			}
			continue
		}

		item := validateTodoItem(patch)
		indexByID[item.ID] = len(items)
		items = append(items, item)
	}
	return items
}

func todoResult(items []TodoItem) TodoToolResult {
	items = cloneTodoItems(items)
	return TodoToolResult{
		Todos:   items,
		Summary: summarizeTodoItems(items),
	}
}

func todoToolError(evidence string, message string) TodoToolResult {
	return TodoToolResult{
		Todos:    []TodoItem{},
		Summary:  TodoSummary{},
		Evidence: evidence,
		Error:    message,
	}
}

func summarizeTodoItems(items []TodoItem) TodoSummary {
	var out TodoSummary
	out.Total = len(items)
	for _, item := range items {
		switch item.Status {
		case TodoStatusInProgress:
			out.InProgress++
		case TodoStatusCompleted:
			out.Completed++
		case TodoStatusCancelled:
			out.Cancelled++
		default:
			out.Pending++
		}
	}
	return out
}

func FormatTodoItemsForInjection(items []TodoItem) string {
	if len(items) == 0 {
		return ""
	}

	markers := map[TodoStatus]string{
		TodoStatusCompleted:  "[x]",
		TodoStatusInProgress: "[>]",
		TodoStatusPending:    "[ ]",
		TodoStatusCancelled:  "[~]",
	}

	lines := []string{"[Your active task list was preserved across context compression]"}
	for _, item := range validateTodoItems(items) {
		if item.Status != TodoStatusPending && item.Status != TodoStatusInProgress {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s %s. %s (%s)", markers[item.Status], item.ID, item.Content, item.Status))
	}
	if len(lines) == 1 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func cloneTodoItems(items []TodoItem) []TodoItem {
	out := make([]TodoItem, len(items))
	copy(out, items)
	return out
}
