package cli

import (
	"errors"
	"testing"
	"time"
)

func TestDebugPasteSweep_ExpiredEntriesDeletedWithFakeClock(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	queue := NewDebugPasteQueue([]DebugPasteEntry{
		{URL: "https://paste.rs/expired", ExpiresAt: now.Add(-time.Minute)},
		{URL: "https://paste.rs/future", ExpiresAt: now.Add(time.Hour)},
	})
	var deleted []string
	result := queue.SweepExpired(now, func(url string) error {
		deleted = append(deleted, url)
		return nil
	})
	if result.Deleted != 1 || result.Remaining != 1 || len(deleted) != 1 || deleted[0] != "https://paste.rs/expired" {
		t.Fatalf("unexpected sweep result=%+v deleted=%v", result, deleted)
	}
	if got := queue.Entries(); len(got) != 1 || got[0].URL != "https://paste.rs/future" {
		t.Fatalf("expired entry was not removed: %#v", got)
	}
}

func TestDebugPasteSweep_UnexpiredEntriesRemain(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	queue := NewDebugPasteQueue([]DebugPasteEntry{{URL: "https://paste.rs/future", ExpiresAt: now.Add(time.Second)}})
	result := queue.SweepExpired(now, func(url string) error { t.Fatalf("deleter called for %s", url); return nil })
	if result.Deleted != 0 || result.Remaining != 1 || len(result.Evidence) != 0 {
		t.Fatalf("unexpected result for unexpired entry: %+v", result)
	}
}

func TestDebugPasteSweep_DeleteFailureRetainsEntryWithEvidence(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	queue := NewDebugPasteQueue([]DebugPasteEntry{{URL: "https://paste.rs/expired", ExpiresAt: now.Add(-time.Second)}})
	result := queue.SweepExpired(now, func(string) error { return errors.New("offline") })
	if result.Deleted != 0 || result.Remaining != 1 {
		t.Fatalf("failed delete should retain entry: %+v", result)
	}
	if len(result.Evidence) != 1 || result.Evidence[0].Code != DebugPasteEvidenceDeleteFailed || result.Evidence[0].URL != "https://paste.rs/expired" {
		t.Fatalf("missing delete-failure evidence: %+v", result.Evidence)
	}
	if got := queue.Entries(); len(got) != 1 || got[0].URL != "https://paste.rs/expired" {
		t.Fatalf("failed delete removed entry: %#v", got)
	}
}
