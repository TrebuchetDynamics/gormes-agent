package contextrefs

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"sync"
)

const StatusPending = "pending"

// Record is the generic representation of an attached @ reference. Start and
// End are audit metadata only: stable IDs are derived from the referenced
// object, not from where the token appeared in a specific prompt.
type Record struct {
	Raw       string `json:"raw"`
	Kind      string `json:"kind"`
	Target    string `json:"target,omitempty"`
	Start     int    `json:"start,omitempty"`
	End       int    `json:"end,omitempty"`
	LineStart int    `json:"line_start,omitempty"`
	LineEnd   int    `json:"line_end,omitempty"`
}

type Handle struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Record Record `json:"record"`
}

type Store struct {
	mu    sync.Mutex
	order []string
	byID  map[string]Handle
}

func NewStore() *Store {
	return &Store{
		byID: make(map[string]Handle),
	}
}

func (s *Store) Put(record Record) Handle {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.byID == nil {
		s.byID = make(map[string]Handle)
	}
	id := StableID(record)
	if existing, ok := s.byID[id]; ok {
		return existing
	}
	handle := Handle{
		ID:     id,
		Status: StatusPending,
		Record: record,
	}
	s.byID[id] = handle
	s.order = append(s.order, id)
	return handle
}

func (s *Store) Snapshot() []Handle {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Handle, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.byID[id])
	}
	return out
}

func StableID(record Record) string {
	canonical := strings.Join([]string{
		strings.TrimSpace(record.Kind),
		strings.TrimSpace(record.Target),
		strconv.Itoa(record.LineStart),
		strconv.Itoa(record.LineEnd),
	}, "\x00")
	sum := sha256.Sum256([]byte(canonical))
	return "ctxref_" + hex.EncodeToString(sum[:])[:20]
}
