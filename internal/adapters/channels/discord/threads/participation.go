package threads

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const DefaultDiscordThreadParticipationLimit = 500

// ThreadParticipationOptions configures the persisted Discord thread tracker.
type ThreadParticipationOptions struct {
	Path       string
	MaxTracked int
}

// ThreadParticipationEvidence reports redacted tracker degraded-mode state.
type ThreadParticipationEvidence struct {
	Code string
}

// ThreadParticipationTracker stores the bounded set of Discord threads where
// the bot has participated. It keeps insertion order so eviction is stable.
type ThreadParticipationTracker struct {
	mu           sync.RWMutex
	path         string
	maxTracked   int
	threads      map[string]struct{}
	order        []string
	loadEvidence ThreadParticipationEvidence
}

// NewThreadParticipationTracker loads a tracker from disk. Missing, corrupt, or
// unreadable state never prevents gateway startup.
func NewThreadParticipationTracker(opts ThreadParticipationOptions) *ThreadParticipationTracker {
	path := strings.TrimSpace(opts.Path)
	if path == "" {
		path = defaultDiscordThreadParticipationPath()
	}
	maxTracked := opts.MaxTracked
	if maxTracked <= 0 {
		maxTracked = DefaultDiscordThreadParticipationLimit
	}
	tracker := &ThreadParticipationTracker{
		path:       path,
		maxTracked: maxTracked,
		threads:    map[string]struct{}{},
	}
	tracker.load()
	return tracker
}

// LoadEvidence returns any nonfatal load/reset evidence from startup.
func (t *ThreadParticipationTracker) LoadEvidence() ThreadParticipationEvidence {
	if t == nil {
		return ThreadParticipationEvidence{}
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.loadEvidence
}

// Contains reports whether id is a known participated Discord thread.
func (t *ThreadParticipationTracker) Contains(id string) bool {
	if t == nil {
		return false
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.threads[id]
	return ok
}

// Snapshot returns a stable oldest-to-newest copy for tests and diagnostics.
func (t *ThreadParticipationTracker) Snapshot() []string {
	if t == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]string, len(t.order))
	copy(out, t.order)
	return out
}

// Mark records id as participated and persists the bounded tracker.
func (t *ThreadParticipationTracker) Mark(id string) (ThreadParticipationEvidence, error) {
	if t == nil {
		return ThreadParticipationEvidence{}, nil
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ThreadParticipationEvidence{}, nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.threads[id]; ok {
		return ThreadParticipationEvidence{}, nil
	}
	t.threads[id] = struct{}{}
	t.order = append(t.order, id)
	t.enforceLimitLocked()
	if err := t.saveLocked(); err != nil {
		return ThreadParticipationEvidence{Code: "discord_thread_tracker_unwritable"}, err
	}
	return ThreadParticipationEvidence{}, nil
}

func (t *ThreadParticipationTracker) load() {
	if t.path == "" {
		return
	}
	raw, err := os.ReadFile(t.path)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.loadEvidence = ThreadParticipationEvidence{Code: "discord_thread_tracker_reset"}
		return
	}
	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil {
		t.loadEvidence = ThreadParticipationEvidence{Code: "discord_thread_tracker_reset"}
		return
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := t.threads[id]; ok {
			continue
		}
		t.threads[id] = struct{}{}
		t.order = append(t.order, id)
	}
	t.enforceLimitLocked()
}

func (t *ThreadParticipationTracker) enforceLimitLocked() {
	if t.maxTracked <= 0 || len(t.order) <= t.maxTracked {
		return
	}
	drop := len(t.order) - t.maxTracked
	for _, id := range t.order[:drop] {
		delete(t.threads, id)
	}
	t.order = append([]string(nil), t.order[drop:]...)
}

func (t *ThreadParticipationTracker) saveLocked() error {
	if strings.TrimSpace(t.path) == "" {
		return fmt.Errorf("discord thread tracker path is empty")
	}
	dir := filepath.Dir(t.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	raw, err := json.Marshal(t.order)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".discord-threads-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpName, t.path)
}

func defaultDiscordThreadParticipationPath() string {
	home := strings.TrimSpace(os.Getenv("GORMES_HOME"))
	if home == "" {
		if userHome, err := os.UserHomeDir(); err == nil {
			home = filepath.Join(userHome, ".gormes")
		}
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, "discord_threads.json")
}
