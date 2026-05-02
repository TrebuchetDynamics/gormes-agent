package goncho

import (
	"context"
	"sort"
	"time"
)

type CrossSessionMemory struct {
	store MemoryToolStore
}

func NewCrossSessionMemory(store MemoryToolStore) *CrossSessionMemory {
	return &CrossSessionMemory{store: store}
}

func (csm *CrossSessionMemory) LoadRelevant(ctx context.Context, query string, limit int) ([]MemoryToolEntry, error) {
	if limit <= 0 {
		limit = 5
	}
	return csm.store.Retrieve(ctx, query, limit)
}

func (csm *CrossSessionMemory) DetectContradictions(ctx context.Context, newEntry MemoryToolEntry) ([]MemoryToolEntry, error) {
	existing, err := csm.store.Retrieve(ctx, newEntry.Content, 5)
	if err != nil || len(existing) == 0 {
		return nil, err
	}
	var conflicts []MemoryToolEntry
	for _, e := range existing {
		if e.ID != newEntry.ID && isContradictory(e.Content, newEntry.Content) {
			conflicts = append(conflicts, e)
		}
	}
	return conflicts, nil
}

func isContradictory(a, b string) bool {
	return len(a) > 5 && len(b) > 5 && a != b
}

func RecentEntries(entries []MemoryToolEntry, now time.Time, maxAge time.Duration) []MemoryToolEntry {
	var recent []MemoryToolEntry
	for _, e := range entries {
		if now.Sub(e.CreatedAt) <= maxAge {
			recent = append(recent, e)
		}
	}
	sort.Slice(recent, func(i, j int) bool {
		return recent[i].CreatedAt.After(recent[j].CreatedAt)
	})
	return recent
}
