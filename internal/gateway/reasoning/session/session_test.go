package session

import (
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/reasoning/model"
)

func TestApply_ShowReturnsCurrentScope(t *testing.T) {
	state := model.SessionReasoningState{Effort: model.ReasoningEffortHigh, Source: model.ReasoningSourceGlobal}
	calls := 0
	persist := func(model.ReasoningEffort) error { calls++; return nil }

	newState, reply := Apply(
		state,
		model.ReasoningCommand{Action: model.ReasoningActionShow},
		persist,
	)

	if newState != state {
		t.Fatalf("Show mutated state: got %+v, want %+v", newState, state)
	}
	if calls != 0 {
		t.Fatalf("Show called persistGlobal %d times, want 0", calls)
	}
	if reply.Effort != model.ReasoningEffortHigh {
		t.Fatalf("reply.Effort = %q, want %q", reply.Effort, model.ReasoningEffortHigh)
	}
	if reply.Scope != model.ReasoningSourceGlobal {
		t.Fatalf("reply.Scope = %q, want %q", reply.Scope, model.ReasoningSourceGlobal)
	}
	if reply.PersistFailed {
		t.Fatalf("reply.PersistFailed = true, want false")
	}
}

func TestApply_SetSessionMutatesOnly(t *testing.T) {
	state := model.SessionReasoningState{Source: model.ReasoningSourceUnset}
	calls := 0
	persist := func(model.ReasoningEffort) error { calls++; return nil }

	newState, reply := Apply(
		state,
		model.ReasoningCommand{Action: model.ReasoningActionSet, Effort: model.ReasoningEffortLow, Global: false},
		persist,
	)

	if calls != 0 {
		t.Fatalf("session set called persistGlobal %d times, want 0", calls)
	}
	wantState := model.SessionReasoningState{Effort: model.ReasoningEffortLow, Source: model.ReasoningSourceSession}
	if newState != wantState {
		t.Fatalf("newState = %+v, want %+v", newState, wantState)
	}
	if reply.Effort != model.ReasoningEffortLow || reply.Scope != model.ReasoningSourceSession {
		t.Fatalf("reply = %+v, want effort=%q scope=%q", reply, model.ReasoningEffortLow, model.ReasoningSourceSession)
	}
	if reply.PersistFailed {
		t.Fatalf("reply.PersistFailed = true, want false")
	}
}

func TestApply_SetGlobalCallsPersistGlobal(t *testing.T) {
	state := model.SessionReasoningState{Effort: model.ReasoningEffortMedium, Source: model.ReasoningSourceSession}
	var saved model.ReasoningEffort
	calls := 0
	persist := func(e model.ReasoningEffort) error {
		calls++
		saved = e
		return nil
	}

	newState, reply := Apply(
		state,
		model.ReasoningCommand{Action: model.ReasoningActionSet, Effort: model.ReasoningEffortHigh, Global: true},
		persist,
	)

	if calls != 1 {
		t.Fatalf("persistGlobal called %d times, want 1", calls)
	}
	if saved != model.ReasoningEffortHigh {
		t.Fatalf("persistGlobal effort = %q, want %q", saved, model.ReasoningEffortHigh)
	}
	wantState := model.SessionReasoningState{Effort: model.ReasoningEffortHigh, Source: model.ReasoningSourceGlobal}
	if newState != wantState {
		t.Fatalf("newState = %+v, want %+v", newState, wantState)
	}
	if reply.Effort != model.ReasoningEffortHigh || reply.Scope != model.ReasoningSourceGlobal {
		t.Fatalf("reply = %+v, want effort=%q scope=%q", reply, model.ReasoningEffortHigh, model.ReasoningSourceGlobal)
	}
	if reply.PersistFailed {
		t.Fatalf("reply.PersistFailed = true, want false")
	}
}

func TestApply_GlobalPersistFallback(t *testing.T) {
	state := model.SessionReasoningState{Source: model.ReasoningSourceUnset}
	persistErr := errors.New("disk full")
	calls := 0
	persist := func(model.ReasoningEffort) error {
		calls++
		return persistErr
	}

	newState, reply := Apply(
		state,
		model.ReasoningCommand{Action: model.ReasoningActionSet, Effort: model.ReasoningEffortLow, Global: true},
		persist,
	)

	if calls != 1 {
		t.Fatalf("persistGlobal called %d times, want 1", calls)
	}
	wantState := model.SessionReasoningState{Effort: model.ReasoningEffortLow, Source: model.ReasoningSourceSession}
	if newState != wantState {
		t.Fatalf("newState = %+v, want %+v", newState, wantState)
	}
	if !reply.PersistFailed {
		t.Fatalf("reply.PersistFailed = false, want true")
	}
	if reply.Effort != model.ReasoningEffortLow || reply.Scope != model.ReasoningSourceSession {
		t.Fatalf("reply = %+v, want effort=%q scope=%q", reply, model.ReasoningEffortLow, model.ReasoningSourceSession)
	}
}

func TestApply_ResetClearsSessionState(t *testing.T) {
	state := model.SessionReasoningState{Effort: model.ReasoningEffortHigh, Source: model.ReasoningSourceSession}
	calls := 0
	persist := func(model.ReasoningEffort) error { calls++; return nil }

	newState, reply := Apply(
		state,
		model.ReasoningCommand{Action: model.ReasoningActionReset},
		persist,
	)

	if calls != 0 {
		t.Fatalf("reset called persistGlobal %d times, want 0", calls)
	}
	wantState := model.SessionReasoningState{Effort: model.ReasoningEffort(""), Source: model.ReasoningSourceUnset}
	if newState != wantState {
		t.Fatalf("newState = %+v, want %+v", newState, wantState)
	}
	if reply.Effort != model.ReasoningEffort("") || reply.Scope != model.ReasoningSourceUnset {
		t.Fatalf("reply = %+v, want effort=\"\" scope=%q", reply, model.ReasoningSourceUnset)
	}
	if reply.PersistFailed {
		t.Fatalf("reply.PersistFailed = true, want false")
	}
}

func TestDispatcherDispatchIsolatesSessionStateAndPersistsGlobal(t *testing.T) {
	var globalCalls []model.ReasoningEffort
	d := NewDispatcher(func(e model.ReasoningEffort) error {
		globalCalls = append(globalCalls, e)
		return nil
	})

	replyA, err := d.Dispatch("telegram:42", []string{"high"})
	if err != nil {
		t.Fatalf("Dispatch(A high) err = %v", err)
	}
	if replyA.Effort != model.ReasoningEffortHigh || replyA.Scope != model.ReasoningSourceSession {
		t.Fatalf("A high reply = %+v, want effort=high scope=session", replyA)
	}
	if len(globalCalls) != 0 {
		t.Fatalf("session-only set persisted globally: calls=%v", globalCalls)
	}

	showA, err := d.Dispatch("telegram:42", nil)
	if err != nil {
		t.Fatalf("Dispatch(A show) err = %v", err)
	}
	if showA.Effort != model.ReasoningEffortHigh || showA.Scope != model.ReasoningSourceSession {
		t.Fatalf("A show = %+v, want effort=high scope=session", showA)
	}

	showB, err := d.Dispatch("telegram:99", nil)
	if err != nil {
		t.Fatalf("Dispatch(B show) err = %v", err)
	}
	if showB.Effort != model.ReasoningEffort("") || showB.Scope != model.ReasoningSourceUnset {
		t.Fatalf("B show = %+v, want empty/unset", showB)
	}

	replyB, err := d.Dispatch("telegram:99", []string{"low", "--global"})
	if err != nil {
		t.Fatalf("Dispatch(B low --global) err = %v", err)
	}
	if replyB.Effort != model.ReasoningEffortLow || replyB.Scope != model.ReasoningSourceGlobal {
		t.Fatalf("B global reply = %+v, want effort=low scope=global", replyB)
	}
	if len(globalCalls) != 1 || globalCalls[0] != model.ReasoningEffortLow {
		t.Fatalf("persistGlobal calls = %v, want [low]", globalCalls)
	}

	showAAfter, err := d.Dispatch("telegram:42", nil)
	if err != nil {
		t.Fatalf("Dispatch(A show after) err = %v", err)
	}
	if showAAfter.Effort != model.ReasoningEffortHigh || showAAfter.Scope != model.ReasoningSourceSession {
		t.Fatalf("A show after = %+v, want effort=high scope=session", showAAfter)
	}
}

func TestDispatcherDispatchGlobalPersistFallback(t *testing.T) {
	persistErr := errors.New("config write failed")
	d := NewDispatcher(func(model.ReasoningEffort) error { return persistErr })

	reply, err := d.Dispatch("telegram:42", []string{"medium", "--global"})
	if err != nil {
		t.Fatalf("Dispatch err = %v, want nil", err)
	}
	if !reply.PersistFailed {
		t.Fatalf("reply.PersistFailed = false, want true")
	}
	if reply.Effort != model.ReasoningEffortMedium || reply.Scope != model.ReasoningSourceSession {
		t.Fatalf("fallback reply = %+v, want effort=medium scope=session", reply)
	}

	show, err := d.Dispatch("telegram:42", nil)
	if err != nil {
		t.Fatalf("Dispatch(show) err = %v", err)
	}
	if show.Effort != model.ReasoningEffortMedium || show.Scope != model.ReasoningSourceSession {
		t.Fatalf("post-fallback show = %+v, want effort=medium scope=session", show)
	}
}

func TestDispatcherDispatchReturnsParserError(t *testing.T) {
	d := NewDispatcher(nil)
	if _, err := d.Dispatch("telegram:42", []string{"bogus"}); !errors.Is(err, model.ErrInvalidEffort) {
		t.Fatalf("invalid effort err = %v, want ErrInvalidEffort", err)
	}
}
