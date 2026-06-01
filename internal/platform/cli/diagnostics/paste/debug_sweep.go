// Package paste contains paste-related operator diagnostics.
// Debug paste sweep provides a persisted deletion queue for paste.rs debug-share
// URLs and an hourly scheduler hook that sweeps expired entries with bounded
// evidence while retaining opportunistic CLI-only cleanup as a fallback.
package paste

import "time"

// PasteEntry represents one pending paste deletion record.
type PasteEntry struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	ExpireAt  time.Time `json:"expire_at"`
	CreatedAt time.Time `json:"created_at"`
}

// PasteDeleter defines the interface for issuing a DELETE request to a paste
// service. Implementations must be hermetic — no live network in tests.
type PasteDeleter interface {
	DeletePaste(url string) error
}

// PasteStore defines the interface for loading and saving pending paste entries.
// In production this reads/writes a JSON file; in tests it uses an in-memory map.
type PasteStore interface {
	Load() ([]PasteEntry, error)
	Save(entries []PasteEntry) error
}

// InMemoryPasteStore is a hermetic in-memory implementation of PasteStore for
// testing and offline CLI usage.
type InMemoryPasteStore struct {
	Entries map[string]PasteEntry
}

// Load returns all stored paste entries.
func (s *InMemoryPasteStore) Load() ([]PasteEntry, error) {
	if s.Entries == nil {
		s.Entries = make(map[string]PasteEntry)
	}
	result := make([]PasteEntry, 0, len(s.Entries))
	for _, e := range s.Entries {
		result = append(result, e)
	}
	return result, nil
}

// Save replaces the stored entries with the provided slice.
func (s *InMemoryPasteStore) Save(entries []PasteEntry) error {
	s.Entries = make(map[string]PasteEntry, len(entries))
	for _, e := range entries {
		s.Entries[e.ID] = e
	}
	return nil
}

// PasteSweeper coordinates expired-paste deletion. It loads pending entries,
// deletes expired ones via the injected Deleter, and persists the updated
// queue. Failed deletes are retained for up to 24 hours past expiration and
// then given up (matching Hermes behavior).
type PasteSweeper struct {
	Store   PasteStore
	Deleter PasteDeleter
	// Now returns the current time. If nil, uses time.Now.
	Now func() time.Time
}

// SweepResult captures the outcome of one sweep pass.
type SweepResult struct {
	Deleted   int          `json:"deleted"`
	Remaining int          `json:"remaining"`
	Errors    []SweepError `json:"errors,omitempty"`
}

// SweepError describes one delete failure with evidence.
type SweepError struct {
	URL       string `json:"url"`
	Evidence  string `json:"evidence"`
	ExpiredAt string `json:"expired_at"`
}

// PasteURLScheme extracts the paste service name from a URL for validation.
// Only paste.rs is supported for deletion; dpaste.com expires automatically.
func PasteURLScheme(url string) string {
	const pasteRS = "https://paste.rs/"
	if len(url) >= len(pasteRS) && url[:len(pasteRS)] == pasteRS {
		return "paste.rs"
	}
	return ""
}

// SweepExpired synchronously deletes any pending pastes whose ExpireAt has
// passed. Returns (deleted, remaining). Failed deletes stay in the pending
// store and will be retried on the next sweep. This method is idempotent
// and safe to call on every CLI invocation as an opportunistic fallback.
func (s *PasteSweeper) SweepExpired() (*SweepResult, error) {
	now := s.Now
	if now == nil {
		now = time.Now
	}

	entries, err := s.Store.Load()
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return &SweepResult{Deleted: 0, Remaining: 0}, nil
	}

	current := now()
	var remaining []PasteEntry
	var result SweepResult

	for _, entry := range entries {
		if entry.ExpireAt.After(current) {
			remaining = append(remaining, entry)
			continue
		}

		scheme := PasteURLScheme(entry.URL)
		if scheme != "paste.rs" {
			// dpaste.com pastes auto-expire; skip retention logic.
			result.Deleted++
			continue
		}

		delErr := s.Deleter.DeletePaste(entry.URL)
		if delErr != nil {
			// Retain failed deletes for up to 24h past expiration, then give up.
			// Hermes grace period: expire_at + 86400 seconds.
			if entry.ExpireAt.Add(24 * time.Hour).After(current) {
				remaining = append(remaining, entry)
				result.Errors = append(result.Errors, SweepError{
					URL:       entry.URL,
					Evidence:  "paste_delete_failed",
					ExpiredAt: entry.ExpireAt.Format(time.RFC3339),
				})
			} else {
				// Past the 24h grace period — count as reaped; paste.rs will GC.
				result.Deleted++
			}
			continue
		}

		result.Deleted++
	}

	result.Remaining = len(remaining)
	if result.Deleted > 0 {
		if err := s.Store.Save(remaining); err != nil {
			return &result, err
		}
	}

	return &result, nil
}
