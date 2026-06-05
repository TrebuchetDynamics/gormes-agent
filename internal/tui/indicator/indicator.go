package indicator

import "strings"

type Style string

const (
	StyleASCII   Style = "ascii"
	StyleEmoji   Style = "emoji"
	StyleKaomoji Style = "kaomoji"
	StyleUnicode Style = "unicode"
)

var emojiFrames = []string{"⚕ ", "🌀", "🤔", "✨", "🍵", "🔮"}
var asciiFrames = []string{"|", "/", "-", "\\"}
var unicodeFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
var kaomojiFrames = []string{
	"(｡◕‿◕｡)", "(◕‿◕✿)", "٩(◕‿◕｡)۶", "(✿◠‿◠)", "( ˘▽˘)っ",
	"♪(´ε` )", "(◕ᴗ◕✿)", "ヾ(＾∇＾)", "(≧◡≦)", "(★ω★)",
}

func NormalizeStyle(raw string) Style {
	switch Style(strings.ToLower(strings.TrimSpace(raw))) {
	case StyleASCII:
		return StyleASCII
	case StyleEmoji:
		return StyleEmoji
	case StyleUnicode:
		return StyleUnicode
	case StyleKaomoji:
		return StyleKaomoji
	default:
		return StyleKaomoji
	}
}

func RenderFrame(style Style, frameIndex int) string {
	frames := Frames(style)
	if len(frames) == 0 {
		return ""
	}
	if frameIndex < 0 {
		frameIndex = 0
	}
	return frames[frameIndex%len(frames)]
}

func Frames(style Style) []string {
	switch NormalizeStyle(string(style)) {
	case StyleASCII:
		return asciiFrames
	case StyleEmoji:
		return emojiFrames
	case StyleUnicode:
		return unicodeFrames
	default:
		return kaomojiFrames
	}
}
