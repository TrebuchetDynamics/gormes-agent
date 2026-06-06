package indicator

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/indicator/slash"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/indicator/style"
)

type Style = style.Style

const (
	StyleASCII   Style = style.ASCII
	StyleEmoji   Style = style.Emoji
	StyleKaomoji Style = style.Kaomoji
	StyleUnicode Style = style.Unicode
)

const SlashUsage = slash.Usage

type SlashResult = slash.Result

func NormalizeStyle(raw string) Style {
	return style.Normalize(raw)
}

func RenderFrame(indicatorStyle Style, frameIndex int) string {
	return style.RenderFrame(indicatorStyle, frameIndex)
}

func Frames(indicatorStyle Style) []string {
	return style.Frames(indicatorStyle)
}

// ParseSlash resolves /indicator input into display evidence and an optional
// style mutation. Invalid invocations return SlashUsage and Apply=false.
func ParseSlash(input string, current Style) SlashResult {
	return slash.Parse(input, current)
}
