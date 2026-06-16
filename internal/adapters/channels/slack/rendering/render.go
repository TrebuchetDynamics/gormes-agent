package rendering

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

const MaxSlackText = 40000

func FormatPending() string {
	return "⏳"
}

func FormatStream(f kernel.RenderFrame) string {
	return channelutil.FormatRenderStream(f, MaxSlackText, FormatPending())
}

func FormatFinal(f kernel.RenderFrame) string {
	return channelutil.FormatRenderFinal(f, MaxSlackText, "(empty reply)")
}

func FormatError(f kernel.RenderFrame) string {
	return channelutil.FormatRenderError(f, MaxSlackText, true)
}

func TruncateSlack(s string) string {
	return channelutil.TruncateRunesWithSuffix(s, MaxSlackText, "…")
}
