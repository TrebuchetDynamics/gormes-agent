package model

// TargetDisplayName returns the user-facing selector text for a directory entry.
// It is shared by target resolution and model/tool display rendering so both
// surfaces keep the same platform-specific naming policy.
func TargetDisplayName(platform string, entry Entry) string {
	platform = NormalizePlatform(platform)
	name := trimText(entry.Name)
	if platform == "discord" && EntryGuild(entry) != "" {
		return "#" + name
	}
	if entryType := trimText(entry.Type); platform != "discord" && entryType != "" {
		return name + " (" + entryType + ")"
	}
	return name
}
