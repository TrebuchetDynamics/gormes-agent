package tui

import "strings"

type IndicatorStyle string

const (
	IndicatorStyleASCII   IndicatorStyle = "ascii"
	IndicatorStyleEmoji   IndicatorStyle = "emoji"
	IndicatorStyleKaomoji IndicatorStyle = "kaomoji"
	IndicatorStyleUnicode IndicatorStyle = "unicode"
)

var indicatorSubcommands = []string{"ascii", "emoji", "kaomoji", "unicode"}

var indicatorEmojiFrames = []string{"⚕ ", "🌀", "🤔", "✨", "🍵", "🔮"}
var indicatorASCIIFrames = []string{"|", "/", "-", "\\"}
var indicatorUnicodeFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func NormalizeIndicatorStyle(raw string) IndicatorStyle {
	switch IndicatorStyle(strings.ToLower(strings.TrimSpace(raw))) {
	case IndicatorStyleASCII:
		return IndicatorStyleASCII
	case IndicatorStyleEmoji:
		return IndicatorStyleEmoji
	case IndicatorStyleUnicode:
		return IndicatorStyleUnicode
	case IndicatorStyleKaomoji:
		return IndicatorStyleKaomoji
	default:
		return IndicatorStyleKaomoji
	}
}

func RenderIndicatorFrame(style IndicatorStyle, frameIndex int) string {
	frames := indicatorFrames(style)
	if len(frames) == 0 {
		return ""
	}
	if frameIndex < 0 {
		frameIndex = 0
	}
	return frames[frameIndex%len(frames)]
}

func indicatorFrames(style IndicatorStyle) []string {
	switch NormalizeIndicatorStyle(string(style)) {
	case IndicatorStyleASCII:
		return indicatorASCIIFrames
	case IndicatorStyleEmoji:
		return indicatorEmojiFrames
	case IndicatorStyleUnicode:
		return indicatorUnicodeFrames
	default:
		return kawaiiWaitingFaces
	}
}
