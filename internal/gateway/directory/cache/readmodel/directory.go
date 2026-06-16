package readmodel

import (
	"errors"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/storage"
)

const channelDirectoryFileName = "channel_directory.json"

var channelDirectoryFileSpec = storage.Spec{Name: channelDirectoryFileName, TmpPattern: ".channel_directory-*.tmp", Label: "channel directory"}

// Directory is the channel-neutral cached read model for reachable messaging
// targets. It mirrors Hermes' channel_directory.json shape while keeping Gormes
// runtime behavior native Go and fixture-driven.
type Directory struct {
	UpdatedAt string                   `json:"updated_at,omitempty"`
	Platforms map[string][]model.Entry `json:"platforms"`
}

// Store persists channel_directory.json under a caller-owned Gormes state/home
// root. Tests pass temp roots; no live operator home is read.
type Store struct {
	file storage.File
}

// NewStore returns a store rooted at root.
func NewStore(root string) Store {
	return Store{file: channelDirectoryFileSpec.File(root)}
}

// NewDirectory returns a directory with initialized platform buckets.
func NewDirectory(updatedAt string) Directory {
	return Directory{UpdatedAt: strings.TrimSpace(updatedAt), Platforms: model.EmptyPlatformBuckets[model.Entry]()}
}

// UpsertEntries normalizes a platform bucket and merges complete target entries
// into the directory. It is the shared merge contract for adapter inventory and
// remembered-source refresh contributions.
func (d *Directory) UpsertEntries(platform string, entries ...model.Entry) int {
	d.Platforms = model.EnsurePlatformBuckets(d.Platforms)
	platform = model.NormalizePlatform(platform)
	if platform == "" || containsDirectoryControl(platform) {
		return 0
	}
	merged := 0
	for _, entry := range entries {
		var ok bool
		d.Platforms[platform], ok = model.UpsertValidEntry(d.Platforms[platform], entry)
		if ok {
			merged++
		}
	}
	return merged
}

// Root returns the store root for fixture setup.
func (s Store) Root() string { return s.jsonFile().Root.String() }

func (s Store) jsonFile() storage.File {
	return channelDirectoryFileSpec.Apply(s.file)
}

// Load reads the directory. Missing or invalid files return empty directories
// plus structured degraded evidence.
func (s Store) Load() (Directory, model.Evidence) {
	file := s.jsonFile()
	if err := file.Require(); err != nil {
		return emptyDirectory(), model.Evidence{Code: model.EvidenceChannelDirectoryInvalid}
	}
	dir, err := storage.LoadValue(file, emptyDirectory, ensureDirectory)
	if errors.Is(err, os.ErrNotExist) {
		return emptyDirectory(), model.Evidence{Code: model.EvidenceChannelDirectoryMissing}
	} else if err != nil {
		return emptyDirectory(), model.Evidence{Code: model.EvidenceChannelDirectoryInvalid}
	}
	return dir, model.Evidence{}
}

// Save atomically writes the directory.
func (s Store) Save(dir Directory) error {
	return s.SaveWithWriter(dir, os.WriteFile)
}

// SaveWithWriter exists for deterministic atomic-write failure tests. It writes
// a temp file, then renames only after the writer succeeds, so old complete JSON
// remains visible after an injected partial-write failure.
func (s Store) SaveWithWriter(dir Directory, writer func(string, []byte, os.FileMode) error) error {
	if writer == nil {
		writer = os.WriteFile
	}
	dir = normalizeDecodedDirectory(dir)
	return s.jsonFile().WriteAtomic(dir, storage.Writer(writer))
}

func emptyDirectory() Directory {
	return NewDirectory("")
}

func ensureDirectory(dir Directory) Directory {
	return normalizeDecodedDirectory(dir)
}

// normalizeDecodedDirectory is the load-time value contract for persisted
// directory JSON. Adapter refreshes already upsert normalized entries; decoded
// operator-edited or older cache files need the same platform/field trimming so
// lookup, display, and stale-target validation all see one canonical shape.
func normalizeDecodedDirectory(dir Directory) Directory {
	dir.UpdatedAt = strings.Join(strings.Fields(dir.UpdatedAt), " ")
	platforms := model.EmptyPlatformBuckets[model.Entry]()
	for platform, entries := range model.EnsurePlatformBuckets(dir.Platforms) {
		platform = model.NormalizePlatform(platform)
		if platform == "" || containsDirectoryControl(platform) {
			continue
		}
		for _, entry := range entries {
			platforms[platform], _ = model.UpsertValidEntry(platforms[platform], entry)
		}
	}
	dir.Platforms = platforms
	return dir
}

func containsDirectoryControl(value string) bool {
	return strings.ContainsFunc(value, func(r rune) bool { return r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) })
}
