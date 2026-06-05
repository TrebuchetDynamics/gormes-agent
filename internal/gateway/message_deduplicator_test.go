package gateway

import "testing"

func TestMessageDeduplicatorCompatibilityWrapper(t *testing.T) {
	dedup := NewMessageDeduplicator(1)
	if result := dedup.Track("msg-1"); result.Duplicate || result.Evidence != "" {
		t.Fatalf("first Track = %+v, want new message", result)
	}
	if result := dedup.Track("msg-1"); !result.Duplicate || result.Evidence != MessageDeduplicatorEvidenceDuplicate {
		t.Fatalf("second Track = %+v, want duplicate evidence", result)
	}
}
