package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestKanbanSlashDispatchUsesInjectedRunner(t *testing.T) {
	var gotInput string
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{
		KanbanSlash: func(input string) (string, error) {
			gotInput = input
			return "No Kanban tasks.", nil
		},
	})

	res := NewDefaultSlashRegistry().Dispatch("/kanban list", &m)
	if !res.Handled {
		t.Fatal("Handled = false for /kanban, want native TUI handler")
	}
	if gotInput != "/kanban list" {
		t.Fatalf("runner input = %q, want full slash input", gotInput)
	}
	if res.StatusMessage != "No Kanban tasks." {
		t.Fatalf("StatusMessage = %q, want command output", res.StatusMessage)
	}
}

func TestKanbanSlashDispatchConsumesEditorWithoutSubmit(t *testing.T) {
	sub := &nopSubmitter{}
	m := newSlashDispatchBehaviorModel(sub)
	m.kanbanSlash = func(string) (string, error) {
		return "kanban initialized at /tmp/gormes/kanban.db", nil
	}

	m = enterSlashDispatchBehavior(t, m, "/kanban init")
	if sub.calls != 0 {
		t.Fatalf("/kanban reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /kanban = %q, want cleared", got)
	}
	if !strings.Contains(m.statusMessage, "kanban initialized") {
		t.Fatalf("status after /kanban = %q, want command output", m.statusMessage)
	}
}

func TestKanbanSlashDispatchFailureIsEvidence(t *testing.T) {
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{
		KanbanSlash: func(string) (string, error) {
			return "", errors.New("kanban db unavailable")
		},
	})

	res := NewDefaultSlashRegistry().Dispatch("/kanban list", &m)
	if !res.Handled {
		t.Fatal("Handled = false for /kanban error, want consumed")
	}
	if !strings.Contains(res.StatusMessage, "kanban: kanban db unavailable") {
		t.Fatalf("StatusMessage = %q, want kanban error evidence", res.StatusMessage)
	}
}

func TestKanbanSlashIsBusyAvailable(t *testing.T) {
	names := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range names {
		if name == "kanban" {
			return
		}
	}
	t.Fatalf("busy-available slashes = %v, want kanban", names)
}

func TestKanbanSlashStatusIsBounded(t *testing.T) {
	long := strings.Repeat("task\n", 240)
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{
		KanbanSlash: func(string) (string, error) {
			return long, nil
		},
	})

	res := NewDefaultSlashRegistry().Dispatch("/kanban list", &m)
	if len(res.StatusMessage) > maxKanbanSlashStatusRunes+len("...") {
		t.Fatalf("status length = %d, want bounded to %d", len(res.StatusMessage), maxKanbanSlashStatusRunes)
	}
}
