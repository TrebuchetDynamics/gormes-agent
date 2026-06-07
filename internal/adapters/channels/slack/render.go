package slack

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/slack/rendering"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

const maxSlackText = rendering.MaxSlackText

func formatPending() string {
	return rendering.FormatPending()
}

func formatStream(f kernel.RenderFrame) string {
	return rendering.FormatStream(f)
}

func formatFinal(f kernel.RenderFrame) string {
	return rendering.FormatFinal(f)
}

func formatError(f kernel.RenderFrame) string {
	return rendering.FormatError(f)
}

func truncateSlack(s string) string {
	return rendering.TruncateSlack(s)
}
