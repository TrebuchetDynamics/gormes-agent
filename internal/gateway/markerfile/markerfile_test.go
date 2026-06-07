package markerfile

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCurrentTimeUsesUTCClockOrNowFallback(t *testing.T) {
	zone := time.FixedZone("offset", 3600)
	want := time.Date(2026, 6, 6, 12, 0, 0, 0, zone).UTC()
	got := CurrentTime(func() time.Time { return time.Date(2026, 6, 6, 12, 0, 0, 0, zone) })
	if !got.Equal(want) || got.Location() != time.UTC {
		t.Fatalf("CurrentTime = %v %v, want %v UTC", got, got.Location(), want)
	}
	if fallback := CurrentTime(nil); fallback.IsZero() {
		t.Fatal("CurrentTime(nil) returned zero time")
	}
}

func TestPositiveDuration(t *testing.T) {
	if got := PositiveDuration(2*time.Second, time.Minute); got != 2*time.Second {
		t.Fatalf("PositiveDuration configured = %v, want 2s", got)
	}
	if got := PositiveDuration(0, time.Minute); got != time.Minute {
		t.Fatalf("PositiveDuration fallback = %v, want 1m", got)
	}
}

func TestClearRemovesExistingAndIgnoresMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marker.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}
	if err := Clear(context.Background(), path, "test marker"); err != nil {
		t.Fatalf("Clear existing: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("stat after clear = %v, want missing", err)
	}
	if err := Clear(context.Background(), path, "test marker"); err != nil {
		t.Fatalf("Clear missing: %v", err)
	}
}

func TestClearHonorsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Clear(ctx, filepath.Join(t.TempDir(), "marker.json"), "test marker"); err == nil {
		t.Fatal("Clear canceled context error = nil")
	}
}
