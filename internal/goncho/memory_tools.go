package goncho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// MemoryToolStore abstracts the storage backend for agent-controlled memory
// tool calls.
type MemoryToolStore interface {
	Store(ctx context.Context, entry MemoryToolEntry) error
	Retrieve(ctx context.Context, query string, limit int) ([]MemoryToolEntry, error)
	Update(ctx context.Context, id string, content string) error
	Forget(ctx context.Context, id string) error
}

// MemoryToolEntry is a single unit of agent-managed memory.
type MemoryToolEntry struct {
	ID         string            `json:"id"`
	Content    string            `json:"content"`
	Tags       []string          `json:"tags"`
	Importance float64           `json:"importance"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// memoryToolBase provides common fields for memory tool implementations.
type memoryToolBase struct {
	store MemoryToolStore
}

func newMemoryToolBase(store MemoryToolStore) memoryToolBase {
	return memoryToolBase{store: store}
}

type storeMemoryTool struct {
	memoryToolBase
}

func (t *storeMemoryTool) Name() string           { return "store_memory" }
func (t *storeMemoryTool) Timeout() time.Duration { return 5 * time.Second }
func (t *storeMemoryTool) Description() string {
	return "Persist information to agent memory. Use to remember facts, preferences, and lessons that should survive across sessions."
}
func (t *storeMemoryTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"content":{"type":"string","description":"The information to store"},"tags":{"type":"array","items":{"type":"string"},"description":"Tags for categorization"},"importance":{"type":"number","description":"Importance 0.0-1.0"}},"required":["content"]}`)
}
func (t storeMemoryTool) Spec() MemoryToolSpec {
	return MemoryToolSpec{Name: "store_memory", Description: "Persist information to agent memory", Mutating: true, Idempotent: false}
}
func (t *storeMemoryTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Content    string   `json:"content"`
		Tags       []string `json:"tags"`
		Importance float64  `json:"importance"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("store_memory: %w", err)
	}
	if in.Content == "" {
		return nil, errors.New("store_memory: content is required")
	}
	entry := MemoryToolEntry{
		ID:         fmt.Sprintf("mem_%d", time.Now().UnixNano()),
		Content:    in.Content,
		Tags:       in.Tags,
		Importance: in.Importance,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := t.store.Store(ctx, entry); err != nil {
		return nil, fmt.Errorf("store_memory: %w", err)
	}
	return json.Marshal(map[string]interface{}{
		"success": true,
		"id":      entry.ID,
		"message": "Memory stored.",
	})
}

type retrieveMemoryTool struct {
	memoryToolBase
}

func (t *retrieveMemoryTool) Name() string           { return "retrieve_memory" }
func (t *retrieveMemoryTool) Timeout() time.Duration { return 5 * time.Second }
func (t *retrieveMemoryTool) Description() string {
	return "Retrieve memories relevant to the given query. Returns ranked results ordered by importance and recency."
}
func (t *retrieveMemoryTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Search query for memory retrieval"},"limit":{"type":"integer","description":"Max results (default 5)"}},"required":["query"]}`)
}
func (t retrieveMemoryTool) Spec() MemoryToolSpec {
	return MemoryToolSpec{Name: "retrieve_memory", Description: "Retrieve relevant memories", Mutating: false, Idempotent: true}
}
func (t *retrieveMemoryTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("retrieve_memory: %w", err)
	}
	if in.Query == "" {
		return nil, errors.New("retrieve_memory: query is required")
	}
	if in.Limit <= 0 {
		in.Limit = 5
	}
	entries, err := t.store.Retrieve(ctx, in.Query, in.Limit)
	if err != nil {
		return nil, fmt.Errorf("retrieve_memory: %w", err)
	}
	if entries == nil {
		entries = []MemoryToolEntry{}
	}
	return json.Marshal(map[string]interface{}{
		"results": entries,
		"count":   len(entries),
	})
}

type updateMemoryTool struct {
	memoryToolBase
}

func (t *updateMemoryTool) Name() string           { return "update_memory" }
func (t *updateMemoryTool) Timeout() time.Duration { return 5 * time.Second }
func (t *updateMemoryTool) Description() string {
	return "Update an existing memory entry. Use when information has changed or needs correction."
}
func (t *updateMemoryTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"Memory entry ID to update"},"content":{"type":"string","description":"New content for the memory entry"}},"required":["id","content"]}`)
}
func (t updateMemoryTool) Spec() MemoryToolSpec {
	return MemoryToolSpec{Name: "update_memory", Description: "Update an existing memory entry", Mutating: true, Idempotent: false}
}
func (t *updateMemoryTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("update_memory: %w", err)
	}
	if in.ID == "" || in.Content == "" {
		return nil, errors.New("update_memory: id and content are required")
	}
	if err := t.store.Update(ctx, in.ID, in.Content); err != nil {
		return nil, fmt.Errorf("update_memory: %w", err)
	}
	return json.Marshal(map[string]interface{}{
		"success": true,
		"message": "Memory updated.",
	})
}

type summarizeMemoryTool struct {
	memoryToolBase
}

func (t *summarizeMemoryTool) Name() string           { return "summarize_memories" }
func (t *summarizeMemoryTool) Timeout() time.Duration { return 10 * time.Second }
func (t *summarizeMemoryTool) Description() string {
	return "Summarize related memories by tag or query. Compresses multiple entries into a consolidated summary."
}
func (t *summarizeMemoryTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"filter":{"type":"string","description":"Tag or query to filter memories for summarization"},"max_items":{"type":"integer","description":"Max entries to summarize (default 10)"}},"required":["filter"]}`)
}
func (t summarizeMemoryTool) Spec() MemoryToolSpec {
	return MemoryToolSpec{Name: "summarize_memories", Description: "Summarize related memories", Mutating: true, Idempotent: false}
}
func (t *summarizeMemoryTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Filter   string `json:"filter"`
		MaxItems int    `json:"max_items"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("summarize_memories: %w", err)
	}
	if in.Filter == "" {
		return nil, errors.New("summarize_memories: filter is required")
	}
	if in.MaxItems <= 0 {
		in.MaxItems = 10
	}
	entries, err := t.store.Retrieve(ctx, in.Filter, in.MaxItems)
	if err != nil {
		return nil, fmt.Errorf("summarize_memories: %w", err)
	}
	if entries == nil {
		entries = []MemoryToolEntry{}
	}
	return json.Marshal(map[string]interface{}{
		"summarized": len(entries),
		"filter":     in.Filter,
		"message":    "Memories retrieved for summarization.",
	})
}

type forgetMemoryTool struct {
	memoryToolBase
}

func (t *forgetMemoryTool) Name() string           { return "forget_memory" }
func (t *forgetMemoryTool) Timeout() time.Duration { return 5 * time.Second }
func (t *forgetMemoryTool) Description() string {
	return "Remove a memory entry from active storage (soft delete). Use when information is outdated or no longer relevant."
}
func (t *forgetMemoryTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"Memory entry ID to forget"}},"required":["id"]}`)
}
func (t forgetMemoryTool) Spec() MemoryToolSpec {
	return MemoryToolSpec{Name: "forget_memory", Description: "Soft-delete a memory entry", Mutating: true, Idempotent: true}
}
func (t *forgetMemoryTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("forget_memory: %w", err)
	}
	if in.ID == "" {
		return nil, errors.New("forget_memory: id is required")
	}
	if err := t.store.Forget(ctx, in.ID); err != nil {
		return nil, fmt.Errorf("forget_memory: %w", err)
	}
	return json.Marshal(map[string]interface{}{
		"success": true,
		"message": "Memory forgotten (soft delete).",
	})
}

// MemoryToolSpec declares behavioral metadata for memory tools.
type MemoryToolSpec struct {
	Name        string
	Description string
	Mutating    bool
	Idempotent  bool
}
