package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/discord"

const (
	DiscordToolName      = discord.DiscordToolName
	DiscordAdminToolName = discord.DiscordAdminToolName

	DiscordToolsetCore  = discord.DiscordToolsetCore
	DiscordToolsetAdmin = discord.DiscordToolsetAdmin

	DiscordToolStatusAvailable                = discord.DiscordToolStatusAvailable
	DiscordToolStatusDisabled                 = discord.DiscordToolStatusDisabled
	DiscordToolStatusMissingToken             = discord.DiscordToolStatusMissingToken
	DiscordToolStatusUnavailablePlatformScope = discord.DiscordToolStatusUnavailablePlatformScope
	DiscordToolStatusNoActions                = discord.DiscordToolStatusNoActions
)

type DiscordToolStatus = discord.DiscordToolStatus
type DiscordApplicationCapabilities = discord.DiscordApplicationCapabilities
type DiscordToolsetOptions = discord.DiscordToolsetOptions
type DiscordToolsetStatus = discord.DiscordToolsetStatus

func DiscordToolsetAllowedForPlatform(toolset, platform string) bool {
	return discord.DiscordToolsetAllowedForPlatform(toolset, platform)
}

func DiscordToolsetDescriptors(opts DiscordToolsetOptions) []ToolDescriptor {
	return discord.DiscordToolsetDescriptors(opts)
}

func DiscordToolsetStatuses(opts DiscordToolsetOptions) []DiscordToolsetStatus {
	return discord.DiscordToolsetStatuses(opts)
}
