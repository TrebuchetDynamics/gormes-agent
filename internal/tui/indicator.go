package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/indicator"

type IndicatorStyle = indicator.Style

const (
	IndicatorStyleASCII   IndicatorStyle = indicator.StyleASCII
	IndicatorStyleEmoji   IndicatorStyle = indicator.StyleEmoji
	IndicatorStyleKaomoji IndicatorStyle = indicator.StyleKaomoji
	IndicatorStyleUnicode IndicatorStyle = indicator.StyleUnicode
)

func NormalizeIndicatorStyle(raw string) IndicatorStyle {
	return indicator.NormalizeStyle(raw)
}

func RenderIndicatorFrame(style IndicatorStyle, frameIndex int) string {
	return indicator.RenderFrame(style, frameIndex)
}
