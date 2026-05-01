package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
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

// ChannelDirectoryEntry describes one platform target that can be selected by
// exact ID, human display name, guild-qualified name, or type/display lookup.
type ChannelDirectoryEntry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type,omitempty"`
	Guild     string `json:"guild,omitempty"`
	ChatID    string `json:"chat_id,omitempty"`
	ThreadID  string `json:"thread_id,omitempty"`
	ChatTopic string `json:"chat_topic,omitempty"`
}

// ChannelDirectoryEvidence carries user-safe degraded-mode evidence without
// leaking local state paths.
type ChannelDirectoryEvidence struct {
	Code     string
	Platform string
	Query    string
}

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
	body, err := os.ReadFile(s.path())
	if errors.Is(err, os.ErrNotExist) {
		return emptyChannelDirectory(), ChannelDirectoryEvidence{Code: "channel_directory_missing"}
	}
	if err != nil {
		return emptyChannelDirectory(), ChannelDirectoryEvidence{Code: "channel_directory_invalid"}
	}
	var dir ChannelDirectory
	if err := json.Unmarshal(body, &dir); err != nil {
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
	body, err := json.MarshalIndent(dir, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.root, ".channel_directory-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := writer(tmpPath, body, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, s.path()); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func emptyChannelDirectory() ChannelDirectory {
	return ChannelDirectory{Platforms: map[string][]ChannelDirectoryEntry{}}
}

// Resolve resolves a human-friendly channel target for platform into a concrete
// DeliveryTarget. It follows Hermes matching order: exact ID, exact display/name,
// Discord guild-qualified name, then unambiguous prefix.
func (d ChannelDirectory) Resolve(platform, query string) (DeliveryTarget, ChannelDirectoryEvidence) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	raw := strings.TrimSpace(query)
	entries := d.Platforms[platform]
	if platform == "" || raw == "" || len(entries) == 0 {
		return DeliveryTarget{}, ChannelDirectoryEvidence{Code: "channel_directory_missing", Platform: platform, Query: raw}
	}
	for _, entry := range entries {
		if entry.ID == raw {
			return entry.deliveryTarget(platform), ChannelDirectoryEvidence{}
		}
	}
	normalized := normalizeChannelQuery(raw)
	for _, entry := range entries {
		if normalizeChannelQuery(entry.Name) == normalized || normalizeChannelQuery(channelTargetName(platform, entry)) == normalized {
			return entry.deliveryTarget(platform), ChannelDirectoryEvidence{}
		}
	}
	if strings.Contains(normalized, "/") {
		parts := strings.Split(normalized, "/")
		guildPart := strings.Join(parts[:len(parts)-1], "/")
		channelPart := parts[len(parts)-1]
		for _, entry := range entries {
			if strings.ToLower(strings.TrimSpace(entry.Guild)) == guildPart && normalizeChannelQuery(entry.Name) == channelPart {
				return entry.deliveryTarget(platform), ChannelDirectoryEvidence{}
			}
		}
	}
	matches := make([]ChannelDirectoryEntry, 0, 1)
	for _, entry := range entries {
		if strings.HasPrefix(normalizeChannelQuery(entry.Name), normalized) {
			matches = append(matches, entry)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0].deliveryTarget(platform), ChannelDirectoryEvidence{}
	case 0:
		return DeliveryTarget{}, ChannelDirectoryEvidence{Code: "channel_directory_missing", Platform: platform, Query: raw}
	default:
		return DeliveryTarget{}, ChannelDirectoryEvidence{Code: "channel_target_ambiguous", Platform: platform, Query: raw}
	}
}

// ValidateDeliveryTarget returns channel_target_stale when an explicit target
// no longer appears in a refreshed platform directory. Platform-only, local,
// origin, and unknown-directory targets are left to existing home-channel and
// missing-directory resolution paths.
func (d ChannelDirectory) ValidateDeliveryTarget(target DeliveryTarget) (DeliveryTarget, ChannelDirectoryEvidence) {
	platform := strings.ToLower(strings.TrimSpace(target.Platform))
	if target.IsOrigin || !target.IsExplicit || platform == "" || strings.EqualFold(platform, "local") || strings.TrimSpace(target.ChatID) == "" {
		return target, ChannelDirectoryEvidence{}
	}
	entries, ok := d.Platforms[platform]
	if !ok || len(entries) == 0 {
		return target, ChannelDirectoryEvidence{}
	}
	for _, entry := range entries {
		candidate := entry.deliveryTarget(platform)
		if candidate.ChatID == strings.TrimSpace(target.ChatID) && candidate.ThreadID == strings.TrimSpace(target.ThreadID) {
			return target, ChannelDirectoryEvidence{}
		}
	}
	return DeliveryTarget{}, ChannelDirectoryEvidence{Code: "channel_target_stale", Platform: platform, Query: target.String()}
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

func (e ChannelDirectoryEntry) deliveryTarget(platform string) DeliveryTarget {
	chatID := strings.TrimSpace(e.ChatID)
	threadID := strings.TrimSpace(e.ThreadID)
	if chatID == "" {
		parts := strings.SplitN(strings.TrimSpace(e.ID), ":", 2)
		chatID = parts[0]
		if len(parts) == 2 && threadID == "" {
			threadID = parts[1]
		}
	}
	return DeliveryTarget{Platform: platform, ChatID: chatID, ThreadID: threadID, IsExplicit: true}
}

func normalizeChannelQuery(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimLeft(value, "#")))
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
