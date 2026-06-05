package model

import entrymodel "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/entry"

// TargetDisplayName returns the user-facing selector text for a directory entry.
// It is shared by target resolution and model/tool display rendering so both
// surfaces keep the same platform-specific naming policy.
func TargetDisplayName(platform string, entry Entry) string {
	return entrymodel.TargetDisplayName(platform, entry)
}
