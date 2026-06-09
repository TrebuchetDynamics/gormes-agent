package sources

import (
	"context"
	"os"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/storage"
)

const channelDirectorySourcesFileName = "channel_directory_sources.json"

var channelDirectorySourcesFileSpec = storage.Spec{Name: channelDirectorySourcesFileName, TmpPattern: ".channel_directory_sources-*.tmp", Label: "channel directory source"}

// RememberedStore is the fakeable ledger seam used by Manager to persist
// allowed inbound channel sources without mutating channel_directory.json. A
// later refresh slice can merge this ledger into the directory read model.
type RememberedStore interface {
	RememberSource(context.Context, model.RememberedSourceEntry) error
}

// Store persists a remembered-source ledger under a caller-owned root. It is
// distinct from channel_directory.json on purpose.
type Store struct {
	file storage.File
	now  func() time.Time
}

func NewStore(root string) Store {
	return Store{file: channelDirectorySourcesFileSpec.File(root), now: time.Now}
}

func (s Store) jsonFile() storage.File {
	return channelDirectorySourcesFileSpec.Apply(s.file)
}

func (s Store) loadFile() (storage.File, bool) {
	file := s.jsonFile()
	if file.Require() != nil {
		return storage.File{}, false
	}
	return file, true
}

// Load reads the remembered-source ledger. Missing ledgers are not failures;
// they simply contribute no session-discovered entries during refresh.
func (s Store) Load() (model.RememberedSourceLedger, model.Evidence) {
	file, ok := s.loadFile()
	if !ok {
		return model.EmptyRememberedSourceLedger(), model.Evidence{}
	}
	ledger, err := storage.LoadValue(file, model.EmptyRememberedSourceLedger, model.EnsureRememberedSourceLedger)
	if err != nil {
		if os.IsNotExist(err) {
			return model.EmptyRememberedSourceLedger(), model.Evidence{}
		}
		return model.EmptyRememberedSourceLedger(), model.Evidence{Code: model.EvidenceChannelDirectorySourcesInvalid}
	}
	return ledger, model.Evidence{}
}

func (s Store) RememberSource(_ context.Context, entry model.RememberedSourceEntry) error {
	if err := s.jsonFile().Require(); err != nil {
		return err
	}
	entry = model.NormalizeRememberedSourceEntry(entry)
	if entry.Platform == "" || entry.ID == "" {
		return nil
	}
	ledger := model.EmptyRememberedSourceLedger()
	if err := s.jsonFile().Read(&ledger); err != nil && !os.IsNotExist(err) {
		return err
	}
	ledger = model.EnsureRememberedSourceLedger(ledger)
	if s.now == nil {
		s.now = time.Now
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	entry.UpdatedAt = now
	ledger.UpdatedAt = now
	ledger.Platforms[entry.Platform], _ = model.UpsertRememberedSourceEntry(ledger.Platforms[entry.Platform], entry)
	return s.save(ledger)
}

func (s Store) save(ledger model.RememberedSourceLedger) error {
	return s.jsonFile().WriteAtomic(ledger, nil)
}
