package gateway

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestPairingStore_MarkPaired_RoundTripsThroughDisk confirms the minimal
// per-platform paired/unpaired read-model persists to a JSON file and can be
// reloaded into a fresh in-memory store.
func TestPairingStore_MarkPaired_RoundTripsThroughDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pairing.json")
	fixed := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)

	writer, err := LoadPairingStore(path)
	if err != nil {
		t.Fatalf("LoadPairingStore(empty) error = %v", err)
	}
	writer.SetClock(func() time.Time { return fixed })

	if err := writer.MarkPaired("telegram", "123", "alice"); err != nil {
		t.Fatalf("MarkPaired(telegram): %v", err)
	}
	if err := writer.MarkPaired("discord", "abc-def", "bob"); err != nil {
		t.Fatalf("MarkPaired(discord): %v", err)
	}

	reader, err := LoadPairingStore(path)
	if err != nil {
		t.Fatalf("LoadPairingStore(after write): %v", err)
	}

	if !reader.IsPaired("telegram", "123") {
		t.Errorf("IsPaired(telegram, 123) = false after round-trip, want true")
	}
	if !reader.IsPaired("discord", "abc-def") {
		t.Errorf("IsPaired(discord, abc-def) = false after round-trip, want true")
	}
	if reader.IsPaired("slack", "unknown") {
		t.Errorf("IsPaired on unknown platform: got true, want false")
	}
	if reader.IsPaired("telegram", "nobody") {
		t.Errorf("IsPaired on unknown user: got true, want false")
	}

	entries := reader.Snapshot()
	if len(entries) != 2 {
		t.Fatalf("Snapshot length = %d, want 2", len(entries))
	}
	if entries[0].Platform != "discord" || entries[1].Platform != "telegram" {
		t.Errorf("Snapshot platform order = [%s %s], want [discord telegram]", entries[0].Platform, entries[1].Platform)
	}
	for _, e := range entries {
		if !e.PairedAt.Equal(fixed) {
			t.Errorf("entry %s/%s PairedAt = %v, want %v", e.Platform, e.UserID, e.PairedAt, fixed)
		}
	}
}

// TestPairingStore_MarkUnpaired_RemovesUserAndPersists confirms MarkUnpaired
// drops the entry and rewrites the JSON file atomically.
func TestPairingStore_MarkUnpaired_RemovesUserAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pairing.json")
	writer, err := LoadPairingStore(path)
	if err != nil {
		t.Fatalf("LoadPairingStore: %v", err)
	}
	if err := writer.MarkPaired("telegram", "123", "alice"); err != nil {
		t.Fatalf("MarkPaired: %v", err)
	}
	if err := writer.MarkPaired("telegram", "456", "carol"); err != nil {
		t.Fatalf("MarkPaired: %v", err)
	}

	if err := writer.MarkUnpaired("telegram", "123"); err != nil {
		t.Fatalf("MarkUnpaired: %v", err)
	}

	reader, err := LoadPairingStore(path)
	if err != nil {
		t.Fatalf("LoadPairingStore (after unpair): %v", err)
	}
	if reader.IsPaired("telegram", "123") {
		t.Errorf("IsPaired(telegram, 123) = true after MarkUnpaired, want false")
	}
	if !reader.IsPaired("telegram", "456") {
		t.Errorf("IsPaired(telegram, 456) = false after sibling unpair, want true")
	}

	// MarkUnpaired for a user that was never paired is a no-op and must not
	// error — future adapter flows can call it idempotently.
	if err := reader.MarkUnpaired("telegram", "never-paired"); err != nil {
		t.Errorf("MarkUnpaired(unknown user) = %v, want nil (idempotent)", err)
	}
}

// TestPairingStore_Snapshot_IsDeterministic pins the sort order so the future
// `gormes gateway status` read-only surface sees a stable rendering.
func TestPairingStore_Snapshot_IsDeterministic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pairing.json")
	store, err := LoadPairingStore(path)
	if err != nil {
		t.Fatalf("LoadPairingStore: %v", err)
	}

	// Interleave inserts from multiple platforms in non-sorted order.
	inserts := []struct {
		platform, userID, name string
	}{
		{"telegram", "9", "ninth"},
		{"discord", "1", "first"},
		{"telegram", "1", "second"},
		{"slack", "b", "beta"},
		{"discord", "2", "extra"},
		{"slack", "a", "alpha"},
	}
	for _, in := range inserts {
		if err := store.MarkPaired(in.platform, in.userID, in.name); err != nil {
			t.Fatalf("MarkPaired(%s,%s): %v", in.platform, in.userID, err)
		}
	}

	entries := store.Snapshot()
	got := make([]string, len(entries))
	for i, e := range entries {
		got[i] = e.Platform + "/" + e.UserID
	}
	want := []string{
		"discord/1",
		"discord/2",
		"slack/a",
		"slack/b",
		"telegram/1",
		"telegram/9",
	}
	if !equalStringSlice(got, want) {
		t.Errorf("Snapshot ordering = %v, want %v", got, want)
	}
	// Confirm the slice is actually sorted; defensive assertion in case a
	// future refactor accidentally returns insertion order.
	if !sort.SliceIsSorted(entries, func(i, j int) bool {
		if entries[i].Platform == entries[j].Platform {
			return entries[i].UserID < entries[j].UserID
		}
		return entries[i].Platform < entries[j].Platform
	}) {
		t.Errorf("Snapshot not sorted by (platform, user_id)")
	}
}

// TestPairingStore_AtomicWrite_NoStaleTempFiles confirms the write path uses a
// temp-file + rename strategy and leaves no tmp detritus beside pairing.json.
func TestPairingStore_AtomicWrite_NoStaleTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pairing.json")
	store, err := LoadPairingStore(path)
	if err != nil {
		t.Fatalf("LoadPairingStore: %v", err)
	}

	for i := 0; i < 10; i++ {
		if err := store.MarkPaired("telegram", userID(i), ""); err != nil {
			t.Fatalf("MarkPaired: %v", err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var stray []string
	for _, ent := range entries {
		if ent.Name() == "pairing.json" {
			continue
		}
		stray = append(stray, ent.Name())
	}
	if len(stray) > 0 {
		t.Errorf("stray files alongside pairing.json: %v", stray)
	}
}

// TestPairingStore_FilePermissions confirms the persisted file is 0600
// (owner-only). Windows file mode semantics differ — skip there.
func TestPairingStore_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod semantics differ on windows")
	}

	path := filepath.Join(t.TempDir(), "pairing.json")
	store, err := LoadPairingStore(path)
	if err != nil {
		t.Fatalf("LoadPairingStore: %v", err)
	}
	if err := store.MarkPaired("telegram", "1", "alice"); err != nil {
		t.Fatalf("MarkPaired: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	mode := info.Mode().Perm()
	if mode != 0o600 {
		t.Errorf("pairing.json mode = %o, want 0600 (owner-only; pairing state is security-sensitive)", mode)
	}
}

// TestPairingStore_EmptyFileSurface confirms that loading a store with no
// paired users produces an empty snapshot and that a fresh `LoadPairingStore`
// against a missing file does NOT error — this is the cold-start shape the
// gateway needs before the first pairing write lands.
func TestPairingStore_EmptyFileSurface(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pairing.json")
	store, err := LoadPairingStore(path)
	if err != nil {
		t.Fatalf("LoadPairingStore on missing file = %v, want nil", err)
	}
	if snap := store.Snapshot(); len(snap) != 0 {
		t.Errorf("Snapshot on empty store = %v, want []", snap)
	}
	if store.IsPaired("telegram", "x") {
		t.Errorf("IsPaired on empty store: got true, want false")
	}
}

// TestPairingStore_MarkPaired_SameUserTwice_IsIdempotent confirms re-marking
// the same user updates metadata without duplicating the entry.
func TestPairingStore_MarkPaired_SameUserTwice_IsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pairing.json")
	store, err := LoadPairingStore(path)
	if err != nil {
		t.Fatalf("LoadPairingStore: %v", err)
	}
	if err := store.MarkPaired("telegram", "1", "alice"); err != nil {
		t.Fatalf("MarkPaired#1: %v", err)
	}
	if err := store.MarkPaired("telegram", "1", "alice-renamed"); err != nil {
		t.Fatalf("MarkPaired#2: %v", err)
	}

	entries := store.Snapshot()
	if len(entries) != 1 {
		t.Fatalf("Snapshot length after repaired = %d, want 1", len(entries))
	}
	if entries[0].UserName != "alice-renamed" {
		t.Errorf("UserName after re-mark = %q, want %q", entries[0].UserName, "alice-renamed")
	}
}

// TestPairingStore_MalformedFile_SurfaceError ensures a corrupt pairing.json
// bubbles the JSON error up so operators can see it — silent truncation would
// strand paired users.
func TestPairingStore_MalformedFile_SurfaceError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pairing.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o600); err != nil {
		t.Fatalf("seed corrupt file: %v", err)
	}

	if _, err := LoadPairingStore(path); err == nil {
		t.Fatalf("LoadPairingStore on corrupt file = nil, want non-nil error")
	} else if !strings.Contains(err.Error(), "pairing.json") {
		t.Errorf("error message = %q, want to mention pairing.json for operator clarity", err.Error())
	}
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func userID(i int) string {
	return "u-" + string(rune('a'+i))
}
