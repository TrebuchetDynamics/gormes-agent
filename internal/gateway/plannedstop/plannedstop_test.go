package plannedstop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMarkerConsumesMatchingPIDAndStartTime(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	store := NewStore(filepath.Join(t.TempDir(), ".gateway-planned-stop.json"))
	store.now = func() time.Time { return now }
	store.pid = func() int { return 4242 }
	store.startTime = func(int) (int64, bool) { return 987654, true }

	if err := store.Write(ctx, Marker{
		TargetPID:       4242,
		TargetStartTime: 987654,
		Generation:      9,
		Reason:          "gateway stop",
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	result, err := store.ConsumeForSelf(ctx)
	if err != nil {
		t.Fatalf("ConsumeForSelf: %v", err)
	}
	if !result.Matched || result.Status != ConsumeMatched {
		t.Fatalf("result = %+v, want matched planned stop", result)
	}
	if result.Marker.TargetPID != 4242 || result.Marker.Generation != 9 {
		t.Fatalf("result marker = %+v, want target/generation evidence", result.Marker)
	}
	if _, err := os.Stat(store.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("marker after consume stat = %v, want removed", err)
	}
}

func TestMarkerStaleOrMismatchedNeverMatches(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)

	t.Run("stale marker", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), ".gateway-planned-stop.json"))
		store.now = func() time.Time { return now }
		store.pid = func() int { return 4242 }
		store.startTime = func(int) (int64, bool) { return 987654, true }
		if err := store.Write(ctx, Marker{
			TargetPID:       4242,
			TargetStartTime: 987654,
			WrittenAt:       now.Add(-2 * MarkerTTL).Format(time.RFC3339Nano),
		}); err != nil {
			t.Fatalf("Write: %v", err)
		}

		result, err := store.ConsumeForSelf(ctx)
		if err != nil {
			t.Fatalf("ConsumeForSelf stale: %v", err)
		}
		if result.Matched || result.Status != ConsumeStale {
			t.Fatalf("result = %+v, want stale non-match", result)
		}
		if _, err := os.Stat(store.path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stale marker stat = %v, want removed", err)
		}
	})

	t.Run("start time mismatch", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), ".gateway-planned-stop.json"))
		store.now = func() time.Time { return now }
		store.pid = func() int { return 4242 }
		store.startTime = func(int) (int64, bool) { return 111111, true }
		if err := store.Write(ctx, Marker{
			TargetPID:       4242,
			TargetStartTime: 987654,
		}); err != nil {
			t.Fatalf("Write: %v", err)
		}

		result, err := store.ConsumeForSelf(ctx)
		if err != nil {
			t.Fatalf("ConsumeForSelf mismatch: %v", err)
		}
		if result.Matched || result.Status != ConsumeMismatched {
			t.Fatalf("result = %+v, want mismatch non-match", result)
		}
		if _, err := os.Stat(store.path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("mismatched marker stat = %v, want removed", err)
		}
	})
}
