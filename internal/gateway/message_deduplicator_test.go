package gateway

import "testing"

func TestMessageDeduplicator_MaxSizeEvictsOldest(t *testing.T) {
	dedup := NewMessageDeduplicator(3)

	for _, id := range []string{"msg-1", "msg-2", "msg-3"} {
		if result := dedup.Track(id); result.Duplicate || result.Evidence != "" {
			t.Fatalf("Track(%q) = %+v, want newly tracked without evidence", id, result)
		}
	}

	result := dedup.Track("msg-4")
	if result.Duplicate {
		t.Fatalf("Track(msg-4) duplicate = true, want false")
	}
	if result.Evidence != MessageDeduplicatorEvidenceEvicted || result.EvictedID != "msg-1" {
		t.Fatalf("Track(msg-4) = %+v, want evicted evidence for msg-1", result)
	}

	for _, id := range []string{"msg-3", "msg-4"} {
		result := dedup.Track(id)
		if !result.Duplicate || result.Evidence != MessageDeduplicatorEvidenceDuplicate {
			t.Fatalf("Track(%q) = %+v, want duplicate evidence", id, result)
		}
	}

	result = dedup.Track("msg-1")
	if result.Duplicate {
		t.Fatalf("Track(msg-1) duplicate = true, want oldest ID to be evicted")
	}
}

func TestMessageDeduplicator_DuplicateReturnsSeen(t *testing.T) {
	dedup := NewMessageDeduplicator(2)

	if result := dedup.Track("msg-1"); result.Duplicate || result.Evidence != "" {
		t.Fatalf("first Track(msg-1) = %+v, want newly tracked without evidence", result)
	}

	result := dedup.Track("msg-1")
	if !result.Duplicate {
		t.Fatalf("second Track(msg-1) duplicate = false, want true")
	}
	if result.Evidence != MessageDeduplicatorEvidenceDuplicate || result.EvictedID != "" {
		t.Fatalf("second Track(msg-1) = %+v, want duplicate evidence without eviction", result)
	}
}

func TestMessageDeduplicator_ZeroMaxSizeDisabled(t *testing.T) {
	dedup := NewMessageDeduplicator(0)

	for i := 0; i < 2; i++ {
		result := dedup.Track("msg-1")
		if result.Duplicate {
			t.Fatalf("Track(msg-1) attempt %d duplicate = true, want disabled deduplicator to allow", i+1)
		}
		if result.Evidence != MessageDeduplicatorEvidenceDisabled {
			t.Fatalf("Track(msg-1) attempt %d evidence = %q, want %q", i+1, result.Evidence, MessageDeduplicatorEvidenceDisabled)
		}
	}

	if dedup.seen != nil || dedup.order != nil {
		t.Fatalf("disabled deduplicator allocated seen queue: seen=%v order=%v", dedup.seen, dedup.order)
	}
}
