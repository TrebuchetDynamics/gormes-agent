package tui

type SpinnerKind string

const (
	SpinnerDots    SpinnerKind = "dots"
	SpinnerBounce  SpinnerKind = "bounce"
	SpinnerGrow    SpinnerKind = "grow"
	SpinnerArrows  SpinnerKind = "arrows"
	SpinnerStar    SpinnerKind = "star"
	SpinnerMoon    SpinnerKind = "moon"
	SpinnerPulse   SpinnerKind = "pulse"
	SpinnerBrain   SpinnerKind = "brain"
	SpinnerSparkle SpinnerKind = "sparkle"
	SpinnerEmoji   SpinnerKind = "emoji"
	SpinnerKaomoji SpinnerKind = "kaomoji"
)

var spinnerFrames = map[SpinnerKind][]string{
	SpinnerDots:    {"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	SpinnerBounce:  {"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"},
	SpinnerGrow:    {"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂"},
	SpinnerArrows:  {"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"},
	SpinnerStar:    {"✶", "✷", "✸", "✹", "✺", "✹", "✸", "✷"},
	SpinnerMoon:    {"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"},
	SpinnerPulse:   {"◜", "◠", "◝", "◞", "◡", "◟"},
	SpinnerBrain:   {"🧠", "💭", "💡", "✨", "💫", "🌟", "💡", "💭"},
	SpinnerSparkle: {"⁺", "˚", "*", "✧", "✦", "✧", "*", "˚"},
	SpinnerEmoji:   {"⚕", "⚌", "🤔", "✨", "🍵", "🔮"},
	SpinnerKaomoji: {"(｡•́︿•̀｡)", "(◕‿◕✿)", "(≧◡≦)", "٩(◕‿◕｡)۶", "(★ω★)"},
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

type SpinnerWing struct{ Left, Right string }

func SpinnerFrames(kind SpinnerKind) []string {
	if f, ok := spinnerFrames[kind]; ok {
		return f
	}
	return spinnerFrames[SpinnerDots]
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

func SpinnerWingsForSkin(skinName string) []SpinnerWing {
	switch skinName {
	case "ares":
		return []SpinnerWing{
			{"⟪⚔", "⚔⟫"}, {"⟪▲", "▲⟫"}, {"⟪╸", "╺⟫"}, {"⟪⛨", "⛨⟫"},
		}
	case "poseidon":
		return []SpinnerWing{
			{"⟪≈", "≈⟫"}, {"⟪Ψ", "Ψ⟫"}, {"⟪∿", "∿⟫"}, {"⟪◌", "◌⟫"},
		}
	case "sisyphus":
		return []SpinnerWing{
			{"⟪◉", "◉⟫"}, {"⟪◬", "◬⟫"}, {"⟪◌", "◌⟫"}, {"⟪⬤", "⬤⟫"},
		}
	case "charizard":
		return []SpinnerWing{
			{"⟪✦", "✦⟫"}, {"⟪▲", "▲⟫"}, {"⟪◌", "◌⟫"}, {"⟪◇", "◇⟫"},
		}
	default:
		return nil
	}
}

func SpinnerRender(kind SpinnerKind, tick int, faceIdx int, verbIdx int, wingIdx int, skinName string, elapsed string) string {
	frames := SpinnerFrames(kind)
	frame := frames[tick%len(frames)]
	face := ThinkingFace(faceIdx)
	verb := ThinkingVerb(verbIdx)
	wings := SpinnerWingsForSkin(skinName)

	if len(wings) > 0 && wingIdx < len(wings) {
		w := wings[wingIdx%len(wings)]
		return "  " + w.Left + " " + frame + " " + verb + " " + face + " " + w.Right + " (" + elapsed + "s)"
	}
	return "  " + frame + " " + verb + " " + face + " (" + elapsed + "s)"
}
