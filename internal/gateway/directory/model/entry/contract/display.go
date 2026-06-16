package contract

import "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/policy"

// TargetDisplayName returns the user-facing selector text for a directory entry.
// It is shared by target resolution and model/tool display rendering so both
// surfaces keep the same platform-specific naming policy.
func TargetDisplayName(platform string, item Entry) string {
	platform = NormalizePlatform(platform)
	name := policy.TrimText(item.Name)
	if platform == "discord" && EntryGuild(item) != "" {
		return "#" + name
	}
	if entryType := policy.TrimText(item.Type); platform != "discord" && entryType != "" {
		return name + " (" + entryType + ")"
	}
	return name
}
