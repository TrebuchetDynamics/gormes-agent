package goncho

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
)

type mockMemoryStore struct {
	mu      sync.Mutex
	entries map[string]MemoryToolEntry
	storeFn func(ctx context.Context, entry MemoryToolEntry) error
}

func newMockStore() *mockMemoryStore {
	return &mockMemoryStore{entries: make(map[string]MemoryToolEntry)}
}

func (m *mockMemoryStore) Store(ctx context.Context, entry MemoryToolEntry) error {
	if m.storeFn != nil {
		return m.storeFn(ctx, entry)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[entry.ID] = entry
	return nil
}

func (m *mockMemoryStore) Retrieve(ctx context.Context, query string, limit int) ([]MemoryToolEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []MemoryToolEntry
	for _, e := range m.entries {
		if query == "" || containsTag(e.Tags, query) || containsContent(e.Content, query) {
			results = append(results, e)
			if len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

func (m *mockMemoryStore) Update(ctx context.Context, id string, content string) error {
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

func (m *mockMemoryStore) Forget(ctx context.Context, id string) error {
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

func containsContent(content, query string) bool {
	return len(content) > 0 && len(query) > 0
}

func TestStoreMemory(t *testing.T) {
	store := newMockStore()
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
	if out["id"] == nil || out["id"] == "" {
		t.Fatal("store_memory did not return an id")
	}
}

func TestStoreMemory_MissingContent(t *testing.T) {
	store := newMockStore()
	tool := &storeMemoryTool{newMemoryToolBase(store)}

	args := json.RawMessage(`{"tags":["test"]}`)
	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("store_memory should fail with missing content")
	}
}

func TestRetrieveMemory(t *testing.T) {
	store := newMockStore()
	store.Store(context.Background(), MemoryToolEntry{
		ID: "mem_1", Content: "hello world", Tags: []string{"greeting"}, Importance: 0.9,
	})

	tool := &retrieveMemoryTool{newMemoryToolBase(store)}
	args := json.RawMessage(`{"query":"greeting","limit":5}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("retrieve_memory failed: %v", err)
	}

	var out map[string]interface{}
	json.Unmarshal(result, &out)
	results, ok := out["results"].([]interface{})
	if !ok {
		t.Fatal("retrieve_memory did not return results array")
	}
	if len(results) == 0 {
		t.Fatal("retrieve_memory returned empty results")
	}
}

func TestRetrieveMemory_EmptyQuery(t *testing.T) {
	tool := &retrieveMemoryTool{newMemoryToolBase(newMockStore())}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":""}`))
	if err == nil {
		t.Fatal("retrieve_memory should fail with empty query")
	}
}

func TestUpdateMemory(t *testing.T) {
	store := newMockStore()
	store.Store(context.Background(), MemoryToolEntry{ID: "mem_1", Content: "old content"})

	tool := &updateMemoryTool{newMemoryToolBase(store)}
	args := json.RawMessage(`{"id":"mem_1","content":"new content"}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("update_memory failed: %v", err)
	}

	var out map[string]interface{}
	json.Unmarshal(result, &out)
	if out["success"] != true {
		t.Fatal("update_memory did not succeed")
	}
}

func TestUpdateMemory_MissingFields(t *testing.T) {
	tool := &updateMemoryTool{newMemoryToolBase(newMockStore())}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"id":"x"}`))
	if err == nil {
		t.Fatal("update_memory should fail with missing content")
	}
}

func TestSummarizeMemories(t *testing.T) {
	store := newMockStore()
	store.Store(context.Background(), MemoryToolEntry{ID: "m1", Content: "a", Tags: []string{"proj"}})
	store.Store(context.Background(), MemoryToolEntry{ID: "m2", Content: "b", Tags: []string{"proj"}})

	tool := &summarizeMemoryTool{newMemoryToolBase(store)}
	args := json.RawMessage(`{"filter":"proj","max_items":5}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("summarize_memories failed: %v", err)
	}

	var out map[string]interface{}
	json.Unmarshal(result, &out)
	summarized, ok := out["summarized"].(float64)
	if !ok || summarized == 0 {
		t.Fatal("summarize_memories did not return summarized count")
	}
}

func TestForgetMemory(t *testing.T) {
	store := newMockStore()
	store.Store(context.Background(), MemoryToolEntry{ID: "mem_1", Content: "to forget"})

	tool := &forgetMemoryTool{newMemoryToolBase(store)}
	args := json.RawMessage(`{"id":"mem_1"}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("forget_memory failed: %v", err)
	}

	var out map[string]interface{}
	json.Unmarshal(result, &out)
	if out["success"] != true {
		t.Fatal("forget_memory did not succeed")
	}
}

func TestForgetMemory_MissingID(t *testing.T) {
	tool := &forgetMemoryTool{newMemoryToolBase(newMockStore())}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"id":""}`))
	if err == nil {
		t.Fatal("forget_memory should fail with missing id")
	}
}

func TestMemoryToolNames(t *testing.T) {
	store := newMockStore()
	tools := []struct {
		name string
		t    interface{ Name() string }
	}{
		{"store_memory", &storeMemoryTool{newMemoryToolBase(store)}},
		{"retrieve_memory", &retrieveMemoryTool{newMemoryToolBase(store)}},
		{"update_memory", &updateMemoryTool{newMemoryToolBase(store)}},
		{"summarize_memories", &summarizeMemoryTool{newMemoryToolBase(store)}},
		{"forget_memory", &forgetMemoryTool{newMemoryToolBase(store)}},
	}
	for _, tc := range tools {
		if tc.t.Name() != tc.name {
			t.Errorf("tool Name() = %q, want %q", tc.t.Name(), tc.name)
		}
	}
}
