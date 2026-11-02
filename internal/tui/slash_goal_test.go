package tui

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/goal"
)

func newGoalTestModel(command GoalCommandFunc, status GoalStatusFunc) Model {
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{
		GoalCommand: command,
		GoalStatus:  status,
	})
	m.frame = kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "session-goal"}
	return m
}

func TestGoalSlashHandlerDispatchesToController(t *testing.T) {
	var gotSession, gotInput string
	m := newGoalTestModel(func(sessionID, input string) (string, error) {
		gotSession, gotInput = sessionID, input
		return "Goal active (2/20): ship parity", nil
	}, nil)

	result := NewDefaultSlashRegistry().Dispatch("/goal status", &m)
	if !result.Handled || result.StatusMessage != "Goal active (2/20): ship parity" {
		t.Fatalf("result = %+v", result)
	}
	if gotSession != "session-goal" || gotInput != "/goal status" {
		t.Fatalf("controller args = %q %q", gotSession, gotInput)
	}
}

func TestGoalSlashHandlerUnavailableWithoutController(t *testing.T) {
	m := newGoalTestModel(nil, nil)
	result := NewDefaultSlashRegistry().Dispatch("/goal", &m)
	if !result.Handled || !strings.Contains(result.StatusMessage, "goal store unavailable") {
		t.Fatalf("result = %+v", result)
	}
}

func TestGoalSlashHandlerRequiresSession(t *testing.T) {
	m := newGoalTestModel(func(string, string) (string, error) { return "ok", nil }, nil)
	m.frame.SessionID = ""
	result := NewDefaultSlashRegistry().Dispatch("/goal status", &m)
	if !result.Handled || !strings.Contains(result.StatusMessage, "session unavailable") {
		t.Fatalf("result = %+v", result)
	}
}

func TestGoalSlashHandlerBlocksNewGoalDuringTurn(t *testing.T) {
	called := false
	m := newGoalTestModel(func(string, string) (string, error) {
		called = true
		return "ok", nil
	}, nil)
	m.frame.Phase = kernel.PhaseStreaming

	result := NewDefaultSlashRegistry().Dispatch("/goal ship it", &m)
	if !result.Handled || !strings.Contains(result.StatusMessage, "Agent is running") {
		t.Fatalf("result = %+v", result)
	}
	if called {
		t.Fatal("controller called for a new goal during an active turn")
	}

	result = NewDefaultSlashRegistry().Dispatch("/goal status", &m)
	if !result.Handled || result.StatusMessage != "ok" || !called {
		t.Fatalf("status should be allowed mid-turn: %+v called=%v", result, called)
	}
}

func TestGoalSlashSetsGoalParser(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"/goal ship it", true},
		{"/goal set ship it", true},
		{"/goal status", false},
		{"/goal pause", false},
		{"/goal resume", false},
		{"/goal clear", false},
		{"/goal done", false},
		{"/goal subgoal add tests", false},
		{"/goal", false},
	}
	for _, tt := range tests {
		if got := goalSlashSetsGoal(tt.input); got != tt.want {
			t.Errorf("goalSlashSetsGoal(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestModelRenderGoalBar(t *testing.T) {
	m := newGoalTestModel(nil, func(sessionID string) (goal.Line, bool) {
		if sessionID != "session-goal" {
			t.Fatalf("session id = %q", sessionID)
		}
		return goal.Line{Glyph: "⊙", Label: "goal", Detail: "3/20 turns", Title: "ship parity"}, true
	})
	m.refreshGoalLine()
	if got := m.renderGoalBar(80); got != "⊙ goal · 3/20 turns · ship parity" {
		t.Fatalf("bar = %q", got)
	}

	m.goalStatus = func(string) (goal.Line, bool) { return goal.Line{}, false }
	m.refreshGoalLine()
	if got := m.renderGoalBar(80); got != "" {
		t.Fatalf("inactive bar = %q", got)
	}
}
