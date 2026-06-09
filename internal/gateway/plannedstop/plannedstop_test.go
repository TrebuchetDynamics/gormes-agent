package plannedstop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMarkerConsumeRedactsLegacySecretLikeReason(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	store := NewStore(filepath.Join(t.TempDir(), ".gateway-planned-stop.json"))
	store.now = func() time.Time { return now }
	store.pid = func() int { return 4242 }
	store.startTime = func(int) (int64, bool) { return 111111, true }

	if err := os.WriteFile(store.path, []byte(`{
  "kind": "gormes-gateway-planned-stop",
  "target_pid": 4242,
  "target_start_time": 987654,
  "reason": "operator stop api_key=legacy-secret-token",
  "written_at": "2026-05-07T10:00:00Z"
}
`), 0o600); err != nil {
		t.Fatalf("write legacy marker: %v", err)
	}

	result, err := store.ConsumeForSelf(ctx)
	if err != nil {
		t.Fatalf("ConsumeForSelf: %v", err)
	}
	if result.Status != ConsumeMismatched {
		t.Fatalf("status = %q, want mismatched marker evidence", result.Status)
	}
	for _, forbidden := range []string{"legacy-secret-token", "api_key"} {
		if strings.Contains(result.Marker.Reason, forbidden) {
			t.Fatalf("consume result leaked secret-like marker reason %q in %+v", forbidden, result.Marker)
		}
	}
	if result.Marker.Reason != "operator stop [redacted]" {
		t.Fatalf("marker reason = %q, want redacted", result.Marker.Reason)
	}
}

func TestMarkerWriteRedactsSecretLikeReason(t *testing.T) {
	ctx := context.Background()
	store := NewStore(filepath.Join(t.TempDir(), ".gateway-planned-stop.json"))
	store.pid = func() int { return 4242 }

	if err := store.Write(ctx, Marker{
		TargetPID:       4242,
		TargetStartTime: 987654,
		Reason:          "operator stop api_key=plain-secret-token",
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	raw, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	for _, forbidden := range []string{"plain-secret-token", "api_key"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("planned stop marker leaked secret-like reason %q:\n%s", forbidden, raw)
		}
	}
	if !strings.Contains(string(raw), "[redacted]") {
		t.Fatalf("planned stop marker missing redacted reason evidence:\n%s", raw)
	}
}

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

func TestMarkerConsumePropagatesUnreadableMarker(t *testing.T) {
	ctx := context.Background()
	store := NewStore(t.TempDir())

	result, err := store.ConsumeForSelf(ctx)
	if err == nil {
		t.Fatalf("ConsumeForSelf err = nil result=%+v, want read error for directory marker path", result)
	}
	if result.Status == ConsumeMissing {
		t.Fatalf("ConsumeForSelf result = %+v, must not mask unreadable marker as missing", result)
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

	t.Run("future marker beyond ttl", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), ".gateway-planned-stop.json"))
		store.now = func() time.Time { return now }
		store.pid = func() int { return 4242 }
		store.startTime = func(int) (int64, bool) { return 987654, true }
		if err := store.Write(ctx, Marker{
			TargetPID:       4242,
			TargetStartTime: 987654,
			WrittenAt:       now.Add(2 * MarkerTTL).Format(time.RFC3339Nano),
		}); err != nil {
			t.Fatalf("Write: %v", err)
		}

		result, err := store.ConsumeForSelf(ctx)
		if err != nil {
			t.Fatalf("ConsumeForSelf future: %v", err)
		}
		if result.Matched || result.Status != ConsumeStale {
			t.Fatalf("result = %+v, want future marker rejected as stale", result)
		}
		if _, err := os.Stat(store.path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("future marker stat = %v, want removed", err)
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
