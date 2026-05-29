package details

import "testing"

func TestStateMatchesHermesSectionDefaults(t *testing.T) {
	state := DefaultState()
	if got := state.SectionMode(SectionThinking); got != ModeExpanded {
		t.Fatalf("thinking default = %q, want expanded", got)
	}
	if got := state.SectionMode(SectionTools); got != ModeExpanded {
		t.Fatalf("tools default = %q, want expanded", got)
	}
	if got := state.SectionMode(SectionActivity); got != ModeHidden {
		t.Fatalf("activity default = %q, want hidden", got)
	}
	if got := state.SectionMode(SectionSubagents); got != ModeCollapsed {
		t.Fatalf("subagents default = %q, want collapsed", got)
	}

	state.Global = ModeHidden
	state.CommandOverride = true
	if got := state.SectionMode(SectionThinking); got != ModeHidden {
		t.Fatalf("command override thinking = %q, want hidden", got)
	}
	state.Sections = map[Section]Mode{SectionTools: ModeExpanded}
	if got := state.SectionMode(SectionTools); got != ModeExpanded {
		t.Fatalf("section override tools = %q, want expanded", got)
	}
}
