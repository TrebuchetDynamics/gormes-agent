package spinner

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/spinner/animation"

type Kind = animation.Kind

const (
	Dots    Kind = animation.Dots
	Bounce  Kind = animation.Bounce
	Grow    Kind = animation.Grow
	Arrows  Kind = animation.Arrows
	Star    Kind = animation.Star
	Moon    Kind = animation.Moon
	Pulse   Kind = animation.Pulse
	Brain   Kind = animation.Brain
	Sparkle Kind = animation.Sparkle
	Emoji   Kind = animation.Emoji
	Kaomoji Kind = animation.Kaomoji
)

type Wing = animation.Wing

func Frames(kind Kind) []string { return animation.Frames(kind) }

func WaitingFace(idx int) string { return animation.WaitingFace(idx) }

func ThinkingFace(idx int) string { return animation.ThinkingFace(idx) }

func ThinkingVerb(idx int) string { return animation.ThinkingVerb(idx) }

func WingsForSkin(skinName string) []Wing { return animation.WingsForSkin(skinName) }

func Render(kind Kind, tick int, faceIdx int, verbIdx int, wingIdx int, skinName string, elapsed string) string {
	return animation.Render(kind, tick, faceIdx, verbIdx, wingIdx, skinName, elapsed)
}
