package gateway

import gatewayadmission "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/admission"

// WhitelistConfig represents a per-platform chat/channel/room whitelist.
// When Enabled is false, all chats are allowed (default-open behavior).
// When Enabled is true, only chats with IDs in the IDs slice are permitted.
type WhitelistConfig = gatewayadmission.WhitelistConfig

// WhitelistStatus carries runtime evidence for gateway status and doctor output.
type WhitelistStatus = gatewayadmission.WhitelistStatus

// ParseWhitelistConfig parses a list of allowed IDs into a WhitelistConfig.
// Empty, whitespace-only, and duplicate entries are silently removed.
// An empty or nil input produces a disabled whitelist (Enabled=false).
func ParseWhitelistConfig(ids []string) WhitelistConfig {
	return gatewayadmission.ParseWhitelistConfig(ids)
}
