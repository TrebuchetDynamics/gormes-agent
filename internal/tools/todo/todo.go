package todo

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

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

const ToolName = "todo"

type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusCancelled  Status = "cancelled"
)

const (
	EvidenceInvalidArgs      = "todo_invalid_args"
	EvidenceStoreUnavailable = "todo_store_unavailable"
	EvidenceStoreCorrupt     = "todo_store_corrupt"
)

// Item is one priority-ordered task in the session todo list.
type Item struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Status  Status `json:"status"`
}

// Summary mirrors the Hermes todo_tool summary payload.
type Summary struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	InProgress int `json:"in_progress"`
	Completed  int `json:"completed"`
	Cancelled  int `json:"cancelled"`
}

// Result is the JSON payload returned by Tool and Store helpers.
// Evidence/Error are populated only for degraded-mode results.
type Result struct {
	Todos    []Item  `json:"todos"`
	Summary  Summary `json:"summary"`
	Evidence string  `json:"evidence,omitempty"`
	Error    string  `json:"error,omitempty"`
}

// Store persists a single session's todo list under an injected root.
type Store struct {
	root string
	mu   sync.Mutex
}

func NewStore(root string) *Store {
	return &Store{root: root}
}

func (s *Store) Path() string {
	path, err := s.todoPath()
	if err != nil {
		return ""
	}
	return path
}

func (s *Store) Read() Result {
	if s == nil {
		return toolError(EvidenceStoreUnavailable, "todo store is not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	items, evidence, message := s.loadLocked()
	if evidence != "" {
		return toolError(evidence, message)
	}
	return result(items)
}

func (s *Store) Write(todos []Item, merge bool) Result {
	if s == nil {
		return toolError(EvidenceStoreUnavailable, "todo store is not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, evidence, message := s.loadLocked()
	if evidence != "" {
		return toolError(evidence, message)
	}

	var items []Item
	if merge {
		items = mergeItems(existing, todos)
	} else {
		items = validateItems(dedupeItems(todos))
	}

	if evidence, message := s.saveLocked(items); evidence != "" {
		return toolError(evidence, message)
	}
	return result(items)
}

func (s *Store) HasItems() bool {
	return len(s.Read().Todos) > 0
}

func (s *Store) FormatForInjection() string {
	return FormatItemsForInjection(s.Read().Todos)
}

func (s *Store) todoPath() (string, error) {
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

func (s *Store) loadLocked() ([]Item, string, string) {
	path, err := s.todoPath()
	if err != nil {
		return nil, EvidenceStoreUnavailable, err.Error()
	}

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", ""
	}
	if err != nil {
		return nil, EvidenceStoreUnavailable, fmt.Sprintf("read todo store: %v", err)
	}

	var file storeFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, EvidenceStoreCorrupt, fmt.Sprintf("decode todo store: %v", err)
	}
	return validateItems(dedupeItems(file.Todos)), "", ""
}

func (s *Store) saveLocked(items []Item) (string, string) {
	path, err := s.todoPath()
	if err != nil {
		return EvidenceStoreUnavailable, err.Error()
	}
	root := filepath.Dir(path)
	if err := os.MkdirAll(root, 0o750); err != nil {
		return EvidenceStoreUnavailable, fmt.Sprintf("create todo store: %v", err)
	}

	payload, err := json.MarshalIndent(storeFile{Todos: cloneItems(items)}, "", "  ")
	if err != nil {
		return EvidenceInvalidArgs, fmt.Sprintf("encode todo store: %v", err)
	}

	tmp, err := os.CreateTemp(root, ".todos-*.json")
	if err != nil {
		return EvidenceStoreUnavailable, fmt.Sprintf("create todo temp file: %v", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return EvidenceStoreUnavailable, fmt.Sprintf("write todo temp file: %v", err)
	}
	if err := tmp.Close(); err != nil {
		return EvidenceStoreUnavailable, fmt.Sprintf("close todo temp file: %v", err)
	}
	if err := os.Chmod(tmpName, 0o640); err != nil {
		return EvidenceStoreUnavailable, fmt.Sprintf("chmod todo temp file: %v", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return EvidenceStoreUnavailable, fmt.Sprintf("replace todo store: %v", err)
	}
	return "", ""
}

type storeFile struct {
	Todos []Item `json:"todos"`
}

// Config wires the stateful native todo tool to a per-session store.
type Config struct {
	Store *Store
}

type Tool struct {
	cfg Config
}

func NewTool(cfg Config) *Tool {
	return &Tool{cfg: cfg}
}

func (*Tool) Name() string { return ToolName }

func (*Tool) Description() string {
	return "Manage the current session task list. Use for complex tasks, preserve priority order, and return the full list."
}

func (*Tool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"todos":{"type":"array","description":"Task items to write. Omit to read current list.","items":{"type":"object","properties":{"id":{"type":"string","description":"Unique item identifier"},"content":{"type":"string","description":"Task description"},"status":{"type":"string","enum":["pending","in_progress","completed","cancelled"],"description":"Current status"}},"required":["id","content","status"]}},"merge":{"type":"boolean","description":"true: update existing items by id, add new ones. false (default): replace the entire list.","default":false}},"required":[]}`)
}

func (*Tool) Timeout() time.Duration { return 0 }

func (t *Tool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	_ = ctx
	if t == nil || t.cfg.Store == nil {
		return json.Marshal(toolError(EvidenceStoreUnavailable, "todo store is not initialized"))
	}

	var in toolArgs
	if len(strings.TrimSpace(string(args))) == 0 {
		args = json.RawMessage(`{}`)
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return json.Marshal(toolError(EvidenceInvalidArgs, "invalid todo args: "+err.Error()))
	}

	var result Result
	if in.Todos != nil {
		result = t.cfg.Store.Write(*in.Todos, in.Merge)
	} else {
		result = t.cfg.Store.Read()
	}
	return json.Marshal(result)
}

type toolArgs struct {
	Todos *[]Item `json:"todos"`
	Merge bool    `json:"merge"`
}

func validateItems(items []Item) []Item {
	out := make([]Item, 0, len(items))
	for _, item := range items {
		out = append(out, validateItem(item))
	}
	return out
}

func validateItem(item Item) Item {
	id := strings.TrimSpace(item.ID)
	if id == "" {
		id = "?"
	}

	content := strings.TrimSpace(item.Content)
	if content == "" {
		content = "(no description)"
	}

	status := normalizeStatus(item.Status)
	return Item{ID: id, Content: content, Status: status}
}

func normalizeStatus(status Status) Status {
	switch Status(strings.ToLower(strings.TrimSpace(string(status)))) {
	case StatusInProgress:
		return StatusInProgress
	case StatusCompleted:
		return StatusCompleted
	case StatusCancelled:
		return StatusCancelled
	case StatusPending:
		return StatusPending
	default:
		return StatusPending
	}
}

func validStatus(status Status) (Status, bool) {
	normalized := Status(strings.ToLower(strings.TrimSpace(string(status))))
	switch normalized {
	case StatusPending, StatusInProgress, StatusCompleted, StatusCancelled:
		return normalized, true
	default:
		return "", false
	}
}

func dedupeItems(items []Item) []Item {
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

	out := make([]Item, 0, len(indexes))
	for _, idx := range indexes {
		out = append(out, items[idx])
	}
	return out
}

func mergeItems(existing []Item, patches []Item) []Item {
	items := cloneItems(validateItems(existing))
	indexByID := make(map[string]int, len(items))
	for i, item := range items {
		indexByID[item.ID] = i
	}

	for _, patch := range dedupeItems(patches) {
		id := strings.TrimSpace(patch.ID)
		if id == "" {
			continue
		}
		if idx, ok := indexByID[id]; ok {
			if content := strings.TrimSpace(patch.Content); content != "" {
				items[idx].Content = content
			}
			if status, ok := validStatus(patch.Status); ok {
				items[idx].Status = status
			}
			continue
		}

		item := validateItem(patch)
		indexByID[item.ID] = len(items)
		items = append(items, item)
	}
	return items
}

func result(items []Item) Result {
	items = cloneItems(items)
	return Result{
		Todos:   items,
		Summary: summarizeItems(items),
	}
}

func toolError(evidence string, message string) Result {
	return Result{
		Todos:    []Item{},
		Summary:  Summary{},
		Evidence: evidence,
		Error:    message,
	}
}

func summarizeItems(items []Item) Summary {
	var out Summary
	out.Total = len(items)
	for _, item := range items {
		switch item.Status {
		case StatusInProgress:
			out.InProgress++
		case StatusCompleted:
			out.Completed++
		case StatusCancelled:
			out.Cancelled++
		default:
			out.Pending++
		}
	}
	return out
}

func FormatItemsForInjection(items []Item) string {
	if len(items) == 0 {
		return ""
	}

	markers := map[Status]string{
		StatusCompleted:  "[x]",
		StatusInProgress: "[>]",
		StatusPending:    "[ ]",
		StatusCancelled:  "[~]",
	}

	lines := []string{"[Your active task list was preserved across context compression]"}
	for _, item := range validateItems(items) {
		if item.Status != StatusPending && item.Status != StatusInProgress {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s %s. %s (%s)", markers[item.Status], item.ID, item.Content, item.Status))
	}
	if len(lines) == 1 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func cloneItems(items []Item) []Item {
	out := make([]Item, len(items))
	copy(out, items)
	return out
}

func NewTools(cfg Config) []toolkit.Tool {
	return []toolkit.Tool{NewTool(cfg)}
}
