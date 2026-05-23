package tui

import "strings"

type DetailsMode string

const (
	DetailsModeHidden    DetailsMode = "hidden"
	DetailsModeCollapsed DetailsMode = "collapsed"
	DetailsModeExpanded  DetailsMode = "expanded"
)

type DetailsSection string

const (
	DetailsSectionThinking  DetailsSection = "th" + "in" + "king"
	DetailsSectionTools     DetailsSection = "tools"
	DetailsSectionSubagents DetailsSection = "subagents"
	DetailsSectionActivity  DetailsSection = "activity"
)

var detailsModeOrder = []DetailsMode{DetailsModeHidden, DetailsModeCollapsed, DetailsModeExpanded}
var detailsSections = []DetailsSection{DetailsSectionThinking, DetailsSectionTools, DetailsSectionSubagents, DetailsSectionActivity}

// DetailsState owns Hermes ui-tui detail visibility semantics for the native
// Bubble Tea renderer. The zero value normalizes to Hermes' persisted-config
// default: global collapsed, thinking/tools expanded, activity hidden, and
// subagents falling through to global.
type DetailsState struct {
	Global          DetailsMode
	Sections        map[DetailsSection]DetailsMode
	CommandOverride bool
}

func DefaultDetailsState() DetailsState {
	return DetailsState{Global: DetailsModeCollapsed}
}

func NormalizeDetailsState(state DetailsState) DetailsState {
	if !isDetailsMode(state.Global) {
		state.Global = DetailsModeCollapsed
	}
	if len(state.Sections) > 0 {
		next := make(map[DetailsSection]DetailsMode, len(state.Sections))
		for section, mode := range state.Sections {
			if !isDetailsSection(section) || !isDetailsMode(mode) {
				continue
			}
			next[section] = mode
		}
		state.Sections = next
	}
	return state
}

func ParseDetailsMode(raw string) (DetailsMode, bool) {
	switch DetailsMode(strings.ToLower(strings.TrimSpace(raw))) {
	case DetailsModeHidden:
		return DetailsModeHidden, true
	case DetailsModeCollapsed:
		return DetailsModeCollapsed, true
	case DetailsModeExpanded:
		return DetailsModeExpanded, true
	default:
		return "", false
	}
}

func ParseDetailsSection(raw string) (DetailsSection, bool) {
	switch DetailsSection(strings.ToLower(strings.TrimSpace(raw))) {
	case DetailsSectionThinking:
		return DetailsSectionThinking, true
	case DetailsSectionTools:
		return DetailsSectionTools, true
	case DetailsSectionSubagents:
		return DetailsSectionSubagents, true
	case DetailsSectionActivity:
		return DetailsSectionActivity, true
	default:
		return "", false
	}
}

func NextDetailsMode(mode DetailsMode) DetailsMode {
	mode = NormalizeDetailsState(DetailsState{Global: mode}).Global
	for i, candidate := range detailsModeOrder {
		if candidate == mode {
			return detailsModeOrder[(i+1)%len(detailsModeOrder)]
		}
	}
	return DetailsModeCollapsed
}

func (state DetailsState) SectionMode(section DetailsSection) DetailsMode {
	state = NormalizeDetailsState(state)
	if state.Sections != nil {
		if mode, ok := state.Sections[section]; ok {
			return mode
		}
	}
	if state.CommandOverride {
		return state.Global
	}
	switch section {
	case DetailsSectionThinking, DetailsSectionTools:
		return DetailsModeExpanded
	case DetailsSectionActivity:
		return DetailsModeHidden
	default:
		return state.Global
	}
}

func (state DetailsState) WithGlobal(mode DetailsMode) DetailsState {
	state = NormalizeDetailsState(state)
	state.Global = mode
	state.CommandOverride = true
	state.Sections = nil
	return state
}

func (state DetailsState) WithSection(section DetailsSection, mode DetailsMode) DetailsState {
	state = NormalizeDetailsState(state)
	if state.Sections == nil {
		state.Sections = make(map[DetailsSection]DetailsMode)
	}
	state.Sections[section] = mode
	return state
}

func (state DetailsState) WithoutSection(section DetailsSection) DetailsState {
	state = NormalizeDetailsState(state)
	if state.Sections != nil {
		delete(state.Sections, section)
	}
	return state
}

func (state DetailsState) Status() string {
	state = NormalizeDetailsState(state)
	parts := make([]string, 0, len(detailsSections))
	for _, section := range detailsSections {
		if mode, ok := state.Sections[section]; ok {
			parts = append(parts, string(section)+"="+string(mode))
		}
	}
	if len(parts) == 0 {
		return "details: " + string(state.Global)
	}
	return "details: " + string(state.Global) + "  (" + strings.Join(parts, " ") + ")"
}

func isDetailsMode(mode DetailsMode) bool {
	switch mode {
	case DetailsModeHidden, DetailsModeCollapsed, DetailsModeExpanded:
		return true
	default:
		return false
	}
}

func isDetailsSection(section DetailsSection) bool {
	switch section {
	case DetailsSectionThinking, DetailsSectionTools, DetailsSectionSubagents, DetailsSectionActivity:
		return true
	default:
		return false
	}
}
