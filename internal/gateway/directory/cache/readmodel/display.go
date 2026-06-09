package readmodel

import (
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
)

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
		model.SortEntriesByNameID(entries)
		if platform == "discord" {
			guilds := map[string][]model.Entry{}
			dms := []model.Entry{}
			for _, entry := range entries {
				if guild := model.EntryGuild(entry); guild != "" {
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
				lines = append(lines, "Discord ("+displayDirectoryText(guild)+"):")
				model.SortEntriesByNameID(guilds[guild])
				for _, entry := range guilds[guild] {
					lines = append(lines, "  discord:"+displayDirectoryText(model.TargetDisplayName(platform, entry)))
				}
			}
			if len(dms) > 0 {
				lines = append(lines, "Discord (DMs):")
				model.SortEntriesByNameID(dms)
				for _, entry := range dms {
					lines = append(lines, "  discord:"+displayDirectoryText(model.TargetDisplayName(platform, entry)))
				}
			}
			lines = append(lines, "")
			continue
		}
		lines = append(lines, strings.Title(platform)+":")
		for _, entry := range entries {
			lines = append(lines, "  "+platform+":"+displayDirectoryText(model.TargetDisplayName(platform, entry)))
		}
		lines = append(lines, "")
	}
	lines = append(lines, `Use these as the "target" parameter when sending.`)
	lines = append(lines, `Bare platform name (e.g. "telegram") sends to home channel.`)
	return strings.Join(lines, "\n")
}

func displayDirectoryText(value string) string {
	value = strings.ReplaceAll(value, "`", "'")
	return strings.Join(strings.Fields(value), " ")
}

func (d Directory) hasEntries() bool {
	for _, entries := range d.Platforms {
		if len(entries) > 0 {
			return true
		}
	}
	return false
}
