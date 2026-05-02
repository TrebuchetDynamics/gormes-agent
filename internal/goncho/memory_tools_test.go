package goncho

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

type mockMemoryToolStore struct {
	mu      sync.Mutex
	entries map[string]MemoryToolEntry
}

func newMockToolStore() *mockMemoryToolStore {
	return &mockMemoryToolStore{entries: make(map[string]MemoryToolEntry)}
}

func (m *mockMemoryToolStore) Store(ctx context.Context, entry MemoryToolEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[entry.ID] = entry
	return nil
}

func (m *mockMemoryToolStore) Retrieve(ctx context.Context, query string, limit int) ([]MemoryToolEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []MemoryToolEntry
	for _, e := range m.entries {
		if query == "" || containsTag(e.Tags, query) || containsMemoryContent(e.Content, query) {
			results = append(results, e)
			if len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

func (m *mockMemoryToolStore) Update(ctx context.Context, id string, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.entries[id]
	if !ok {
		return nil
	}
	entry.Content = content
	m.entries[id] = entry
	return nil
}

func (m *mockMemoryToolStore) Forget(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, id)
	return nil
}

func containsTag(tags []string, query string) bool {
	for _, t := range tags {
		if t == query {
			return true
		}
	}
	return false
}

func containsMemoryContent(content string, query string) bool {
	content = strings.ToLower(content)
	query = strings.ToLower(query)
	return strings.Contains(content, query) || strings.Contains(query, content)
}

func TestStoreMemory(t *testing.T) {
	store := newMockToolStore()
	tool := &storeMemoryTool{newMemoryToolBase(store)}
	args := json.RawMessage(`{"content":"test memory","tags":["test"],"importance":0.8}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("store_memory failed: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("invalid result json: %v", err)
	}
	if out["success"] != true {
		t.Fatal("store_memory did not succeed")
	}
}

func TestStoreMemory_MissingContent(t *testing.T) {
	store := newMockToolStore()
	tool := &storeMemoryTool{newMemoryToolBase(store)}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"tags":["test"]}`))
	if err == nil {
		t.Fatal("store_memory should fail with missing content")
	}
}

func TestRetrieveMemory(t *testing.T) {
	store := newMockToolStore()
	store.Store(context.Background(), MemoryToolEntry{ID: "mem_1", Content: "hello world", Tags: []string{"greeting"}, Importance: 0.9})
	tool := &retrieveMemoryTool{newMemoryToolBase(store)}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"greeting","limit":5}`))
	if err != nil {
		t.Fatalf("retrieve_memory failed: %v", err)
	}
	var out map[string]interface{}
	json.Unmarshal(result, &out)
	results, ok := out["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Fatal("retrieve_memory did not return results")
	}
}

func TestUpdateMemory(t *testing.T) {
	store := newMockToolStore()
	store.Store(context.Background(), MemoryToolEntry{ID: "mem_1", Content: "old content"})
	tool := &updateMemoryTool{newMemoryToolBase(store)}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"id":"mem_1","content":"new content"}`))
	if err != nil {
		t.Fatalf("update_memory failed: %v", err)
	}
	var out map[string]interface{}
	json.Unmarshal(result, &out)
	if out["success"] != true {
		t.Fatal("update_memory did not succeed")
	}
}

func TestSummarizeMemories(t *testing.T) {
	store := newMockToolStore()
	store.Store(context.Background(), MemoryToolEntry{ID: "m1", Content: "a", Tags: []string{"proj"}})
	store.Store(context.Background(), MemoryToolEntry{ID: "m2", Content: "b", Tags: []string{"proj"}})
	tool := &summarizeMemoryTool{newMemoryToolBase(store)}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"filter":"proj","max_items":5}`))
	if err != nil {
		t.Fatalf("summarize_memories failed: %v", err)
	}
	var out map[string]interface{}
	json.Unmarshal(result, &out)
	if out["summarized"].(float64) == 0 {
		t.Fatal("summarize_memories did not return summarized count")
	}
}

func TestForgetMemory(t *testing.T) {
	store := newMockToolStore()
	store.Store(context.Background(), MemoryToolEntry{ID: "mem_1", Content: "to forget"})
	tool := &forgetMemoryTool{newMemoryToolBase(store)}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"id":"mem_1"}`))
	if err != nil {
		t.Fatalf("forget_memory failed: %v", err)
	}
	var out map[string]interface{}
	json.Unmarshal(result, &out)
	if out["success"] != true {
		t.Fatal("forget_memory did not succeed")
	}
}

func TestMemoryToolNames(t *testing.T) {
	store := newMockToolStore()
	tests := []struct {
		want string
		tool interface{ Name() string }
	}{
		{"store_memory", &storeMemoryTool{newMemoryToolBase(store)}},
		{"retrieve_memory", &retrieveMemoryTool{newMemoryToolBase(store)}},
		{"update_memory", &updateMemoryTool{newMemoryToolBase(store)}},
		{"summarize_memories", &summarizeMemoryTool{newMemoryToolBase(store)}},
		{"forget_memory", &forgetMemoryTool{newMemoryToolBase(store)}},
	}
	for _, tc := range tests {
		if tc.tool.Name() != tc.want {
			t.Errorf("tool Name() = %q, want %q", tc.tool.Name(), tc.want)
		}
	}
}
