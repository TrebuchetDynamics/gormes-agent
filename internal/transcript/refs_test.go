package transcript

import "testing"

func TestContextReferenceStore_StableIDExcludesMessagePosition(t *testing.T) {
	store := NewContextReferenceStore()
	record := ContextReferenceRecord{
		Raw:       "@file:src/main.go:4-8",
		Kind:      "file",
		Target:    "src/main.go",
		LineStart: 4,
		LineEnd:   8,
	}

	first := store.Put(record)
	record.Start = 80
	record.End = 105
	second := store.Put(record)

	if first.ID != second.ID {
		t.Fatalf("stable IDs differ: %s vs %s", first.ID, second.ID)
	}
	if got := store.Snapshot(); len(got) != 1 {
		t.Fatalf("snapshot len = %d, want 1", len(got))
	}
}

func TestContextReferenceStore_SnapshotIsDefensive(t *testing.T) {
	store := NewContextReferenceStore()
	handle := store.Put(ContextReferenceRecord{Kind: "url", Raw: "@url:https://example.com", Target: "https://example.com"})

	snapshot := store.Snapshot()
	snapshot[0].ID = "mutated"
	snapshot[0].Record.Target = "mutated"

	again := store.Snapshot()
	if again[0].ID != handle.ID {
		t.Fatalf("store ID was mutated through snapshot: %q", again[0].ID)
	}
	if again[0].Record.Target != "https://example.com" {
		t.Fatalf("store target was mutated through snapshot: %q", again[0].Record.Target)
	}
}
