package reasoning

import (
	"errors"
	"testing"
)

func TestApplyReasoningCommand_ShowReturnsCurrentScope(t *testing.T) {
	state := SessionReasoningState{Effort: ReasoningEffortHigh, Source: ReasoningSourceGlobal}
	calls := 0
	persist := func(ReasoningEffort) error { calls++; return nil }

	newState, reply := ApplyReasoningCommand(
		state,
		ReasoningCommand{Action: ReasoningActionShow},
		persist,
	)

	if newState != state {
		t.Fatalf("Show mutated state: got %+v, want %+v", newState, state)
	}
	if calls != 0 {
		t.Fatalf("Show called persistGlobal %d times, want 0", calls)
	}
	if reply.Effort != ReasoningEffortHigh {
		t.Fatalf("reply.Effort = %q, want %q", reply.Effort, ReasoningEffortHigh)
	}
	if reply.Scope != ReasoningSourceGlobal {
		t.Fatalf("reply.Scope = %q, want %q", reply.Scope, ReasoningSourceGlobal)
	}
	if reply.PersistFailed {
		t.Fatalf("reply.PersistFailed = true, want false")
	}
}

func TestApplyReasoningCommand_SetSessionMutatesOnly(t *testing.T) {
	state := SessionReasoningState{Source: ReasoningSourceUnset}
	calls := 0
	persist := func(ReasoningEffort) error { calls++; return nil }

	newState, reply := ApplyReasoningCommand(
		state,
		ReasoningCommand{Action: ReasoningActionSet, Effort: ReasoningEffortLow, Global: false},
		persist,
	)

	if calls != 0 {
		t.Fatalf("session set called persistGlobal %d times, want 0", calls)
	}
	wantState := SessionReasoningState{Effort: ReasoningEffortLow, Source: ReasoningSourceSession}
	if newState != wantState {
		t.Fatalf("newState = %+v, want %+v", newState, wantState)
	}
	if reply.Effort != ReasoningEffortLow || reply.Scope != ReasoningSourceSession {
		t.Fatalf("reply = %+v, want effort=%q scope=%q", reply, ReasoningEffortLow, ReasoningSourceSession)
	}
	if reply.PersistFailed {
		t.Fatalf("reply.PersistFailed = true, want false")
	}
}

func TestApplyReasoningCommand_SetGlobalCallsPersistGlobal(t *testing.T) {
	state := SessionReasoningState{Effort: ReasoningEffortMedium, Source: ReasoningSourceSession}
	var saved ReasoningEffort
	calls := 0
	persist := func(e ReasoningEffort) error {
		calls++
		saved = e
		return nil
	}

	newState, reply := ApplyReasoningCommand(
		state,
		ReasoningCommand{Action: ReasoningActionSet, Effort: ReasoningEffortHigh, Global: true},
		persist,
	)

	if calls != 1 {
		t.Fatalf("persistGlobal called %d times, want 1", calls)
	}
	if saved != ReasoningEffortHigh {
		t.Fatalf("persistGlobal effort = %q, want %q", saved, ReasoningEffortHigh)
	}
	wantState := SessionReasoningState{Effort: ReasoningEffortHigh, Source: ReasoningSourceGlobal}
	if newState != wantState {
		t.Fatalf("newState = %+v, want %+v", newState, wantState)
	}
	if reply.Effort != ReasoningEffortHigh || reply.Scope != ReasoningSourceGlobal {
		t.Fatalf("reply = %+v, want effort=%q scope=%q", reply, ReasoningEffortHigh, ReasoningSourceGlobal)
	}
	if reply.PersistFailed {
		t.Fatalf("reply.PersistFailed = true, want false")
	}
}

func TestApplyReasoningCommand_GlobalPersistFallback(t *testing.T) {
	state := SessionReasoningState{Source: ReasoningSourceUnset}
	persistErr := errors.New("disk full")
	calls := 0
	persist := func(ReasoningEffort) error {
		calls++
		return persistErr
	}

	newState, reply := ApplyReasoningCommand(
		state,
		ReasoningCommand{Action: ReasoningActionSet, Effort: ReasoningEffortLow, Global: true},
		persist,
	)

	if calls != 1 {
		t.Fatalf("persistGlobal called %d times, want 1", calls)
	}
	wantState := SessionReasoningState{Effort: ReasoningEffortLow, Source: ReasoningSourceSession}
	if newState != wantState {
		t.Fatalf("newState = %+v, want %+v", newState, wantState)
	}
	if !reply.PersistFailed {
		t.Fatalf("reply.PersistFailed = false, want true")
	}
	if reply.Effort != ReasoningEffortLow || reply.Scope != ReasoningSourceSession {
		t.Fatalf("reply = %+v, want effort=%q scope=%q", reply, ReasoningEffortLow, ReasoningSourceSession)
	}
}

func TestApplyReasoningCommand_ResetClearsSessionState(t *testing.T) {
	state := SessionReasoningState{Effort: ReasoningEffortHigh, Source: ReasoningSourceSession}
	calls := 0
	persist := func(ReasoningEffort) error { calls++; return nil }

	newState, reply := ApplyReasoningCommand(
		state,
		ReasoningCommand{Action: ReasoningActionReset},
		persist,
	)

	if calls != 0 {
		t.Fatalf("reset called persistGlobal %d times, want 0", calls)
	}
	wantState := SessionReasoningState{Effort: ReasoningEffort(""), Source: ReasoningSourceUnset}
	if newState != wantState {
		t.Fatalf("newState = %+v, want %+v", newState, wantState)
	}
	if reply.Effort != ReasoningEffort("") || reply.Scope != ReasoningSourceUnset {
		t.Fatalf("reply = %+v, want effort=\"\" scope=%q", reply, ReasoningSourceUnset)
	}
	if reply.PersistFailed {
		t.Fatalf("reply.PersistFailed = true, want false")
	}
}

func TestParseReasoningCommand_ShowFormReturnsActionShow(t *testing.T) {
	cmd, err := ParseReasoningCommand(nil)
	if err != nil {
		t.Fatalf("ParseReasoningCommand(nil) err = %v, want nil", err)
	}
	if cmd.Action != ReasoningActionShow {
		t.Fatalf("Action = %v, want ReasoningActionShow", cmd.Action)
	}
	if cmd.Effort != ReasoningEffort("") {
		t.Fatalf("Effort = %q, want empty", cmd.Effort)
	}
	if cmd.Global {
		t.Fatalf("Global = true, want false")
	}

	cmd, err = ParseReasoningCommand([]string{})
	if err != nil {
		t.Fatalf("ParseReasoningCommand([]) err = %v, want nil", err)
	}
	if cmd.Action != ReasoningActionShow {
		t.Fatalf("Action = %v, want ReasoningActionShow", cmd.Action)
	}
}

func TestParseReasoningCommand_SetSessionScoped(t *testing.T) {
	for _, effort := range []string{"high", "low", "medium"} {
		t.Run(effort, func(t *testing.T) {
			cmd, err := ParseReasoningCommand([]string{effort})
			if err != nil {
				t.Fatalf("ParseReasoningCommand([%q]) err = %v, want nil", effort, err)
			}
			if cmd.Action != ReasoningActionSet {
				t.Fatalf("Action = %v, want ReasoningActionSet", cmd.Action)
			}
			if cmd.Effort != ReasoningEffort(effort) {
				t.Fatalf("Effort = %q, want %q", cmd.Effort, effort)
			}
			if cmd.Global {
				t.Fatalf("Global = true, want false")
			}
		})
	}
}

func TestParseReasoningCommand_SetGlobal(t *testing.T) {
	cmd, err := ParseReasoningCommand([]string{"low", "--global"})
	if err != nil {
		t.Fatalf("ParseReasoningCommand([low --global]) err = %v, want nil", err)
	}
	if cmd.Action != ReasoningActionSet {
		t.Fatalf("Action = %v, want ReasoningActionSet", cmd.Action)
	}
	if cmd.Effort != ReasoningEffort("low") {
		t.Fatalf("Effort = %q, want low", cmd.Effort)
	}
	if !cmd.Global {
		t.Fatalf("Global = false, want true")
	}
}

func TestParseReasoningCommand_ResetSession(t *testing.T) {
	cmd, err := ParseReasoningCommand([]string{"reset"})
	if err != nil {
		t.Fatalf("ParseReasoningCommand([reset]) err = %v, want nil", err)
	}
	if cmd.Action != ReasoningActionReset {
		t.Fatalf("Action = %v, want ReasoningActionReset", cmd.Action)
	}
	if cmd.Global {
		t.Fatalf("Global = true, want false")
	}
	if cmd.Effort != ReasoningEffort("") {
		t.Fatalf("Effort = %q, want empty", cmd.Effort)
	}
}

func TestParseReasoningCommand_RejectGlobalReset(t *testing.T) {
	_, err := ParseReasoningCommand([]string{"reset", "--global"})
	if err == nil {
		t.Fatalf("ParseReasoningCommand([reset --global]) err = nil, want ErrResetGlobalUnsupported")
	}
	if !errors.Is(err, ErrResetGlobalUnsupported) {
		t.Fatalf("err = %v, want ErrResetGlobalUnsupported", err)
	}
}

func TestParseReasoningCommand_RejectInvalidEffort(t *testing.T) {
	_, err := ParseReasoningCommand([]string{"bogus"})
	if err == nil {
		t.Fatalf("ParseReasoningCommand([bogus]) err = nil, want ErrInvalidEffort")
	}
	if !errors.Is(err, ErrInvalidEffort) {
		t.Fatalf("err = %v, want ErrInvalidEffort", err)
	}
}
