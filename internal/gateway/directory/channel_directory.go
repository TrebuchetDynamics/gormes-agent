package directory

import (
	"errors"
	"fmt"
	gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/storage"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const channelDirectoryFileName = "channel_directory.json"

// ChannelDirectory is the channel-neutral cached read model for reachable
// messaging targets. It mirrors Hermes' channel_directory.json shape while
// keeping Gormes runtime behavior native Go and fixture-driven.
type ChannelDirectory struct {
	UpdatedAt string                             `json:"updated_at,omitempty"`
	Platforms map[string][]ChannelDirectoryEntry `json:"platforms"`
}

// ChannelDirectoryEntry is kept as the package-level compatibility name for
// the shared directory target value contract.
type ChannelDirectoryEntry = model.Entry

// ChannelDirectoryEvidence is kept as the package-level compatibility name for
// shared user-safe degraded-mode evidence.
type ChannelDirectoryEvidence = model.Evidence

// ChannelDirectoryStore persists channel_directory.json under a caller-owned
// Gormes state/home root. Tests pass temp roots; no live operator home is read.
type ChannelDirectoryStore struct {
	root string
}

// NewChannelDirectoryStore returns a store rooted at root.
func NewChannelDirectoryStore(root string) ChannelDirectoryStore {
	return ChannelDirectoryStore{root: strings.TrimSpace(root)}
}

// Root returns the store root for fixture setup.
func (s ChannelDirectoryStore) Root() string { return s.root }

func (s ChannelDirectoryStore) path() string {
	return filepath.Join(s.root, channelDirectoryFileName)
}

// Load reads the directory. Missing or invalid files return empty directories
// plus structured degraded evidence.
func (s ChannelDirectoryStore) Load() (ChannelDirectory, ChannelDirectoryEvidence) {
	var dir ChannelDirectory
	if err := storage.ReadJSON(s.path(), &dir); errors.Is(err, os.ErrNotExist) {
		return emptyChannelDirectory(), ChannelDirectoryEvidence{Code: "channel_directory_missing"}
	} else if err != nil {
		return emptyChannelDirectory(), ChannelDirectoryEvidence{Code: "channel_directory_invalid"}
	}
	if dir.Platforms == nil {
		dir.Platforms = map[string][]ChannelDirectoryEntry{}
	}
	return dir, ChannelDirectoryEvidence{}
}

// Save atomically writes the directory.
func (s ChannelDirectoryStore) Save(dir ChannelDirectory) error {
	return s.SaveWithWriter(dir, os.WriteFile)
}

// SaveWithWriter exists for deterministic atomic-write failure tests. It writes
// a temp file, then renames only after the writer succeeds, so old complete JSON
// remains visible after an injected partial-write failure.
func (s ChannelDirectoryStore) SaveWithWriter(dir ChannelDirectory, writer func(string, []byte, os.FileMode) error) error {
	if strings.TrimSpace(s.root) == "" {
		return fmt.Errorf("channel directory root is empty")
	}
	if writer == nil {
		writer = os.WriteFile
	}
	if dir.Platforms == nil {
		dir.Platforms = map[string][]ChannelDirectoryEntry{}
	}
	return storage.WriteAtomicJSON(s.root, channelDirectoryFileName, ".channel_directory-*.tmp", dir, storage.Writer(writer))
}

func emptyChannelDirectory() ChannelDirectory {
	return ChannelDirectory{Platforms: map[string][]ChannelDirectoryEntry{}}
}

// Resolve resolves a human-friendly channel target for platform into a concrete
// gatewaydelivery.Target. It follows Hermes matching order: exact ID, exact display/name,
// Discord guild-qualified name, then unambiguous prefix.
func (d ChannelDirectory) Resolve(platform, query string) (gatewaydelivery.Target, ChannelDirectoryEvidence) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	raw := strings.TrimSpace(query)
	entries := d.Platforms[platform]
	if platform == "" || raw == "" || len(entries) == 0 {
		return gatewaydelivery.Target{}, ChannelDirectoryEvidence{Code: "channel_directory_missing", Platform: platform, Query: raw}
	}
	for _, entry := range entries {
		if entry.ID == raw {
			return model.DeliveryTarget(platform, entry), ChannelDirectoryEvidence{}
		}
	}
	normalized := model.NormalizeQuery(raw)
	for _, entry := range entries {
		if model.NormalizeQuery(entry.Name) == normalized || model.NormalizeQuery(channelTargetName(platform, entry)) == normalized {
			return model.DeliveryTarget(platform, entry), ChannelDirectoryEvidence{}
		}
	}
	if strings.Contains(normalized, "/") {
		parts := strings.Split(normalized, "/")
		guildPart := strings.Join(parts[:len(parts)-1], "/")
		channelPart := parts[len(parts)-1]
		for _, entry := range entries {
			if strings.ToLower(strings.TrimSpace(entry.Guild)) == guildPart && model.NormalizeQuery(entry.Name) == channelPart {
				return model.DeliveryTarget(platform, entry), ChannelDirectoryEvidence{}
			}
		}
	}
	matches := make([]ChannelDirectoryEntry, 0, 1)
	for _, entry := range entries {
		if strings.HasPrefix(model.NormalizeQuery(entry.Name), normalized) {
			matches = append(matches, entry)
		}
	}
	switch len(matches) {
	case 1:
		return model.DeliveryTarget(platform, matches[0]), ChannelDirectoryEvidence{}
	case 0:
		return gatewaydelivery.Target{}, ChannelDirectoryEvidence{Code: "channel_directory_missing", Platform: platform, Query: raw}
	default:
		return gatewaydelivery.Target{}, ChannelDirectoryEvidence{Code: "channel_target_ambiguous", Platform: platform, Query: raw}
	}
}

// ValidateDeliveryTarget returns channel_target_stale when an explicit target
// no longer appears in a refreshed platform directory. Platform-only, local,
// origin, and unknown-directory targets are left to existing home-channel and
// missing-directory resolution paths.
func (d ChannelDirectory) ValidateDeliveryTarget(target gatewaydelivery.Target) (gatewaydelivery.Target, ChannelDirectoryEvidence) {
	platform := strings.ToLower(strings.TrimSpace(target.Platform))
	if target.IsOrigin || !target.IsExplicit || platform == "" || strings.EqualFold(platform, "local") || strings.TrimSpace(target.ChatID) == "" {
		return target, ChannelDirectoryEvidence{}
	}
	entries, ok := d.Platforms[platform]
	if !ok || len(entries) == 0 {
		return target, ChannelDirectoryEvidence{}
	}
	for _, entry := range entries {
		candidate := model.DeliveryTarget(platform, entry)
		if candidate.ChatID == strings.TrimSpace(target.ChatID) && candidate.ThreadID == strings.TrimSpace(target.ThreadID) {
			return target, ChannelDirectoryEvidence{}
		}
	}
	return gatewaydelivery.Target{}, ChannelDirectoryEvidence{Code: "channel_target_stale", Platform: platform, Query: target.String()}
}

// LookupType returns the cached channel type for a platform target ID.
func (d ChannelDirectory) LookupType(platform, id string) string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	id = strings.TrimSpace(id)
	for _, entry := range d.Platforms[platform] {
		if entry.ID == id {
			return strings.TrimSpace(entry.Type)
		}
	}
	return ""
}

// FormatForDisplay returns a Hermes-style target list for model/tool guidance.
func (d ChannelDirectory) FormatForDisplay() string {
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
		entries := append([]ChannelDirectoryEntry(nil), d.Platforms[platform]...)
		if platform == "discord" {
			guilds := map[string][]ChannelDirectoryEntry{}
			dms := []ChannelDirectoryEntry{}
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
				sort.Slice(guilds[guild], func(i, j int) bool { return guilds[guild][i].Name < guilds[guild][j].Name })
				for _, entry := range guilds[guild] {
					lines = append(lines, "  discord:"+channelTargetName(platform, entry))
				}
			}
			if len(dms) > 0 {
				lines = append(lines, "Discord (DMs):")
				for _, entry := range dms {
					lines = append(lines, "  discord:"+channelTargetName(platform, entry))
				}
			}
			lines = append(lines, "")
			continue
		}
		lines = append(lines, strings.Title(platform)+":")
		for _, entry := range entries {
			lines = append(lines, "  "+platform+":"+channelTargetName(platform, entry))
		}
		lines = append(lines, "")
	}
	lines = append(lines, `Use these as the "target" parameter when sending.`)
	lines = append(lines, `Bare platform name (e.g. "telegram") sends to home channel.`)
	return strings.Join(lines, "\n")
}

func (d ChannelDirectory) hasEntries() bool {
	for _, entries := range d.Platforms {
		if len(entries) > 0 {
			return true
		}
	}
	return false
}

func channelTargetName(platform string, entry ChannelDirectoryEntry) string {
	name := strings.TrimSpace(entry.Name)
	if platform == "discord" && strings.TrimSpace(entry.Guild) != "" {
		return "#" + name
	}
	if platform != "discord" && strings.TrimSpace(entry.Type) != "" {
		return name + " (" + strings.TrimSpace(entry.Type) + ")"
	}
	return name
}
