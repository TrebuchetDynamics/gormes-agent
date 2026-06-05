package tui

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestRunningPlaceholderCompatibilityWrapper(t *testing.T) {
	m := NewModel(make(chan kernel.RenderFrame), func(string) {}, func() {})
	m.inFlight = false
	if got := m.RunningPlaceholder(); got != "Type a message and hit Enter…" {
		t.Fatalf("idle RunningPlaceholder() = %q", got)
	}

	registry := NewSlashRegistry()
	noop := func(string, *Model) SlashResult { return SlashResult{Handled: true} }
	registry.Register("queue", noop, WithBusyAvailable())
	registry.Register("steer", noop, WithBusyAvailable())
	m.slashRegistry = registry
	m.inFlight = true
	if got := m.RunningPlaceholder(); got != "msg=interrupt · /queue · /steer · Ctrl+C cancel" {
		t.Fatalf("busy RunningPlaceholder() = %q", got)
	}
}
