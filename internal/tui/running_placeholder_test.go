package tui

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// TestRunningPlaceholder_IdleStateUsesIdleText asserts that when the model is
// not in flight, the placeholder is the same prompt the editor ships with at
// startup. Tracks Hermes commit eaa7e2db's idle copy.
func TestRunningPlaceholder_IdleStateUsesIdleText(t *testing.T) {
	m := NewModel(make(chan kernel.RenderFrame), func(string) {}, func() {})
	m.inFlight = false

	got := m.RunningPlaceholder()
	want := "Type a message and hit Enter…"
	if got != want {
		t.Fatalf("RunningPlaceholder() = %q, want %q", got, want)
	}
}

// TestRunningPlaceholder_BusyStateListsRegisteredBusySlashes asserts the
// in-flight placeholder enumerates every slash command registered with
// WithBusyAvailable(), in alphabetical order, between the interrupt prefix and
// the Ctrl+C cancel suffix.
func TestRunningPlaceholder_BusyStateListsRegisteredBusySlashes(t *testing.T) {
	registry := NewSlashRegistry()
	noop := func(string, *Model) SlashResult { return SlashResult{Handled: true} }
	registry.Register("queue", noop, WithBusyAvailable())
	registry.Register("steer", noop, WithBusyAvailable())

	m := NewModel(make(chan kernel.RenderFrame), func(string) {}, func() {})
	m.slashRegistry = registry
	m.inFlight = true

	got := m.RunningPlaceholder()
	want := "msg=interrupt · /queue · /steer · Ctrl+C cancel"
	if got != want {
		t.Fatalf("RunningPlaceholder() = %q, want %q", got, want)
	}
}

// TestRunningPlaceholder_BusyStateWithNoBusySlashesShowsMinimum asserts the
// degraded form: with zero busy-available slashes registered, the placeholder
// surfaces only the always-on interrupt + cancel hints.
func TestRunningPlaceholder_BusyStateWithNoBusySlashesShowsMinimum(t *testing.T) {
	m := NewModel(make(chan kernel.RenderFrame), func(string) {}, func() {})
	m.slashRegistry = NewSlashRegistry()
	m.inFlight = true

	got := m.RunningPlaceholder()
	want := "msg=interrupt · Ctrl+C cancel"
	if got != want {
		t.Fatalf("RunningPlaceholder() = %q, want %q", got, want)
	}
}
