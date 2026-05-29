package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/spinner"

type SpinnerKind = spinner.Kind

const (
	SpinnerDots    SpinnerKind = spinner.Dots
	SpinnerBounce  SpinnerKind = spinner.Bounce
	SpinnerGrow    SpinnerKind = spinner.Grow
	SpinnerArrows  SpinnerKind = spinner.Arrows
	SpinnerStar    SpinnerKind = spinner.Star
	SpinnerMoon    SpinnerKind = spinner.Moon
	SpinnerPulse   SpinnerKind = spinner.Pulse
	SpinnerBrain   SpinnerKind = spinner.Brain
	SpinnerSparkle SpinnerKind = spinner.Sparkle
	SpinnerEmoji   SpinnerKind = spinner.Emoji
	SpinnerKaomoji SpinnerKind = spinner.Kaomoji
)

type SpinnerWing = spinner.Wing

func SpinnerFrames(kind SpinnerKind) []string {
	return spinner.Frames(kind)
}

func WaitingFace(idx int) string {
	return spinner.WaitingFace(idx)
}

func ThinkingFace(idx int) string {
	return spinner.ThinkingFace(idx)
}

func ThinkingVerb(idx int) string {
	return spinner.ThinkingVerb(idx)
}

func SpinnerWingsForSkin(skinName string) []SpinnerWing {
	return spinner.WingsForSkin(skinName)
}

func SpinnerRender(kind SpinnerKind, tick int, faceIdx int, verbIdx int, wingIdx int, skinName string, elapsed string) string {
	return spinner.Render(kind, tick, faceIdx, verbIdx, wingIdx, skinName, elapsed)
}
