package discord

import "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/discord/messaging"

func normalizeReplyToMode(mode string) string {
	return messaging.NormalizeReplyToMode(mode)
}

func isMissingDiscordReplyReference(err error) bool {
	return messaging.IsMissingDiscordReplyReference(err)
}
