package admission

import admissionwhitelist "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/admission/whitelist"

// WhitelistConfig represents a per-platform chat/channel/room whitelist.
// When Enabled is false, all chats are allowed (default-open behavior).
// When Enabled is true, only chats with IDs in the IDs slice are permitted.
type WhitelistConfig = admissionwhitelist.WhitelistConfig

// WhitelistStatus carries runtime evidence for gateway status and doctor output.
type WhitelistStatus = admissionwhitelist.WhitelistStatus

// ParseWhitelistConfig parses a list of allowed IDs into a WhitelistConfig.
// Empty, whitespace-only, and duplicate entries are silently removed.
// An empty or nil input produces a disabled whitelist (Enabled=false).
func ParseWhitelistConfig(ids []string) WhitelistConfig {
	return admissionwhitelist.ParseWhitelistConfig(ids)
}
