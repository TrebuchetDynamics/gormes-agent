package directory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/storage"
)

const channelDirectorySourcesFileName = "channel_directory_sources.json"

// Source is kept as the package-level compatibility name for the shared
// session-origin-shaped source value contract.
type Source = model.Source

// RememberedSourceEntry is kept as the package-level compatibility name for the
// shared remembered-source value contract.
type RememberedSourceEntry = model.RememberedSourceEntry

// RememberedSourceLedger is kept as the package-level compatibility name for
// the shared remembered-source ledger value contract.
type RememberedSourceLedger = model.RememberedSourceLedger

// RememberedSourceStore is the fakeable ledger seam used by Manager to persist
// allowed inbound channel sources without mutating channel_directory.json. A
// later refresh slice can merge this ledger into the directory read model.
type RememberedSourceStore interface {
	RememberSource(context.Context, RememberedSourceEntry) error
}

// ChannelDirectorySourceStore persists a remembered-source ledger under a
// caller-owned root. It is distinct from channel_directory.json on purpose.
type ChannelDirectorySourceStore struct {
	root string
	now  func() time.Time
}

func NewChannelDirectorySourceStore(root string) ChannelDirectorySourceStore {
	return ChannelDirectorySourceStore{root: strings.TrimSpace(root), now: time.Now}
}

func (s ChannelDirectorySourceStore) path() string {
	return filepath.Join(s.root, channelDirectorySourcesFileName)
}

// Load reads the remembered-source ledger. Missing ledgers are not failures;
// they simply contribute no session-discovered entries during refresh.
func (s ChannelDirectorySourceStore) Load() (RememberedSourceLedger, ChannelDirectoryEvidence) {
	body, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return RememberedSourceLedger{Platforms: map[string][]RememberedSourceEntry{}}, ChannelDirectoryEvidence{}
		}
		return RememberedSourceLedger{Platforms: map[string][]RememberedSourceEntry{}}, ChannelDirectoryEvidence{Code: "channel_directory_sources_invalid"}
	}
	ledger := RememberedSourceLedger{Platforms: map[string][]RememberedSourceEntry{}}
	if err := json.Unmarshal(body, &ledger); err != nil {
		return RememberedSourceLedger{Platforms: map[string][]RememberedSourceEntry{}}, ChannelDirectoryEvidence{Code: "channel_directory_sources_invalid"}
	}
	if ledger.Platforms == nil {
		ledger.Platforms = map[string][]RememberedSourceEntry{}
	}
	return ledger, ChannelDirectoryEvidence{}
}

func (s ChannelDirectorySourceStore) RememberSource(_ context.Context, entry RememberedSourceEntry) error {
	if strings.TrimSpace(s.root) == "" {
		return fmt.Errorf("channel directory source root is empty")
	}
	entry = model.NormalizeRememberedSourceEntry(entry)
	if entry.Platform == "" || entry.ID == "" {
		return nil
	}
	ledger := RememberedSourceLedger{Platforms: map[string][]RememberedSourceEntry{}}
	if body, err := os.ReadFile(s.path()); err == nil && len(body) > 0 {
		_ = json.Unmarshal(body, &ledger)
	}
	if ledger.Platforms == nil {
		ledger.Platforms = map[string][]RememberedSourceEntry{}
	}
	if s.now == nil {
		s.now = time.Now
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	entry.UpdatedAt = now
	ledger.UpdatedAt = now
	entries := ledger.Platforms[entry.Platform]
	for i, existing := range entries {
		if strings.TrimSpace(existing.ID) == entry.ID {
			entries[i] = entry
			ledger.Platforms[entry.Platform] = entries
			return s.save(ledger)
		}
	}
	ledger.Platforms[entry.Platform] = append(entries, entry)
	return s.save(ledger)
}

func (s ChannelDirectorySourceStore) save(ledger RememberedSourceLedger) error {
	return storage.WriteAtomicJSON(s.root, channelDirectorySourcesFileName, ".channel_directory_sources-*.tmp", ledger, nil)
}

func RememberedSourceEntryFromSource(source Source) RememberedSourceEntry {
	return model.RememberedSourceEntryFromSource(source)
}
