package event

import "testing"

func TestCloneCopiesStructuredEvent(t *testing.T) {
	original := map[string]any{"method": "event", "id": 1}
	clone := Clone(original)

	clone["method"] = "changed"
	clone["new"] = true

	if got := original["method"]; got != "event" {
		t.Fatalf("original method = %v, want event", got)
	}
	if _, ok := original["new"]; ok {
		t.Fatalf("original unexpectedly received cloned key")
	}
}

func TestCloneNilEventReturnsEmptyMap(t *testing.T) {
	clone := Clone(nil)
	if clone == nil {
		t.Fatalf("Clone(nil) = nil, want empty map")
	}
	if len(clone) != 0 {
		t.Fatalf("Clone(nil) len = %d, want 0", len(clone))
	}
}
