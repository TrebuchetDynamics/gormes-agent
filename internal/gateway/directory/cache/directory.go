package cache

import (
	"errors"
	"os"
	"sort"
	"strings"

	gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/storage"
)

const channelDirectoryFileName = "channel_directory.json"

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
	return Store{file: storage.NewFile(root, channelDirectoryFileName, ".channel_directory-*.tmp", "channel directory")}
}

// NewDirectory returns a directory with initialized platform buckets.
func NewDirectory(updatedAt string) Directory {
	return Directory{UpdatedAt: strings.TrimSpace(updatedAt), Platforms: map[string][]model.Entry{}}
}

// UpsertEntries normalizes a platform bucket and merges complete target entries
// into the directory. It is the shared merge contract for adapter inventory and
// remembered-source refresh contributions.
func (d *Directory) UpsertEntries(platform string, entries ...model.Entry) int {
	if d.Platforms == nil {
		d.Platforms = map[string][]model.Entry{}
	}
	platform = model.NormalizePlatform(platform)
	if platform == "" {
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
	return s.file.WithDefaults(channelDirectoryFileName, ".channel_directory-*.tmp", "channel directory")
}

// Load reads the directory. Missing or invalid files return empty directories
// plus structured degraded evidence.
func (s Store) Load() (Directory, model.Evidence) {
	var dir Directory
	if err := s.jsonFile().Read(&dir); errors.Is(err, os.ErrNotExist) {
		return emptyDirectory(), model.Evidence{Code: "channel_directory_missing"}
	} else if err != nil {
		return emptyDirectory(), model.Evidence{Code: "channel_directory_invalid"}
	}
	if dir.Platforms == nil {
		dir.Platforms = map[string][]model.Entry{}
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
	if dir.Platforms == nil {
		dir.Platforms = map[string][]model.Entry{}
	}
	return s.jsonFile().WriteAtomic(dir, storage.Writer(writer))
}

func emptyDirectory() Directory {
	return NewDirectory("")
}

// Resolve resolves a human-friendly channel target for platform into a concrete
// gatewaydelivery.Target. It follows Hermes matching order: exact ID, exact display/name,
// Discord guild-qualified name, then unambiguous prefix.
func (d Directory) Resolve(platform, query string) (gatewaydelivery.Target, model.Evidence) {
	platform = model.NormalizePlatform(platform)
	raw := strings.TrimSpace(query)
	entries := d.Platforms[platform]
	if platform == "" || raw == "" || len(entries) == 0 {
		return gatewaydelivery.Target{}, model.Evidence{Code: "channel_directory_missing", Platform: platform, Query: raw}
	}
	for _, entry := range entries {
		if entry.ID == raw {
			return model.DeliveryTarget(platform, entry), model.Evidence{}
		}
	}
	normalized := model.NormalizeQuery(raw)
	for _, entry := range entries {
		if model.NormalizeQuery(entry.Name) == normalized || model.NormalizeQuery(model.TargetDisplayName(platform, entry)) == normalized {
			return model.DeliveryTarget(platform, entry), model.Evidence{}
		}
	}
	if strings.Contains(normalized, "/") {
		parts := strings.Split(normalized, "/")
		guildPart := strings.Join(parts[:len(parts)-1], "/")
		channelPart := parts[len(parts)-1]
		for _, entry := range entries {
			if strings.ToLower(strings.TrimSpace(entry.Guild)) == guildPart && model.NormalizeQuery(entry.Name) == channelPart {
				return model.DeliveryTarget(platform, entry), model.Evidence{}
			}
		}
	}
	matches := make([]model.Entry, 0, 1)
	for _, entry := range entries {
		if strings.HasPrefix(model.NormalizeQuery(entry.Name), normalized) {
			matches = append(matches, entry)
		}
	}
	switch len(matches) {
	case 1:
		return model.DeliveryTarget(platform, matches[0]), model.Evidence{}
	case 0:
		return gatewaydelivery.Target{}, model.Evidence{Code: "channel_directory_missing", Platform: platform, Query: raw}
	default:
		return gatewaydelivery.Target{}, model.Evidence{Code: "channel_target_ambiguous", Platform: platform, Query: raw}
	}
}

// ValidateDeliveryTarget returns channel_target_stale when an explicit target
// no longer appears in a refreshed platform directory. Platform-only, local,
// origin, and unknown-directory targets are left to existing home-channel and
// missing-directory resolution paths.
func (d Directory) ValidateDeliveryTarget(target gatewaydelivery.Target) (gatewaydelivery.Target, model.Evidence) {
	platform := model.NormalizePlatform(target.Platform)
	if target.IsOrigin || !target.IsExplicit || platform == "" || strings.EqualFold(platform, "local") || strings.TrimSpace(target.ChatID) == "" {
		return target, model.Evidence{}
	}
	entries, ok := d.Platforms[platform]
	if !ok || len(entries) == 0 {
		return target, model.Evidence{}
	}
	for _, entry := range entries {
		candidate := model.DeliveryTarget(platform, entry)
		if candidate.ChatID == strings.TrimSpace(target.ChatID) && candidate.ThreadID == strings.TrimSpace(target.ThreadID) {
			return target, model.Evidence{}
		}
	}
	return gatewaydelivery.Target{}, model.Evidence{Code: "channel_target_stale", Platform: platform, Query: target.String()}
}

// LookupType returns the cached channel type for a platform target ID.
func (d Directory) LookupType(platform, id string) string {
	platform = model.NormalizePlatform(platform)
	id = strings.TrimSpace(id)
	for _, entry := range d.Platforms[platform] {
		if entry.ID == id {
			return strings.TrimSpace(entry.Type)
		}
	}
	return ""
}

// FormatForDisplay returns a Hermes-style target list for model/tool guidance.
func (d Directory) FormatForDisplay() string {
	if !d.hasEntries() {
		return "No messaging platforms connected or no channels discovered yet."
	}
	lines := []string{"Available messaging targets:\n"}
	platforms := make([]string, 0, len(d.Platforms))
	for platform, entries := range d.Platforms {
		if len(entries) > 0 {
			platforms = append(platforms, platform)
		}
	}
	sort.Strings(platforms)
	for _, platform := range platforms {
		entries := append([]model.Entry(nil), d.Platforms[platform]...)
		if platform == "discord" {
			guilds := map[string][]model.Entry{}
			dms := []model.Entry{}
			for _, entry := range entries {
				if guild := strings.TrimSpace(entry.Guild); guild != "" {
					guilds[guild] = append(guilds[guild], entry)
				} else {
					dms = append(dms, entry)
				}
			}
			guildNames := make([]string, 0, len(guilds))
			for guild := range guilds {
				guildNames = append(guildNames, guild)
			}
			sort.Strings(guildNames)
			for _, guild := range guildNames {
				lines = append(lines, "Discord ("+guild+"):")
				model.SortEntriesByNameID(guilds[guild])
				for _, entry := range guilds[guild] {
					lines = append(lines, "  discord:"+model.TargetDisplayName(platform, entry))
				}
			}
			if len(dms) > 0 {
				lines = append(lines, "Discord (DMs):")
				for _, entry := range dms {
					lines = append(lines, "  discord:"+model.TargetDisplayName(platform, entry))
				}
			}
			lines = append(lines, "")
			continue
		}
		lines = append(lines, strings.Title(platform)+":")
		for _, entry := range entries {
			lines = append(lines, "  "+platform+":"+model.TargetDisplayName(platform, entry))
		}
		lines = append(lines, "")
	}
	lines = append(lines, `Use these as the "target" parameter when sending.`)
	lines = append(lines, `Bare platform name (e.g. "telegram") sends to home channel.`)
	return strings.Join(lines, "\n")
}

func (d Directory) hasEntries() bool {
	for _, entries := range d.Platforms {
		if len(entries) > 0 {
			return true
		}
	}
	return false
}
