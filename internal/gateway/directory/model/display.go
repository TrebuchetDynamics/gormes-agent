package model

import "strings"

// TargetDisplayName returns the user-facing selector text for a directory entry.
// It is shared by target resolution and model/tool display rendering so both
// surfaces keep the same platform-specific naming policy.
func TargetDisplayName(platform string, entry Entry) string {
	platform = NormalizePlatform(platform)
	name := strings.TrimSpace(entry.Name)
	if platform == "discord" && strings.TrimSpace(entry.Guild) != "" {
		return "#" + name
	}
	if platform != "discord" && strings.TrimSpace(entry.Type) != "" {
		return name + " (" + strings.TrimSpace(entry.Type) + ")"
	}
	return name
}
