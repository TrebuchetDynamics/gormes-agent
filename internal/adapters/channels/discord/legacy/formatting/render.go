package formatting

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

const maxDiscordText = 2000

func FormatStream(f kernel.RenderFrame) string {
	return channelutil.FormatRenderStream(f, maxDiscordText, "⏳")
}

func FormatFinal(f kernel.RenderFrame) string {
	return channelutil.FormatRenderFinal(f, maxDiscordText, "(empty reply)")
}

func FormatError(f kernel.RenderFrame) string {
	return channelutil.FormatRenderError(f, maxDiscordText, false)
}

func TruncateDiscord(s string) string {
	return channelutil.TruncateRunesWithSuffix(s, maxDiscordText, "…")
}

func StripSelfMention(text, selfID string) string {
	if selfID == "" {
		return strings.TrimSpace(text)
	}
	replacer := strings.NewReplacer("<@"+selfID+">", "", "<@!"+selfID+">", "")
	return strings.TrimSpace(replacer.Replace(text))
}

func HasSelfMention(text, selfID string) bool {
	if selfID == "" {
		return false
	}
	return strings.Contains(text, "<@"+selfID+">") || strings.Contains(text, "<@!"+selfID+">")
}
