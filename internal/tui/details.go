package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/details"

type DetailsMode = details.Mode

const (
	DetailsModeHidden    DetailsMode = details.ModeHidden
	DetailsModeCollapsed DetailsMode = details.ModeCollapsed
	DetailsModeExpanded  DetailsMode = details.ModeExpanded
)

type DetailsSection = details.Section

const (
	DetailsSectionThinking  DetailsSection = details.SectionThinking
	DetailsSectionTools     DetailsSection = details.SectionTools
	DetailsSectionSubagents DetailsSection = details.SectionSubagents
	DetailsSectionActivity  DetailsSection = details.SectionActivity
)

// DetailsState owns Hermes ui-tui detail visibility semantics for the native
// Bubble Tea renderer. The zero value normalizes to Hermes' persisted-config
// default: global collapsed, thinking/tools expanded, activity hidden, and
// subagents falling through to global.
type DetailsState = details.State

func DefaultDetailsState() DetailsState {
	return details.DefaultState()
}

func NormalizeDetailsState(state DetailsState) DetailsState {
	return details.NormalizeState(state)
}

func ParseDetailsMode(raw string) (DetailsMode, bool) {
	return details.ParseMode(raw)
}

func ParseDetailsSection(raw string) (DetailsSection, bool) {
	return details.ParseSection(raw)
}

func NextDetailsMode(mode DetailsMode) DetailsMode {
	return details.NextMode(mode)
}
