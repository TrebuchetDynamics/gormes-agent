package style

import "strings"

type Style string

const (
	ASCII   Style = "ascii"
	Emoji   Style = "emoji"
	Kaomoji Style = "kaomoji"
	Unicode Style = "unicode"
)

var emojiFrames = []string{"⚕ ", "🌀", "🤔", "✨", "🍵", "🔮"}
var asciiFrames = []string{"|", "/", "-", "\\"}
var unicodeFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
var kaomojiFrames = []string{
	"(｡◕‿◕｡)", "(◕‿◕✿)", "٩(◕‿◕｡)۶", "(✿◠‿◠)", "( ˘▽˘)っ",
	"♪(´ε` )", "(◕ᴗ◕✿)", "ヾ(＾∇＾)", "(≧◡≦)", "(★ω★)",
}

func Normalize(raw string) Style {
	switch Style(strings.ToLower(strings.TrimSpace(raw))) {
	case ASCII:
		return ASCII
	case Emoji:
		return Emoji
	case Unicode:
		return Unicode
	case Kaomoji:
		return Kaomoji
	default:
		return Unicode
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
	switch Normalize(string(style)) {
	case ASCII:
		return asciiFrames
	case Emoji:
		return emojiFrames
	case Unicode:
		return unicodeFrames
	default:
		return kaomojiFrames
	}
}
