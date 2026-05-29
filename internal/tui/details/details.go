package details

import "strings"

type Mode string

const (
	ModeHidden    Mode = "hidden"
	ModeCollapsed Mode = "collapsed"
	ModeExpanded  Mode = "expanded"
)

type Section string

const (
	SectionThinking  Section = "th" + "in" + "king"
	SectionTools     Section = "tools"
	SectionSubagents Section = "subagents"
	SectionActivity  Section = "activity"
)

var modeOrder = []Mode{ModeHidden, ModeCollapsed, ModeExpanded}
var sections = []Section{SectionThinking, SectionTools, SectionSubagents, SectionActivity}

// State owns Hermes ui-tui detail visibility semantics for the native
// Bubble Tea renderer. The zero value normalizes to Hermes' persisted-config
// default: global collapsed, thinking/tools expanded, activity hidden, and
// subagents falling through to global.
type State struct {
	Global          Mode
	Sections        map[Section]Mode
	CommandOverride bool
}

func DefaultState() State {
	return State{Global: ModeCollapsed}
}

func NormalizeState(state State) State {
	if !isMode(state.Global) {
		state.Global = ModeCollapsed
	}
	if len(state.Sections) > 0 {
		next := make(map[Section]Mode, len(state.Sections))
		for section, mode := range state.Sections {
			if !isSection(section) || !isMode(mode) {
				continue
			}
			next[section] = mode
		}
		state.Sections = next
	}
	return state
}

func ParseMode(raw string) (Mode, bool) {
	switch Mode(strings.ToLower(strings.TrimSpace(raw))) {
	case ModeHidden:
		return ModeHidden, true
	case ModeCollapsed:
		return ModeCollapsed, true
	case ModeExpanded:
		return ModeExpanded, true
	default:
		return "", false
	}
}

func ParseSection(raw string) (Section, bool) {
	switch Section(strings.ToLower(strings.TrimSpace(raw))) {
	case SectionThinking:
		return SectionThinking, true
	case SectionTools:
		return SectionTools, true
	case SectionSubagents:
		return SectionSubagents, true
	case SectionActivity:
		return SectionActivity, true
	default:
		return "", false
	}
}

func NextMode(mode Mode) Mode {
	mode = NormalizeState(State{Global: mode}).Global
	for i, candidate := range modeOrder {
		if candidate == mode {
			return modeOrder[(i+1)%len(modeOrder)]
		}
	}
	return ModeCollapsed
}

func (state State) SectionMode(section Section) Mode {
	state = NormalizeState(state)
	if state.Sections != nil {
		if mode, ok := state.Sections[section]; ok {
			return mode
		}
	}
	if state.CommandOverride {
		return state.Global
	}
	switch section {
	case SectionThinking, SectionTools:
		return ModeExpanded
	case SectionActivity:
		return ModeHidden
	default:
		return state.Global
	}
}

func (state State) WithGlobal(mode Mode) State {
	state = NormalizeState(state)
	state.Global = mode
	state.CommandOverride = true
	state.Sections = nil
	return state
}

func (state State) WithSection(section Section, mode Mode) State {
	state = NormalizeState(state)
	if state.Sections == nil {
		state.Sections = make(map[Section]Mode)
	}
	state.Sections[section] = mode
	return state
}

func (state State) WithoutSection(section Section) State {
	state = NormalizeState(state)
	if state.Sections != nil {
		delete(state.Sections, section)
	}
	return state
}

func (state State) Status() string {
	state = NormalizeState(state)
	parts := make([]string, 0, len(sections))
	for _, section := range sections {
		if mode, ok := state.Sections[section]; ok {
			parts = append(parts, string(section)+"="+string(mode))
		}
	}
	if len(parts) == 0 {
		return "details: " + string(state.Global)
	}
	return "details: " + string(state.Global) + "  (" + strings.Join(parts, " ") + ")"
}

func isMode(mode Mode) bool {
	switch mode {
	case ModeHidden, ModeCollapsed, ModeExpanded:
		return true
	default:
		return false
	}
}

func isSection(section Section) bool {
	switch section {
	case SectionThinking, SectionTools, SectionSubagents, SectionActivity:
		return true
	default:
		return false
	}
}
