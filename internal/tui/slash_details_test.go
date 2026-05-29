package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestDetailsStateMatchesHermesSectionDefaults(t *testing.T) {
	state := DefaultDetailsState()
	if got := state.SectionMode(DetailsSectionThinking); got != DetailsModeExpanded {
		t.Fatalf("thinking default = %q, want expanded", got)
	}
	if got := state.SectionMode(DetailsSectionTools); got != DetailsModeExpanded {
		t.Fatalf("tools default = %q, want expanded", got)
	}
	if got := state.SectionMode(DetailsSectionActivity); got != DetailsModeHidden {
		t.Fatalf("activity default = %q, want hidden", got)
	}
	if got := state.SectionMode(DetailsSectionSubagents); got != DetailsModeCollapsed {
		t.Fatalf("subagents default = %q, want collapsed", got)
	}

	state.Global = DetailsModeHidden
	state.CommandOverride = true
	if got := state.SectionMode(DetailsSectionThinking); got != DetailsModeHidden {
		t.Fatalf("command override thinking = %q, want hidden", got)
	}
	state.Sections = map[DetailsSection]DetailsMode{DetailsSectionTools: DetailsModeExpanded}
	if got := state.SectionMode(DetailsSectionTools); got != DetailsModeExpanded {
		t.Fatalf("section override tools = %q, want expanded", got)
	}
}

func TestDetailsSlashControlsThinkingAndToolVisibilityWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	m := newDetailsSlashModel(sub)

	initial := m.View()
	if !strings.Contains(initial, "terminal: ls -la") {
		t.Fatalf("initial view missing tool progress before /details hidden:\n%s", initial)
	}

	m = enterSlashDispatchBehavior(t, m, "/details hidden")
	if sub.calls != 0 {
		t.Fatalf("/details reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /details = %q, want cleared", got)
	}
	if m.detailsState.Global != DetailsModeHidden || !m.detailsState.CommandOverride {
		t.Fatalf("details state after /details hidden = %+v, want hidden command override", m.detailsState)
	}
	if !strings.Contains(m.statusMessage, "details: hidden") {
		t.Fatalf("status after /details hidden = %q, want hidden evidence", m.statusMessage)
	}
	if got := m.View(); strings.Contains(got, "terminal: ls -la") {
		t.Fatalf("/details hidden still rendered tool progress:\n%s", got)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/details fell through to fallback: %q", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, m, "/details tools expanded")
	if got := m.detailsState.SectionMode(DetailsSectionTools); got != DetailsModeExpanded {
		t.Fatalf("tools section after override = %q, want expanded", got)
	}
	if got := m.View(); !strings.Contains(got, "terminal: ls -la") {
		t.Fatalf("/details tools expanded did not restore tool progress:\n%s", got)
	}

	m = enterSlashDispatchBehavior(t, m, "/details tools reset")
	if _, ok := m.detailsState.Sections[DetailsSectionTools]; ok {
		t.Fatalf("tools override still present after reset: %+v", m.detailsState.Sections)
	}
	if got := m.View(); strings.Contains(got, "terminal: ls -la") {
		t.Fatalf("/details tools reset should fall back to global hidden:\n%s", got)
	}
}

func TestDetailsSlashHidesFallbackThinkingIndicator(t *testing.T) {
	m := newDetailsSlashModel(&nopSubmitter{})
	m.frame.SoulEvents = nil
	if got := m.View(); !strings.Contains(got, "Reasoning") {
		t.Fatalf("active view missing fallback thinking before /details hidden:\n%s", got)
	}

	m = enterSlashDispatchBehavior(t, m, "/details hidden")
	if got := m.View(); strings.Contains(got, "Reasoning") {
		t.Fatalf("/details hidden still rendered fallback thinking:\n%s", got)
	}
}

func TestDetailsSlashToggleSectionUsageAndCompletions(t *testing.T) {
	m := newDetailsSlashModel(&nopSubmitter{})
	m = enterSlashDispatchBehavior(t, m, "/details toggle")
	if m.detailsState.Global != DetailsModeExpanded || !m.detailsState.CommandOverride {
		t.Fatalf("details after toggle from default = %+v, want expanded command override", m.detailsState)
	}
	m = enterSlashDispatchBehavior(t, m, "/details activity collapsed")
	if got := m.detailsState.SectionMode(DetailsSectionActivity); got != DetailsModeCollapsed {
		t.Fatalf("activity override = %q, want collapsed", got)
	}
	m = enterSlashDispatchBehavior(t, m, "/details tools blink")
	if !strings.Contains(m.statusMessage, "usage: /details <section> [hidden|collapsed|expanded|reset]") {
		t.Fatalf("status after invalid section mode = %q, want section usage", m.statusMessage)
	}
	m = enterSlashDispatchBehavior(t, m, "/details nope")
	if !strings.Contains(m.statusMessage, "usage: /details [hidden|collapsed|expanded|cycle]") {
		t.Fatalf("status after invalid global mode = %q, want global usage", m.statusMessage)
	}

	var detailsCompletion *SlashCompletion
	for _, completion := range HermesSlashCommandCompletions("/det") {
		if completion.Name == "details" {
			c := completion
			detailsCompletion = &c
			break
		}
	}
	if detailsCompletion == nil || !detailsCompletion.Available {
		t.Fatalf("/det completion = %+v, want available details", detailsCompletion)
	}
	wantSubs := []string{"hidden", "collapsed", "expanded", "cycle", "toggle", "thinking", "tools", "subagents", "activity"}
	if got := completionNames(HermesSlashSubcommandCompletions("/details ")); !reflect.DeepEqual(got, wantSubs) {
		t.Fatalf("/details subcommands = %v, want %v", got, wantSubs)
	}
	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "details" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want details", busy)
}

func newDetailsSlashModel(sub *nopSubmitter) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frame := kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		Model:     "openai/gpt-4.1",
		SessionID: "sess-details",
		History:   []llm.Message{{Role: "user", Content: "inspect details"}},
		SoulEvents: []kernel.SoulEntry{
			{Text: "tool: terminal: ls -la"},
		},
	}
	frames <- frame
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true})
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}
