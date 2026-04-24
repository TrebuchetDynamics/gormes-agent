package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// pairingSchemaVersion pins the on-disk layout of pairing.json. Bump when the
// JSON envelope changes so future readers can migrate older files.
const pairingSchemaVersion = 1

// pairingFileMode is the permission mask for pairing.json. Pairing state
// authorizes messaging adapters so the file is owner-read/write only, in line
// with upstream gateway/pairing.py's chmod 0600 discipline.
const pairingFileMode os.FileMode = 0o600

// PairedUser is one paired user on one platform, as seen by the gateway read
// model. It is the element type returned by Snapshot and is safe to expose to
// the future `gormes gateway status` command.
type PairedUser struct {
	Platform string
	UserID   string
	UserName string
	PairedAt time.Time
}

// pairedUserRecord is the JSON-wire form of a paired user. A separate type
// keeps json tags out of the exported surface and lets Snapshot return Go
// time.Time values while the file stores RFC3339 strings.
type pairedUserRecord struct {
	UserName string `json:"user_name,omitempty"`
	PairedAt string `json:"paired_at"`
}

// platformState tracks paired users on a single platform.
type platformState struct {
	Paired map[string]pairedUserRecord `json:"paired"`
}

// pairingFile is the on-disk JSON envelope.
type pairingFile struct {
	Version   int                      `json:"version"`
	Platforms map[string]platformState `json:"platforms"`
}

// PairingStore is the minimal gateway-local read model for per-platform
// paired/unpaired user state. It is the Go port of the subset of upstream
// gateway/pairing.py PairingStore that is needed before any approval UX.
//
// Pending codes, rate limits, and failed-attempt lockouts belong to the
// sibling "Pairing approval + rate-limit semantics" slice and are deliberately
// not represented here.
//
// PairingStore is safe for concurrent use; all reads and writes are serialized
// through an RWMutex.
type PairingStore struct {
	path string

	mu    sync.RWMutex
	clock func() time.Time
	state *pairingFile
}

// LoadPairingStore reads pairing.json at path into an in-memory store. A
// missing file is treated as an empty store — the cold-start shape of a fresh
// gormes install. A corrupt file surfaces an error so operators can see it
// rather than silently truncating paired users.
func LoadPairingStore(path string) (*PairingStore, error) {
	if path == "" {
		return nil, errors.New("gateway: PairingStore path is required")
	}

	store := &PairingStore{
		path:  path,
		clock: time.Now,
		state: emptyPairingFile(),
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gateway: read pairing.json at %s: %w", path, err)
	}
	if len(data) == 0 {
		return store, nil
	}

	var decoded pairingFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("gateway: decode pairing.json at %s: %w", path, err)
	}
	if decoded.Platforms == nil {
		decoded.Platforms = map[string]platformState{}
	}
	if decoded.Version == 0 {
		decoded.Version = pairingSchemaVersion
	}
	store.state = &decoded
	return store, nil
}

// SetClock injects a deterministic clock. Intended for tests.
func (s *PairingStore) SetClock(clock func() time.Time) {
	if s == nil || clock == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clock = clock
}

// IsPaired reports whether the given platform/user pair is currently paired.
func (s *PairingStore) IsPaired(platform, userID string) bool {
	if s == nil || platform == "" || userID == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	plat, ok := s.state.Platforms[platform]
	if !ok {
		return false
	}
	_, ok = plat.Paired[userID]
	return ok
}

// MarkPaired records the user as paired on the given platform and flushes
// pairing.json atomically. Re-marking an already-paired user refreshes the
// user_name/paired_at fields rather than duplicating the entry.
func (s *PairingStore) MarkPaired(platform, userID, userName string) error {
	if s == nil {
		return errors.New("gateway: nil PairingStore")
	}
	if platform == "" || userID == "" {
		return errors.New("gateway: platform and user_id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	plat, ok := s.state.Platforms[platform]
	if !ok || plat.Paired == nil {
		plat = platformState{Paired: map[string]pairedUserRecord{}}
	}
	plat.Paired[userID] = pairedUserRecord{
		UserName: userName,
		PairedAt: s.clock().UTC().Format(time.RFC3339Nano),
	}
	s.state.Platforms[platform] = plat
	return s.flushLocked()
}

// MarkUnpaired removes the user from the given platform's paired list and
// flushes pairing.json atomically. Unpairing an already-missing user is a
// no-op and does not error.
func (s *PairingStore) MarkUnpaired(platform, userID string) error {
	if s == nil {
		return errors.New("gateway: nil PairingStore")
	}
	if platform == "" || userID == "" {
		return errors.New("gateway: platform and user_id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	plat, ok := s.state.Platforms[platform]
	if !ok || plat.Paired == nil {
		return nil
	}
	if _, exists := plat.Paired[userID]; !exists {
		return nil
	}
	delete(plat.Paired, userID)
	if len(plat.Paired) == 0 {
		delete(s.state.Platforms, platform)
	} else {
		s.state.Platforms[platform] = plat
	}
	return s.flushLocked()
}

// Snapshot returns every paired user across every platform, sorted
// deterministically by (platform, user_id). Callers may mutate the returned
// slice; it is a fresh copy.
func (s *PairingStore) Snapshot() []PairedUser {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []PairedUser
	for platform, plat := range s.state.Platforms {
		for userID, rec := range plat.Paired {
			pairedAt, _ := time.Parse(time.RFC3339Nano, rec.PairedAt)
			out = append(out, PairedUser{
				Platform: platform,
				UserID:   userID,
				UserName: rec.UserName,
				PairedAt: pairedAt,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Platform == out[j].Platform {
			return out[i].UserID < out[j].UserID
		}
		return out[i].Platform < out[j].Platform
	})
	return out
}

// flushLocked writes the in-memory state to pairing.json atomically. Must be
// called with s.mu held for write.
func (s *PairingStore) flushLocked() error {
	payload, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("gateway: encode pairing.json: %w", err)
	}
	return writeFileAtomic(s.path, payload, pairingFileMode)
}

// writeFileAtomic writes data to path via a sibling temp file and rename, so
// readers always see either the previous complete file or the new one — never
// a partial write.
func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("gateway: create dir for %s: %w", path, err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("gateway: create temp file for %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	cleanupTmp := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanupTmp()
		return fmt.Errorf("gateway: write temp file for %s: %w", path, err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		cleanupTmp()
		return fmt.Errorf("gateway: chmod temp file for %s: %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanupTmp()
		return fmt.Errorf("gateway: fsync temp file for %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		cleanupTmp()
		return fmt.Errorf("gateway: close temp file for %s: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanupTmp()
		return fmt.Errorf("gateway: rename temp file into %s: %w", path, err)
	}
	return nil
}

func emptyPairingFile() *pairingFile {
	return &pairingFile{
		Version:   pairingSchemaVersion,
		Platforms: map[string]platformState{},
	}
}
