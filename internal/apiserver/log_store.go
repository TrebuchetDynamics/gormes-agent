package apiserver

import (
	"net/http"
	"sync"
	"time"
)

// LogEntry is a structured log event stored in the ring buffer.
type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Source  string `json:"source,omitempty"`
}

// LogStore is a thread-safe ring buffer for dashboard log entries.
type LogStore struct {
	mu    sync.RWMutex
	entries []LogEntry
	cap   int
	next  int
	full  bool
}

// NewLogStore creates a ring buffer that holds up to n entries.
func NewLogStore(n int) *LogStore {
	return &LogStore{
		entries: make([]LogEntry, n),
		cap:   n,
	}
}

// Append adds an entry to the ring buffer.
func (s *LogStore) Append(level, message, source string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[s.next] = LogEntry{
		Time:    time.Now().UTC().Format(time.RFC3339),
		Level:   level,
		Message: message,
		Source:  source,
	}
	s.next = (s.next + 1) % s.cap
	if s.next == 0 {
		s.full = true
	}
}

// Entries returns all entries in chronological order.
func (s *LogStore) Entries() []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.full {
		out := make([]LogEntry, s.next)
		copy(out, s.entries[:s.next])
		return out
	}
	out := make([]LogEntry, s.cap)
	copy(out, s.entries[s.next:])
	copy(out[s.cap-s.next:], s.entries[:s.next])
	return out
}

func (s *Server) handleDashboardLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	entries := s.logStore.Entries()
	if entries == nil {
		entries = []LogEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}
