package codingagents

import (
	"context"
	"testing"
)

func TestDetectAvailability_MissingBinary(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	got := DetectAvailability(ctx, "gormes-fake-binary-xyz")
	if got.Available {
		t.Fatalf("expected Available=false for missing binary, got %#v", got)
	}
	if got.Name != "gormes-fake-binary-xyz" {
		t.Fatalf("Name not echoed back, got %q", got.Name)
	}
	if got.Version != "" {
		t.Fatalf("expected empty Version, got %q", got.Version)
	}
}

func TestDetectAvailability_EmptyName(t *testing.T) {
	t.Parallel()
	got := DetectAvailability(context.Background(), "   ")
	if got.Available {
		t.Fatalf("empty name must report unavailable, got %#v", got)
	}
	if got.Error == "" {
		t.Fatalf("expected Error to describe empty name")
	}
}

func TestDetectAll_ReturnsKnownBackends(t *testing.T) {
	t.Parallel()
	results := DetectAll(context.Background())
	for _, name := range KnownBackends {
		if _, ok := results[name]; !ok {
			t.Fatalf("DetectAll missing entry for %q", name)
		}
	}
	if len(results) != len(KnownBackends) {
		t.Fatalf("DetectAll returned %d entries, want %d", len(results), len(KnownBackends))
	}
}
