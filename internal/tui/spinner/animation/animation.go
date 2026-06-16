package animation

type Kind string

const (
	Dots    Kind = "dots"
	Bounce  Kind = "bounce"
	Grow    Kind = "grow"
	Arrows  Kind = "arrows"
	Star    Kind = "star"
	Moon    Kind = "moon"
	Pulse   Kind = "pulse"
	Brain   Kind = "brain"
	Sparkle Kind = "sparkle"
	Emoji   Kind = "emoji"
	Kaomoji Kind = "kaomoji"
)

var frames = map[Kind][]string{
	Dots:    {"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	Bounce:  {"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"},
	Grow:    {"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂"},
	Arrows:  {"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"},
	Star:    {"✶", "✷", "✸", "✹", "✺", "✹", "✸", "✷"},
	Moon:    {"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"},
	Pulse:   {"◜", "◠", "◝", "◞", "◡", "◟"},
	Brain:   {"🧠", "💭", "💡", "✨", "💫", "🌟", "💡", "💭"},
	Sparkle: {"⁺", "˚", "*", "✧", "✦", "✧", "*", "˚"},
	Emoji:   {"⚕", "⚌", "🤔", "✨", "🍵", "🔮"},
	Kaomoji: {"(｡•́︿•̀｡)", "(◕‿◕✿)", "(≧◡≦)", "٩(◕‿◕｡)۶", "(★ω★)"},
}

var kawaiiWaitingFaces = []string{
	"(｡◕‿◕｡)", "(◕‿◕✿)", "٩(◕‿◕｡)۶", "(✿◠‿◠)", "( ˘▽˘)っ",
	"♪(´ε` )", "(◕ᴗ◕✿)", "ヾ(＾∇＾)", "(≧◡≦)", "(★ω★)",
}

var kawaiiThinkingFaces = []string{
	"(｡•́︿•̀｡)", "(◔_◔)", "(¬‿¬)", "( •_•)>⌐■-■", "(⌐■_■)",
	"(´･_･`)", "◉_◉", "(°ロ°)", "( ˘⌣˘)♡", "ヽ(>∀<☆)☆",
	"٩(๑❛ᴗ❛๑)۶", "(⊙_⊙)", "(¬_¬)", "( ͡° ͜ʖ ͡°)", "ಠ_ಠ",
}

var thinkingVerbs = []string{
	"pondering", "contemplating", "musing", "cogitating", "ruminating",
	"deliberating", "mulling", "reflecting", "processing", "reasoning",
	"analyzing", "computing", "synthesizing", "formulating", "brainstorming",
}

type Wing struct{ Left, Right string }

func Frames(kind Kind) []string {
	if f, ok := frames[kind]; ok {
		return f
	}
	return frames[Dots]
}

func WaitingFace(idx int) string {
	return kawaiiWaitingFaces[idx%len(kawaiiWaitingFaces)]
}

func ThinkingFace(idx int) string {
	return kawaiiThinkingFaces[idx%len(kawaiiThinkingFaces)]
}

func ThinkingVerb(idx int) string {
	return thinkingVerbs[idx%len(thinkingVerbs)]
}

func WingsForSkin(skinName string) []Wing {
	switch skinName {
	case "ares":
		return []Wing{
			{"⟪⚔", "⚔⟫"}, {"⟪▲", "▲⟫"}, {"⟪╸", "╺⟫"}, {"⟪⛨", "⛨⟫"},
		}
	case "poseidon":
		return []Wing{
			{"⟪≈", "≈⟫"}, {"⟪Ψ", "Ψ⟫"}, {"⟪∿", "∿⟫"}, {"⟪◌", "◌⟫"},
		}
	case "sisyphus":
		return []Wing{
			{"⟪◉", "◉⟫"}, {"⟪◬", "◬⟫"}, {"⟪◌", "◌⟫"}, {"⟪⬤", "⬤⟫"},
		}
	case "charizard":
		return []Wing{
			{"⟪✦", "✦⟫"}, {"⟪▲", "▲⟫"}, {"⟪◌", "◌⟫"}, {"⟪◇", "◇⟫"},
		}
	default:
		return nil
	}
}

func Render(kind Kind, tick int, faceIdx int, verbIdx int, wingIdx int, skinName string, elapsed string) string {
	frames := Frames(kind)
	frame := frames[tick%len(frames)]
	face := ThinkingFace(faceIdx)
	verb := ThinkingVerb(verbIdx)
	wings := WingsForSkin(skinName)

	if len(wings) > 0 && wingIdx < len(wings) {
		w := wings[wingIdx%len(wings)]
		return "  " + w.Left + " " + frame + " " + verb + " " + face + " " + w.Right + " (" + elapsed + "s)"
	}
	return "  " + frame + " " + verb + " " + face + " (" + elapsed + "s)"
}
