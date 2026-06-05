package details

import "testing"

func TestApplySlashUpdatesStateAndReturnsStatus(t *testing.T) {
	state, status := ApplySlash("/details hidden", DefaultState())
	if status != "details: hidden" || state.Global != ModeHidden || !state.CommandOverride {
		t.Fatalf("/details hidden = (%+v, %q), want hidden command override", state, status)
	}

	state, status = ApplySlash("/details tools expanded", state)
	if status != "details tools: expanded" || state.SectionMode(SectionTools) != ModeExpanded {
		t.Fatalf("/details tools expanded = (%+v, %q), want tools expanded", state, status)
	}

	state, status = ApplySlash("/details tools reset", state)
	if status != "details tools: reset" {
		t.Fatalf("/details tools reset status = %q", status)
	}
	if _, ok := state.Sections[SectionTools]; ok {
		t.Fatalf("tools override still present after reset: %+v", state.Sections)
	}

	_, status = ApplySlash("/details tools blink", state)
	if status != SectionSlashUsage {
		t.Fatalf("invalid section mode status = %q, want %q", status, SectionSlashUsage)
	}
	_, status = ApplySlash("/details nope", state)
	if status != SlashUsage {
		t.Fatalf("invalid global mode status = %q, want %q", status, SlashUsage)
	}
}

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
