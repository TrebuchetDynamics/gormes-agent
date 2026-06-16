// Package logs provides a small, concurrency-safe in-memory ring buffer of the
// most recent dashboard log lines. It is intentionally bounded and process-
// local: the dashboard reads recent activity without a persistent log backend.
package logs

import "sync"

// Entry is a single retained log line.
type Entry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// Store keeps the last N log entries in a fixed-capacity ring buffer.
type Store struct {
	mu      sync.Mutex
	entries []Entry
	max     int
}

const defaultMaxEntries = 200

// NewStore returns a Store retaining up to max entries (a non-positive max
// falls back to the default). The parameter is kept named retentionDays for
// call-site compatibility but is treated as a max-entry count.
func NewStore(retentionDays int) *Store {
	max := retentionDays
	if max <= 0 {
		max = defaultMaxEntries
	}
	return &Store{max: max}
}

// Append records a log line, evicting the oldest entry when at capacity.
func (s *Store) Append(level, message string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, Entry{Level: level, Message: message})
	if len(s.entries) > s.max {
		s.entries = s.entries[len(s.entries)-s.max:]
	}
}

// Recent returns a copy of the retained entries, oldest first.
func (s *Store) Recent() []Entry {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}
