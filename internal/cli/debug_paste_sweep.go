package cli

import (
	"sync"
	"time"
)

const DebugPasteEvidenceDeleteFailed = "paste_delete_failed"

// DebugPasteEntry records a pending paste.rs URL and when Gormes should attempt
// deletion. Callers keep persistence outside this helper so tests can stay
// hermetic and production can choose its store.
type DebugPasteEntry struct {
	URL       string
	ExpiresAt time.Time
}

type DebugPasteEvidence struct {
	Code  string `json:"code"`
	URL   string `json:"url,omitempty"`
	Error string `json:"error,omitempty"`
}

type DebugPasteSweepResult struct {
	Deleted   int                  `json:"deleted"`
	Remaining int                  `json:"remaining"`
	Evidence  []DebugPasteEvidence `json:"evidence,omitempty"`
}

type DebugPasteDeleter func(url string) error

// DebugPasteQueue is a small in-memory sweep model for Hermes-compatible
// debug-share paste cleanup. Persistence and network deletion are injected so
// the public contract is deterministic and unit-testable.
type DebugPasteQueue struct {
	mu      sync.Mutex
	entries []DebugPasteEntry
}

func NewDebugPasteQueue(entries []DebugPasteEntry) *DebugPasteQueue {
	q := &DebugPasteQueue{}
	q.entries = append(q.entries, entries...)
	return q
}

func (q *DebugPasteQueue) Entries() []DebugPasteEntry {
	if q == nil {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]DebugPasteEntry, len(q.entries))
	copy(out, q.entries)
	return out
}

func (q *DebugPasteQueue) SweepExpired(now time.Time, deleter DebugPasteDeleter) DebugPasteSweepResult {
	if q == nil {
		return DebugPasteSweepResult{}
	}
	if deleter == nil {
		deleter = func(string) error { return nil }
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	kept := q.entries[:0]
	result := DebugPasteSweepResult{}
	for _, entry := range q.entries {
		if entry.ExpiresAt.IsZero() || entry.ExpiresAt.After(now) {
			kept = append(kept, entry)
			continue
		}
		if err := deleter(entry.URL); err != nil {
			kept = append(kept, entry)
			result.Evidence = append(result.Evidence, DebugPasteEvidence{Code: DebugPasteEvidenceDeleteFailed, URL: entry.URL, Error: err.Error()})
			continue
		}
		result.Deleted++
	}
	q.entries = kept
	result.Remaining = len(q.entries)
	return result
}
