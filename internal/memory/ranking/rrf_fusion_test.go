package ranking

import (
	"reflect"
	"testing"
)

func TestRRFFusion_ProducesRankedResults(t *testing.T) {
	fts := []int64{1, 2, 3}
	sem := []int64{4, 5, 6}
	result := FuseIDs(fts, sem, 60, 1.0, 1.0)
	if len(result) != 6 {
		t.Fatalf("expected 6 results, got %d", len(result))
	}
	// ID 1 should be first (rank 1 in FTS, no semantic)
	if result[0] != 1 {
		t.Errorf("expected first result = 1, got %d", result[0])
	}
}

func TestRRFFusion_DeduplicatesAcrossSources(t *testing.T) {
	fts := []int64{1, 2, 3}
	sem := []int64{2, 3, 4}
	result := FuseIDs(fts, sem, 60, 1.0, 1.0)
	// ID 2 appears in both lists — should rank higher than 1
	if result[0] != 2 {
		t.Errorf("expected ID 2 first (appears in both), got %d", result[0])
	}
	// No duplicates in result
	seen := make(map[int64]bool)
	for _, id := range result {
		if seen[id] {
			t.Errorf("duplicate ID %d in result", id)
		}
		seen[id] = true
	}
}

func TestRRFFusion_EmptyInputs(t *testing.T) {
	if got := FuseIDs(nil, nil, 60, 1.0, 1.0); len(got) != 0 {
		t.Errorf("expected empty result for nil inputs, got %v", got)
	}
	if got := FuseIDs([]int64{1}, nil, 60, 1.0, 1.0); len(got) != 1 || got[0] != 1 {
		t.Errorf("expected [1] for fts-only, got %v", got)
	}
	if got := FuseIDs(nil, []int64{2}, 60, 1.0, 1.0); len(got) != 1 || got[0] != 2 {
		t.Errorf("expected [2] for sem-only, got %v", got)
	}
}

func TestRRFFusion_CustomWeights(t *testing.T) {
	fts := []int64{1}
	sem := []int64{2}
	result := FuseIDs(fts, sem, 60, 2.0, 1.0)
	// FTS weight 2.0 should make ID 1 rank higher than ID 2
	if result[0] != 1 {
		t.Errorf("expected ID 1 first (higher FTS weight), got %d", result[0])
	}
}

func TestRRFFusion_OrderPreservation(t *testing.T) {
	// When only one source has results, order should be preserved
	fts := []int64{10, 20, 30}
	result := FuseIDs(fts, nil, 60, 1.0, 1.0)
	expected := []int64{10, 20, 30}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("order not preserved: got %v, want %v", result, expected)
	}
}
