package reasoning

import (
	"errors"
	"testing"
)

func TestDispatcherDispatchIsolatesSessionStateAndPersistsGlobal(t *testing.T) {
	var globalCalls []ReasoningEffort
	d := NewDispatcher(func(e ReasoningEffort) error {
		globalCalls = append(globalCalls, e)
		return nil
	})

	replyA, err := d.Dispatch("telegram:42", []string{"high"})
	if err != nil {
		t.Fatalf("Dispatch(A high) err = %v", err)
	}
	if replyA.Effort != ReasoningEffortHigh || replyA.Scope != ReasoningSourceSession {
		t.Fatalf("A high reply = %+v, want effort=high scope=session", replyA)
	}
	if len(globalCalls) != 0 {
		t.Fatalf("session-only set persisted globally: calls=%v", globalCalls)
	}

	showA, err := d.Dispatch("telegram:42", nil)
	if err != nil {
		t.Fatalf("Dispatch(A show) err = %v", err)
	}
	if showA.Effort != ReasoningEffortHigh || showA.Scope != ReasoningSourceSession {
		t.Fatalf("A show = %+v, want effort=high scope=session", showA)
	}

	showB, err := d.Dispatch("telegram:99", nil)
	if err != nil {
		t.Fatalf("Dispatch(B show) err = %v", err)
	}
	if showB.Effort != ReasoningEffort("") || showB.Scope != ReasoningSourceUnset {
		t.Fatalf("B show = %+v, want empty/unset", showB)
	}

	replyB, err := d.Dispatch("telegram:99", []string{"low", "--global"})
	if err != nil {
		t.Fatalf("Dispatch(B low --global) err = %v", err)
	}
	if replyB.Effort != ReasoningEffortLow || replyB.Scope != ReasoningSourceGlobal {
		t.Fatalf("B global reply = %+v, want effort=low scope=global", replyB)
	}
	if len(globalCalls) != 1 || globalCalls[0] != ReasoningEffortLow {
		t.Fatalf("persistGlobal calls = %v, want [low]", globalCalls)
	}

	showAAfter, err := d.Dispatch("telegram:42", nil)
	if err != nil {
		t.Fatalf("Dispatch(A show after) err = %v", err)
	}
	if showAAfter.Effort != ReasoningEffortHigh || showAAfter.Scope != ReasoningSourceSession {
		t.Fatalf("A show after = %+v, want effort=high scope=session", showAAfter)
	}
}

func TestDispatcherDispatchGlobalPersistFallback(t *testing.T) {
	persistErr := errors.New("config write failed")
	d := NewDispatcher(func(ReasoningEffort) error { return persistErr })

	reply, err := d.Dispatch("telegram:42", []string{"medium", "--global"})
	if err != nil {
		t.Fatalf("Dispatch err = %v, want nil", err)
	}
	if !reply.PersistFailed {
		t.Fatalf("reply.PersistFailed = false, want true")
	}
	if reply.Effort != ReasoningEffortMedium || reply.Scope != ReasoningSourceSession {
		t.Fatalf("fallback reply = %+v, want effort=medium scope=session", reply)
	}

	show, err := d.Dispatch("telegram:42", nil)
	if err != nil {
		t.Fatalf("Dispatch(show) err = %v", err)
	}
	if show.Effort != ReasoningEffortMedium || show.Scope != ReasoningSourceSession {
		t.Fatalf("post-fallback show = %+v, want effort=medium scope=session", show)
	}
}

func TestDispatcherDispatchReturnsParserError(t *testing.T) {
	d := NewDispatcher(nil)
	if _, err := d.Dispatch("telegram:42", []string{"bogus"}); !errors.Is(err, ErrInvalidEffort) {
		t.Fatalf("invalid effort err = %v, want ErrInvalidEffort", err)
	}
}
