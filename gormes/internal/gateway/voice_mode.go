package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// VoiceMode is the persisted per-chat reply mode exposed via /voice.
type VoiceMode string

const (
	VoiceModeOff       VoiceMode = "off"
	VoiceModeVoiceOnly VoiceMode = "voice_only"
	VoiceModeAll       VoiceMode = "all"
)

// VoiceModeRecord is the JSON-backed operator surface shared across gateway
// restarts.
type VoiceModeRecord struct {
	Platform          string    `json:"platform"`
	ChatID            string    `json:"chat_id"`
	ChatName          string    `json:"chat_name,omitempty"`
	Mode              VoiceMode `json:"mode"`
	UpdatedAt         time.Time `json:"updated_at"`
	UpdatedByUserID   string    `json:"updated_by_user_id,omitempty"`
	UpdatedByUserName string    `json:"updated_by_user_name,omitempty"`
}

// VoiceModeStore persists per-chat /voice state to one JSON file.
type VoiceModeStore struct {
	mu      sync.RWMutex
	path    string
	entries map[string]VoiceModeRecord
}

func newVoiceModeStore(path string) *VoiceModeStore {
	return &VoiceModeStore{
		path:    strings.TrimSpace(path),
		entries: map[string]VoiceModeRecord{},
	}
}

// OpenVoiceModeStore loads the voice mode JSON surface from disk.
func OpenVoiceModeStore(path string) (*VoiceModeStore, error) {
	store := newVoiceModeStore(path)
	if store.path == "" {
		return store, nil
	}

	if err := os.MkdirAll(filepath.Dir(store.path), 0o700); err != nil {
		return nil, fmt.Errorf("voice mode: create parent dir for %s: %w", store.path, err)
	}

	data, err := os.ReadFile(store.path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, fmt.Errorf("voice mode: read %s: %w", store.path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return store, nil
	}

	var records []VoiceModeRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("voice mode: decode %s: %w", store.path, err)
	}
	for _, record := range records {
		record = normalizeVoiceModeRecord(record)
		if record.Platform == "" || record.ChatID == "" {
			continue
		}
		store.entries[voiceModeKey(record.Platform, record.ChatID)] = record
	}
	return store, nil
}

// Path returns the persisted JSON location.
func (s *VoiceModeStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Put inserts or replaces one per-chat voice mode record.
func (s *VoiceModeStore) Put(record VoiceModeRecord) error {
	if s == nil {
		return nil
	}
	record = normalizeVoiceModeRecord(record)
	if record.Platform == "" || record.ChatID == "" {
		return fmt.Errorf("voice mode: platform and chat_id are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[voiceModeKey(record.Platform, record.ChatID)] = record
	return s.persistLocked()
}

// Lookup returns one persisted per-chat voice mode.
func (s *VoiceModeStore) Lookup(platform, chatID string) (VoiceModeRecord, bool) {
	if s == nil {
		return VoiceModeRecord{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.entries[voiceModeKey(normalizePlatform(platform), strings.TrimSpace(chatID))]
	return record, ok
}

// Snapshot returns every stored voice mode in deterministic order.
func (s *VoiceModeStore) Snapshot() []VoiceModeRecord {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return snapshotVoiceModes(s.entries)
}

func (s *VoiceModeStore) persistLocked() error {
	if strings.TrimSpace(s.path) == "" {
		return nil
	}
	data, err := json.MarshalIndent(snapshotVoiceModes(s.entries), "", "  ")
	if err != nil {
		return fmt.Errorf("voice mode: encode %s: %w", s.path, err)
	}
	return writeJSONAtomic(s.path, data, 0o600)
}

func normalizeVoiceModeRecord(record VoiceModeRecord) VoiceModeRecord {
	record.Platform = normalizePlatform(record.Platform)
	record.ChatID = strings.TrimSpace(record.ChatID)
	record.ChatName = strings.TrimSpace(record.ChatName)
	record.UpdatedByUserID = strings.TrimSpace(record.UpdatedByUserID)
	record.UpdatedByUserName = strings.TrimSpace(record.UpdatedByUserName)
	record.Mode = normalizeVoiceMode(record.Mode)
	if record.Mode == "" {
		record.Mode = VoiceModeOff
	}
	return record
}

func snapshotVoiceModes(entries map[string]VoiceModeRecord) []VoiceModeRecord {
	out := make([]VoiceModeRecord, 0, len(entries))
	for _, record := range entries {
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Platform != out[j].Platform {
			return out[i].Platform < out[j].Platform
		}
		if out[i].ChatID != out[j].ChatID {
			return out[i].ChatID < out[j].ChatID
		}
		return out[i].UpdatedAt.Before(out[j].UpdatedAt)
	})
	return out
}

func normalizeVoiceMode(mode VoiceMode) VoiceMode {
	switch strings.ToLower(strings.TrimSpace(string(mode))) {
	case string(VoiceModeVoiceOnly):
		return VoiceModeVoiceOnly
	case string(VoiceModeAll):
		return VoiceModeAll
	case string(VoiceModeOff):
		return VoiceModeOff
	default:
		return VoiceModeOff
	}
}

func voiceModeKey(platform, chatID string) string {
	return platform + "\x00" + chatID
}

func voiceModeUsageText() string {
	return strings.Join([]string{
		"Voice mode commands:",
		"`/voice` toggle on/off",
		"`/voice on` voice replies only for voice messages",
		"`/voice tts` voice replies for all messages",
		"`/voice off` disable voice replies",
		"`/voice status` show the current setting",
	}, "\n")
}

func voiceModeStatusMessage(mode VoiceMode) string {
	return fmt.Sprintf("Current voice mode: %s", mode)
}

func voiceModeSetMessage(mode VoiceMode) string {
	return fmt.Sprintf("Voice mode is now %s for this chat.", mode)
}

func toggleVoiceMode(current VoiceMode) VoiceMode {
	if current == VoiceModeOff {
		return VoiceModeVoiceOnly
	}
	return VoiceModeOff
}

func lookupVoiceMode(store *VoiceModeStore, platform, chatID string) VoiceMode {
	if record, ok := store.Lookup(platform, chatID); ok {
		return record.Mode
	}
	return VoiceModeOff
}

func voiceModeForAction(action string, current VoiceMode) (VoiceMode, bool) {
	switch action {
	case "toggle":
		return toggleVoiceMode(current), true
	case "on":
		return VoiceModeVoiceOnly, true
	case "tts":
		return VoiceModeAll, true
	case "off":
		return VoiceModeOff, true
	default:
		return "", false
	}
}

func (m *Manager) handleVoiceCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	action := strings.ToLower(strings.TrimSpace(ev.Text))
	if action == "" {
		action = "toggle"
	}

	current := lookupVoiceMode(m.cfg.VoiceModes, ev.Platform, ev.ChatID)

	if action == "status" {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, voiceModeStatusMessage(current))
		return
	}

	if next, ok := voiceModeForAction(action, current); ok {
		m.applyVoiceMode(ctx, ch, ev, next)
		return
	}

	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, voiceModeUsageText())
}

func (m *Manager) applyVoiceMode(ctx context.Context, ch Channel, ev InboundEvent, mode VoiceMode) {
	record := VoiceModeRecord{
		Platform:          ev.Platform,
		ChatID:            ev.ChatID,
		ChatName:          ev.ChatName,
		Mode:              mode,
		UpdatedAt:         time.Now().UTC(),
		UpdatedByUserID:   ev.UserID,
		UpdatedByUserName: ev.UserName,
	}
	if err := m.cfg.VoiceModes.Put(record); err != nil {
		m.log.Warn("persist voice mode", "platform", ev.Platform, "chat_id", ev.ChatID, "err", err)
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Failed to persist voice mode.")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, voiceModeSetMessage(mode))
}
