package discord

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/discord/toolsets"

const (
	DiscordToolName      = toolsets.DiscordToolName
	DiscordAdminToolName = toolsets.DiscordAdminToolName

	DiscordToolsetCore  = toolsets.DiscordToolsetCore
	DiscordToolsetAdmin = toolsets.DiscordToolsetAdmin
)

type ToolDescriptor = toolsets.ToolDescriptor

type DiscordToolStatus = toolsets.DiscordToolStatus

const (
	DiscordToolStatusAvailable                = toolsets.DiscordToolStatusAvailable
	DiscordToolStatusDisabled                 = toolsets.DiscordToolStatusDisabled
	DiscordToolStatusMissingToken             = toolsets.DiscordToolStatusMissingToken
	DiscordToolStatusUnavailablePlatformScope = toolsets.DiscordToolStatusUnavailablePlatformScope
	DiscordToolStatusNoActions                = toolsets.DiscordToolStatusNoActions
)

type DiscordApplicationCapabilities = toolsets.DiscordApplicationCapabilities

type DiscordToolsetOptions = toolsets.DiscordToolsetOptions

type DiscordToolsetStatus = toolsets.DiscordToolsetStatus

// DiscordToolsetAllowedForPlatform reports whether a Discord toolset may be
// configured for a platform. The toolsets are still default-off on allowed
// platforms and must be explicitly enabled by the caller.
func DiscordToolsetAllowedForPlatform(toolset, platform string) bool {
	return toolsets.DiscordToolsetAllowedForPlatform(toolset, platform)
}

// DiscordToolsetDescriptors returns the model-visible Discord descriptors for
// the supplied platform/toolset/capability snapshot.
func DiscordToolsetDescriptors(opts DiscordToolsetOptions) []ToolDescriptor {
	return toolsets.DiscordToolsetDescriptors(opts)
}

// DiscordToolsetStatuses returns descriptor availability for both split
// Discord toolsets in deterministic descriptor order.
func DiscordToolsetStatuses(opts DiscordToolsetOptions) []DiscordToolsetStatus {
	return toolsets.DiscordToolsetStatuses(opts)
}
