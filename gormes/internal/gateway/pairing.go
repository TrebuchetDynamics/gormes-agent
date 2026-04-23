package gateway

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// PairingStatus tracks whether an operator still needs to approve a user.
type PairingStatus string

const (
	PairingStatusPending  PairingStatus = "pending"
	PairingStatusApproved PairingStatus = "approved"
)

// PairingRecord is the shared persistence shape for runtime pairing state.
type PairingRecord struct {
	Platform    string        `json:"platform"`
	UserID      string        `json:"user_id"`
	UserName    string        `json:"user_name,omitempty"`
	ChatID      string        `json:"chat_id,omitempty"`
	Code        string        `json:"code,omitempty"`
	Status      PairingStatus `json:"status"`
	RequestedAt time.Time     `json:"requested_at"`
	ApprovedAt  *time.Time    `json:"approved_at,omitempty"`
	ApprovedBy  string        `json:"approved_by,omitempty"`
}

// PairingStore persists pairing records to one JSON file so adapters can
// share the same operator-visible state across restarts.
type PairingStore struct {
	mu      sync.RWMutex
	path    string
	entries map[string]PairingRecord
}

// OpenPairingStore loads a JSON-backed pairing store from path. Missing files
// are treated as an empty store.
func OpenPairingStore(path string) (*PairingStore, error) {
	store := &PairingStore{
		path:    path,
		entries: map[string]PairingRecord{},
	}
	if strings.TrimSpace(path) == "" {
		return store, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("pairing: create parent dir for %s: %w", path, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, fmt.Errorf("pairing: read %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return store, nil
	}

	var records []PairingRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("pairing: decode %s: %w", path, err)
	}
	for _, record := range records {
		record = normalizePairingRecord(record)
		if record.Platform == "" || record.UserID == "" {
			continue
		}
		store.entries[pairingKey(record.Platform, record.UserID)] = record
	}
	return store, nil
}

// Put inserts or replaces one pairing record and flushes the store to disk.
func (s *PairingStore) Put(record PairingRecord) error {
	if s == nil {
		return nil
	}
	record = normalizePairingRecord(record)
	if record.Platform == "" || record.UserID == "" {
		return fmt.Errorf("pairing: platform and user_id are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[pairingKey(record.Platform, record.UserID)] = record
	return s.persistLocked()
}

// Snapshot returns a deterministic copy of every persisted pairing record.
func (s *PairingStore) Snapshot() []PairingRecord {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return snapshotPairings(s.entries)
}

func (s *PairingStore) persistLocked() error {
	if strings.TrimSpace(s.path) == "" {
		return nil
	}

	data, err := json.MarshalIndent(snapshotPairings(s.entries), "", "  ")
	if err != nil {
		return fmt.Errorf("pairing: encode %s: %w", s.path, err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("pairing: write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("pairing: rename %s -> %s: %w", tmp, s.path, err)
	}
	return nil
}

// StatusSnapshot is the operator-facing readout behind gateway /status.
type StatusSnapshot struct {
	Channels         []string
	ShuttingDown     bool
	PendingPairings  int
	ApprovedPairings int
	Pairings         []PairingRecord
}

// StatusSnapshot returns the current gateway control-plane summary.
func (m *Manager) StatusSnapshot() StatusSnapshot {
	pairings := m.cfg.Pairings.Snapshot()
	pending, approved := pairingCounts(pairings)
	snapshot := StatusSnapshot{
		Channels:         m.connectedPlatforms(),
		ShuttingDown:     m.isShuttingDown(),
		PendingPairings:  pending,
		ApprovedPairings: approved,
		Pairings:         pairings,
	}
	return snapshot
}

// FormatStatusReadout renders the stable operator-facing /status response.
func FormatStatusReadout(snapshot StatusSnapshot) string {
	var b strings.Builder
	b.WriteString("Gateway status\n")
	if len(snapshot.Channels) == 0 {
		b.WriteString("channels: none\n")
	} else {
		b.WriteString("channels: " + strings.Join(snapshot.Channels, ", ") + "\n")
	}
	if snapshot.ShuttingDown {
		b.WriteString("shutting_down: yes\n")
	} else {
		b.WriteString("shutting_down: no\n")
	}
	b.WriteString(fmt.Sprintf("pairings_pending: %d\n", snapshot.PendingPairings))
	b.WriteString(fmt.Sprintf("pairings_approved: %d\n", snapshot.ApprovedPairings))
	if len(snapshot.Pairings) == 0 {
		b.WriteString("pairings: none\n")
		return b.String()
	}
	b.WriteString("pairings:\n")
	for _, pairing := range snapshot.Pairings {
		b.WriteString(formatPairingLine(pairing))
		b.WriteByte('\n')
	}
	return b.String()
}

func snapshotPairings(entries map[string]PairingRecord) []PairingRecord {
	out := make([]PairingRecord, 0, len(entries))
	for _, record := range entries {
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Platform != out[j].Platform {
			return out[i].Platform < out[j].Platform
		}
		if out[i].UserID != out[j].UserID {
			return out[i].UserID < out[j].UserID
		}
		return out[i].RequestedAt.Before(out[j].RequestedAt)
	})
	return out
}

func pairingCounts(records []PairingRecord) (pending, approved int) {
	for _, record := range records {
		if record.Status == PairingStatusApproved {
			approved++
			continue
		}
		pending++
	}
	return pending, approved
}

func formatPairingLine(record PairingRecord) string {
	status := string(record.Status)
	if status == "" {
		status = string(PairingStatusPending)
	}
	line := fmt.Sprintf("- %s platform=%s user_id=%s chat_id=%s user_name=%q", status, record.Platform, record.UserID, record.ChatID, record.UserName)
	if record.Status == PairingStatusApproved {
		if record.ApprovedBy != "" {
			line += " approved_by=" + record.ApprovedBy
		}
		return line
	}
	if record.Code != "" {
		line += " code=" + record.Code
	}
	return line
}

func normalizePairingRecord(record PairingRecord) PairingRecord {
	record.Platform = normalizePlatform(record.Platform)
	record.UserID = strings.TrimSpace(record.UserID)
	record.UserName = strings.TrimSpace(record.UserName)
	record.ChatID = strings.TrimSpace(record.ChatID)
	record.Code = strings.TrimSpace(record.Code)
	record.ApprovedBy = strings.TrimSpace(record.ApprovedBy)
	record.Status = normalizePairingStatus(record.Status)
	if record.Status == "" {
		record.Status = PairingStatusPending
	}
	return record
}

func normalizePairingStatus(status PairingStatus) PairingStatus {
	switch strings.ToLower(strings.TrimSpace(string(status))) {
	case string(PairingStatusApproved):
		return PairingStatusApproved
	default:
		return PairingStatusPending
	}
}

func pairingKey(platform, userID string) string {
	return platform + "\x00" + userID
}
