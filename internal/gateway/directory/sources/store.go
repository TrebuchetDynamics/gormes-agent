package sources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/storage"
)

const channelDirectorySourcesFileName = "channel_directory_sources.json"

// RememberedStore is the fakeable ledger seam used by Manager to persist
// allowed inbound channel sources without mutating channel_directory.json. A
// later refresh slice can merge this ledger into the directory read model.
type RememberedStore interface {
	RememberSource(context.Context, model.RememberedSourceEntry) error
}

// Store persists a remembered-source ledger under a caller-owned root. It is
// distinct from channel_directory.json on purpose.
type Store struct {
	root string
	now  func() time.Time
}

func NewStore(root string) Store {
	return Store{root: strings.TrimSpace(root), now: time.Now}
}

func (s Store) path() string {
	return filepath.Join(s.root, channelDirectorySourcesFileName)
}

// Load reads the remembered-source ledger. Missing ledgers are not failures;
// they simply contribute no session-discovered entries during refresh.
func (s Store) Load() (model.RememberedSourceLedger, model.Evidence) {
	ledger := model.RememberedSourceLedger{Platforms: map[string][]model.RememberedSourceEntry{}}
	if err := storage.ReadJSON(s.path(), &ledger); err != nil {
		if os.IsNotExist(err) {
			return model.RememberedSourceLedger{Platforms: map[string][]model.RememberedSourceEntry{}}, model.Evidence{}
		}
		return model.RememberedSourceLedger{Platforms: map[string][]model.RememberedSourceEntry{}}, model.Evidence{Code: "channel_directory_sources_invalid"}
	}
	if ledger.Platforms == nil {
		ledger.Platforms = map[string][]model.RememberedSourceEntry{}
	}
	return ledger, model.Evidence{}
}

func (s Store) RememberSource(_ context.Context, entry model.RememberedSourceEntry) error {
	if strings.TrimSpace(s.root) == "" {
		return fmt.Errorf("channel directory source root is empty")
	}
	entry = model.NormalizeRememberedSourceEntry(entry)
	if entry.Platform == "" || entry.ID == "" {
		return nil
	}
	ledger := model.RememberedSourceLedger{Platforms: map[string][]model.RememberedSourceEntry{}}
	_ = storage.ReadJSON(s.path(), &ledger)
	if ledger.Platforms == nil {
		ledger.Platforms = map[string][]model.RememberedSourceEntry{}
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

func (s Store) save(ledger model.RememberedSourceLedger) error {
	return storage.WriteAtomicJSON(s.root, channelDirectorySourcesFileName, ".channel_directory_sources-*.tmp", ledger, nil)
}
