// Package voicemode owns the per-chat voice reply mode used by the gateway.
//
// Hermes exposes /voice [on|off|tts|status] and persists the mode per chat in
// gateway_voice_mode.json. The mode controls whether replies are spoken:
//
//	off        - never speak replies, even for voice input
//	voice_only - speak replies only when the inbound turn requested audio
//	all        - speak every reply
//
// The package is intentionally text-only so it builds in slim binaries; the
// audio engine remains owned by the tools layer.
package voicemode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Mode is the persisted per-chat voice reply mode.
type Mode string

const (
	ModeOff       Mode = "off"
	ModeVoiceOnly Mode = "voice_only"
	ModeAll       Mode = "all"
)

// String returns the persisted form of the mode.
func (m Mode) String() string { return string(m) }

// Valid reports whether the mode is a known persisted value.
func (m Mode) Valid() bool {
	switch m {
	case ModeOff, ModeVoiceOnly, ModeAll:
		return true
	default:
		return false
	}
}

// Label returns the operator-facing label used by /voice status.
func (m Mode) Label() string {
	switch m {
	case ModeOff:
		return "off"
	case ModeVoiceOnly:
		return "voice only"
	case ModeAll:
		return "all replies"
	default:
		return string(m)
	}
}

// Parse normalizes a /voice argument into a mode. Hermes aliases on/enable to
// voice_only and tts to all.
func Parse(raw string) (Mode, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "off", "disable", "disabled", "false", "0":
		return ModeOff, true
	case "on", "enable", "enabled", "voice_only", "voice-only", "voiceonly", "1":
		return ModeVoiceOnly, true
	case "tts", "all", "2", "true":
		return ModeAll, true
	default:
		return "", false
	}
}

// Store persists per-chat voice modes. Get reports found=false when the chat
// has no explicit mode, which lets delivery fall back to the legacy TTS config.
type Store interface {
	Get(ctx context.Context, chatKey string) (mode Mode, found bool, err error)
	Set(ctx context.Context, chatKey string, mode Mode) error
	List(ctx context.Context) (map[string]Mode, error)
}

// DefaultFileName is the Hermes-compatible state file name under Gormes home.
const DefaultFileName = "gateway_voice_mode.json"

// DefaultPath returns the state path inside a Gormes home directory.
func DefaultPath(home string) string {
	return filepath.Join(home, DefaultFileName)
}

// MemoryStore is an in-process Store for tests and ephemeral runtimes.
type MemoryStore struct {
	mu    sync.RWMutex
	modes map[string]Mode
}

// NewMemoryStore returns an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{modes: make(map[string]Mode)}
}

// Get returns the mode for chatKey.
func (s *MemoryStore) Get(_ context.Context, chatKey string) (Mode, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mode, ok := s.modes[chatKey]
	return mode, ok, nil
}

// Set stores the mode for chatKey.
func (s *MemoryStore) Set(_ context.Context, chatKey string, mode Mode) error {
	if err := validate(chatKey, mode); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modes[chatKey] = mode
	return nil
}

// List returns a copy of every stored mode.
func (s *MemoryStore) List(_ context.Context) (map[string]Mode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]Mode, len(s.modes))
	for key, mode := range s.modes {
		out[key] = mode
	}
	return out, nil
}

// FileStore persists modes as a JSON object.
type FileStore struct {
	path string
	mu   sync.Mutex
}

// NewFileStore returns a store backed by path.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// Path returns the backing file path.
func (s *FileStore) Path() string { return s.path }

// Get returns the mode for chatKey.
func (s *FileStore) Get(_ context.Context, chatKey string) (Mode, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	modes, err := s.readAll()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	mode, ok := modes[chatKey]
	return mode, ok, nil
}

// Set stores the mode for chatKey.
func (s *FileStore) Set(_ context.Context, chatKey string, mode Mode) error {
	if err := validate(chatKey, mode); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	modes, err := s.readAll()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		modes = make(map[string]Mode)
	}
	modes[chatKey] = mode
	return s.writeAll(modes)
}

// List returns a copy of every stored mode.
func (s *FileStore) List(_ context.Context) (map[string]Mode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	modes, err := s.readAll()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]Mode{}, nil
		}
		return nil, err
	}
	return modes, nil
}

func (s *FileStore) readAll() (map[string]Mode, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	var modes map[string]Mode
	if err := json.Unmarshal(raw, &modes); err != nil {
		return nil, fmt.Errorf("voicemode: parse %s: %w", s.path, err)
	}
	for key, mode := range modes {
		if !mode.Valid() {
			delete(modes, key)
		}
	}
	return modes, nil
}

func (s *FileStore) writeAll(modes map[string]Mode) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(modes, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".gateway_voice_mode-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, s.path)
}

func validate(chatKey string, mode Mode) error {
	if strings.TrimSpace(chatKey) == "" {
		return errors.New("voicemode: chat key is required")
	}
	if !mode.Valid() {
		return fmt.Errorf("voicemode: unknown mode %q", mode)
	}
	return nil
}
