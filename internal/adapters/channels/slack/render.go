package slack

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

const maxSlackText = 40000

func formatPending() string {
	return "⏳"
}

func formatStream(f kernel.RenderFrame) string {
	return channelutil.FormatRenderStream(f, maxSlackText, formatPending())
}

func formatFinal(f kernel.RenderFrame) string {
	return channelutil.FormatRenderFinal(f, maxSlackText, "(empty reply)")
}

func formatError(f kernel.RenderFrame) string {
	return channelutil.FormatRenderError(f, maxSlackText, true)
}

func truncateSlack(s string) string {
	return channelutil.TruncateRunesWithSuffix(s, maxSlackText, "…")
}
