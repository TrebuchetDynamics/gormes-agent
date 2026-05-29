package cli

import (
	"errors"
	"testing"
	"time"
)

func TestDebugPasteSweep_ExpiredEntriesDeletedWithFakeClock(t *testing.T) {
	// Fixed time anchor: 2026-04-29T12:00:00Z
	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)

	store := &InMemoryPasteStore{}
	deleted := make([]string, 0)
	fakeDeleter := &fakePasteDeleter{
		deleteFn: func(url string) error {
			deleted = append(deleted, url)
			return nil
		},
	}

	// Create entries: one expired 1 hour ago, one expired 2 hours ago.
	expired1 := PasteEntry{
		ID:        "p1",
		URL:       "https://paste.rs/abc123",
		ExpireAt:  now.Add(-1 * time.Hour),
		CreatedAt: now.Add(-7 * time.Hour),
	}
	expired2 := PasteEntry{
		ID:        "p2",
		URL:       "https://paste.rs/def456",
		ExpireAt:  now.Add(-2 * time.Hour),
		CreatedAt: now.Add(-8 * time.Hour),
	}
	// Unexpired entry (expires in 5 hours)
	pending := PasteEntry{
		ID:        "p3",
		URL:       "https://paste.rs/ghi789",
		ExpireAt:  now.Add(5 * time.Hour),
		CreatedAt: now.Add(-1 * time.Hour),
	}
	store.Entries = map[string]PasteEntry{
		"p1": expired1,
		"p2": expired2,
		"p3": pending,
	}

	sweeper := &PasteSweeper{
		Store:   store,
		Deleter: fakeDeleter,
		Now:     func() time.Time { return now },
	}

	result, err := sweeper.SweepExpired()
	if err != nil {
		t.Fatalf("SweepExpired() error = %v", err)
	}

	// Expired entries should be deleted exactly once.
	if len(deleted) != 2 {
		t.Errorf("expected 2 deleted URLs, got %d", len(deleted))
	}
	if result.Deleted != 2 {
		t.Errorf("result.Deleted = %d, want 2", result.Deleted)
	}
	if result.Remaining != 1 {
		t.Errorf("result.Remaining = %d, want 1", result.Remaining)
	}
	// Pending entry should be retained.
	if len(store.Entries) != 1 {
		t.Errorf("store has %d entries, want 1", len(store.Entries))
	}
	if _, ok := store.Entries["p3"]; !ok {
		t.Error("pending entry p3 should be retained")
	}
}

func TestDebugPasteSweep_UnexpiredEntriesRemain(t *testing.T) {
	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)

	store := &InMemoryPasteStore{}
	fakeDeleter := &fakePasteDeleter{
		deleteFn: func(url string) error {
			t.Errorf("unexpected DeletePaste call for %s", url)
			return nil
		},
	}

	// All entries expire in the future.
	entries := []PasteEntry{
		{ID: "f1", URL: "https://paste.rs/f1", ExpireAt: now.Add(1 * time.Hour), CreatedAt: now},
		{ID: "f2", URL: "https://paste.rs/f2", ExpireAt: now.Add(3 * time.Hour), CreatedAt: now},
		{ID: "f3", URL: "https://paste.rs/f3", ExpireAt: now.Add(6 * time.Hour), CreatedAt: now},
	}
	store.Entries = make(map[string]PasteEntry, len(entries))
	for _, e := range entries {
		store.Entries[e.ID] = e
	}

	sweeper := &PasteSweeper{
		Store:   store,
		Deleter: fakeDeleter,
		Now:     func() time.Time { return now },
	}

	result, err := sweeper.SweepExpired()
	if err != nil {
		t.Fatalf("SweepExpired() error = %v", err)
	}

	if result.Deleted != 0 {
		t.Errorf("result.Deleted = %d, want 0", result.Deleted)
	}
	if result.Remaining != 3 {
		t.Errorf("result.Remaining = %d, want 3", result.Remaining)
	}
	if len(store.Entries) != 3 {
		t.Errorf("store should have 3 entries, got %d", len(store.Entries))
	}
}

func TestDebugPasteSweep_DeleteFailureRetainsEntryWithEvidence(t *testing.T) {
	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)

	store := &InMemoryPasteStore{}
	deleteCalled := make([]string, 0)
	fakeDeleter := &fakePasteDeleter{
		deleteFn: func(url string) error {
			deleteCalled = append(deleteCalled, url)
			return errors.New("network error")
		},
	}

	// Expired 30 minutes ago — still within 24h grace period.
	expired := PasteEntry{
		ID:        "fail1",
		URL:       "https://paste.rs/fail1",
		ExpireAt:  now.Add(-30 * time.Minute),
		CreatedAt: now.Add(-7 * time.Hour),
	}
	store.Entries = map[string]PasteEntry{"fail1": expired}

	sweeper := &PasteSweeper{
		Store:   store,
		Deleter: fakeDeleter,
		Now:     func() time.Time { return now },
	}

	result, err := sweeper.SweepExpired()
	if err != nil {
		t.Fatalf("SweepExpired() error = %v", err)
	}

	if len(deleteCalled) != 1 {
		t.Errorf("DeletePaste called %d times, want 1", len(deleteCalled))
	}
	if result.Deleted != 0 {
		t.Errorf("result.Deleted = %d, want 0 (failure retained)", result.Deleted)
	}
	if result.Remaining != 1 {
		t.Errorf("result.Remaining = %d, want 1", result.Remaining)
	}
	if len(result.Errors) != 1 {
		t.Errorf("result.Errors length = %d, want 1", len(result.Errors))
	}
	if result.Errors[0].Evidence != "paste_delete_failed" {
		t.Errorf("result.Errors[0].Evidence = %q, want paste_delete_failed", result.Errors[0].Evidence)
	}
	// Entry should still be in the store.
	if _, ok := store.Entries["fail1"]; !ok {
		t.Error("failed entry should be retained in store")
	}
}

func TestDebugPasteSweep_GracePeriodExpireGivesUp(t *testing.T) {
	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)

	store := &InMemoryPasteStore{}
	deleteCalled := make([]string, 0)
	fakeDeleter := &fakePasteDeleter{
		deleteFn: func(url string) error {
			deleteCalled = append(deleteCalled, url)
			return errors.New("persistent failure")
		},
	}

	// Expired 25 hours ago — past the 24h grace period.
	expired := PasteEntry{
		ID:        "old1",
		URL:       "https://paste.rs/old1",
		ExpireAt:  now.Add(-25 * time.Hour),
		CreatedAt: now.Add(-31 * time.Hour),
	}
	store.Entries = map[string]PasteEntry{"old1": expired}

	sweeper := &PasteSweeper{
		Store:   store,
		Deleter: fakeDeleter,
		Now:     func() time.Time { return now },
	}

	result, err := sweeper.SweepExpired()
	if err != nil {
		t.Fatalf("SweepExpired() error = %v", err)
	}

	// Should have called delete but counts as reaped past grace period.
	if len(deleteCalled) != 1 {
		t.Errorf("DeletePaste called %d times, want 1", len(deleteCalled))
	}
	if result.Deleted != 1 {
		t.Errorf("result.Deleted = %d, want 1 (grace expired)", result.Deleted)
	}
	if result.Remaining != 0 {
		t.Errorf("result.Remaining = %d, want 0", result.Remaining)
	}
	if len(store.Entries) != 0 {
		t.Errorf("store should be empty, got %d entries", len(store.Entries))
	}
}

func TestDebugPasteSweep_DpasteIgnored(t *testing.T) {
	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)

	store := &InMemoryPasteStore{}
	deleteCalled := false
	fakeDeleter := &fakePasteDeleter{
		deleteFn: func(url string) error {
			deleteCalled = true
			return nil
		},
	}

	// dpaste.com URLs auto-expire and cannot be deleted via API.
	dpasteEntry := PasteEntry{
		ID:        "dp1",
		URL:       "https://dpaste.com/abc123",
		ExpireAt:  now.Add(-1 * time.Hour),
		CreatedAt: now.Add(-7 * time.Hour),
	}
	store.Entries = map[string]PasteEntry{"dp1": dpasteEntry}

	sweeper := &PasteSweeper{
		Store:   store,
		Deleter: fakeDeleter,
		Now:     func() time.Time { return now },
	}

	result, err := sweeper.SweepExpired()
	if err != nil {
		t.Fatalf("SweepExpired() error = %v", err)
	}

	// dpaste entries should be counted as deleted without calling Deleter.
	if deleteCalled {
		t.Error("DeletePaste should not be called for dpaste.com URLs")
	}
	if result.Deleted != 1 {
		t.Errorf("result.Deleted = %d, want 1 (dpaste auto-expires)", result.Deleted)
	}
	if len(store.Entries) != 0 {
		t.Errorf("store should be empty, got %d", len(store.Entries))
	}
}

func TestDebugPasteSweep_EmptyStore(t *testing.T) {
	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	store := &InMemoryPasteStore{}
	fakeDeleter := &fakePasteDeleter{
		deleteFn: func(url string) error {
			t.Errorf("unexpected DeletePaste call")
			return nil
		},
	}
	sweeper := &PasteSweeper{
		Store:   store,
		Deleter: fakeDeleter,
		Now:     func() time.Time { return now },
	}

	result, err := sweeper.SweepExpired()
	if err != nil {
		t.Fatalf("SweepExpired() error = %v", err)
	}
	if result.Deleted != 0 {
		t.Errorf("result.Deleted = %d, want 0", result.Deleted)
	}
	if result.Remaining != 0 {
		t.Errorf("result.Remaining = %d, want 0", result.Remaining)
	}
}

func TestDebugPasteSweep_MultipleExpiredDeletesAll(t *testing.T) {
	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	store := &InMemoryPasteStore{}
	deleted := make([]string, 0)
	fakeDeleter := &fakePasteDeleter{
		deleteFn: func(url string) error {
			deleted = append(deleted, url)
			return nil
		},
	}
	// Create 5 expired entries.
	entries := make([]PasteEntry, 5)
	for i := 0; i < 5; i++ {
		entries[i] = PasteEntry{
			ID:        string(rune('a' + i)),
			URL:       "https://paste.rs/entry" + string(rune('0'+i)),
			ExpireAt:  now.Add(-time.Duration(i+1) * time.Hour),
			CreatedAt: now.Add(-8 * time.Hour),
		}
	}
	store.Entries = make(map[string]PasteEntry, len(entries))
	for _, e := range entries {
		store.Entries[e.ID] = e
	}
	sweeper := &PasteSweeper{
		Store:   store,
		Deleter: fakeDeleter,
		Now:     func() time.Time { return now },
	}

	result, err := sweeper.SweepExpired()
	if err != nil {
		t.Fatalf("SweepExpired() error = %v", err)
	}
	if result.Deleted != 5 {
		t.Errorf("result.Deleted = %d, want 5", result.Deleted)
	}
	if result.Remaining != 0 {
		t.Errorf("result.Remaining = %d, want 0", result.Remaining)
	}
	if len(deleted) != 5 {
		t.Errorf("DeletePaste called %d times, want 5", len(deleted))
	}
}

// fakePasteDeleter is a test double for PasteDeleter.
type fakePasteDeleter struct {
	deleteFn func(url string) error
}

func (f *fakePasteDeleter) DeletePaste(url string) error {
	return f.deleteFn(url)
}
