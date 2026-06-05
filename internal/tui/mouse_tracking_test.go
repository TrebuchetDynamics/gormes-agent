package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestMouseSlashUpdatesRuntimeWithoutSubmitting(t *testing.T) {
	var submitted []string
	rec := &mouseModeRecorder{}

	m := NewModelWithOptions(
		make(chan kernel.RenderFrame),
		func(text string) { submitted = append(submitted, text) },
		func() {},
		Options{MouseTracking: false, MouseModeCmd: rec.cmd},
	)

	m.editor.SetValue("/mouse on")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	runTestCmd(t, cmd)

	if len(submitted) != 0 {
		t.Fatalf("/mouse was submitted to kernel: %#v", submitted)
	}
	if !m.mouseTracking {
		t.Fatal("mouseTracking = false after /mouse on, want true")
	}
	if !reflect.DeepEqual(rec.modes, []bool{true}) {
		t.Fatalf("emitted modes = %#v, want enable once", rec.modes)
	}

	m.editor.SetValue("/mouse on")
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	runTestCmd(t, cmd)

	if !reflect.DeepEqual(rec.modes, []bool{true}) {
		t.Fatalf("repeated /mouse on emitted modes = %#v, want no duplicate", rec.modes)
	}

	m.editor.SetValue("/mouse toggle")
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	runTestCmd(t, cmd)

	if m.mouseTracking {
		t.Fatal("mouseTracking = true after /mouse toggle from on, want false")
	}
	if !reflect.DeepEqual(rec.modes, []bool{true, false}) {
		t.Fatalf("emitted modes = %#v, want enable then disable", rec.modes)
	}
}

func TestInitDisablesMouseTrackingWhenConfiguredOff(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	rec := &mouseModeRecorder{}

	m := NewModelWithOptions(
		frames,
		func(string) {},
		func() {},
		Options{MouseTracking: false, MouseModeCmd: rec.cmd},
	)

	runTestCmd(t, m.Init())

	if !reflect.DeepEqual(rec.modes, []bool{false}) {
		t.Fatalf("initial emitted modes = %#v, want explicit disable on alt-screen entry", rec.modes)
	}
}

func TestViewKeepsMouseTrackingOffQuietWhenIdle(t *testing.T) {
	m := NewModelWithOptions(
		make(chan kernel.RenderFrame),
		func(string) {},
		func() {},
		Options{MouseTracking: false},
	)
	m.width = 120
	m.height = 40
	m.frame = kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "hermes-agent"}

	if out := m.View(); strings.Contains(out, "mouse: disabled") {
		t.Fatalf("idle View() leaked persistent disabled mouse noise:\n%s", out)
	}
}

type mouseModeRecorder struct {
	modes []bool
}

func (r *mouseModeRecorder) cmd(enabled bool) tea.Cmd {
	return func() tea.Msg {
		r.modes = append(r.modes, enabled)
		return nil
	}
}

func runTestCmd(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case nil:
	case tea.BatchMsg:
		for _, c := range msg {
			runTestCmd(t, c)
		}
	default:
	}
}
