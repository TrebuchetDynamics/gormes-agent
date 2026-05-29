package gateway

import (
	"errors"
	"log/slog"
	"testing"
)

func TestGatewayManagerDispatchesReasoning(t *testing.T) {
	var globalCalls []ReasoningEffort
	persist := func(e ReasoningEffort) error {
		globalCalls = append(globalCalls, e)
		return nil
	}

	m := NewManager(ManagerConfig{
		PersistReasoningGlobal: persist,
	}, nil, slog.Default())

	// Set session-only override on chat A.
	replyA, err := m.DispatchReasoning("telegram:42", []string{"high"})
	if err != nil {
		t.Fatalf("DispatchReasoning(A high) err = %v", err)
	}
	if replyA.Effort != ReasoningEffortHigh || replyA.Scope != ReasoningSourceSession {
		t.Fatalf("chat A set reply = %+v, want effort=high scope=session", replyA)
	}
	if len(globalCalls) != 0 {
		t.Fatalf("session-only set persisted globally: calls=%v", globalCalls)
	}

	// Show on chat A reflects the override.
	showA, err := m.DispatchReasoning("telegram:42", nil)
	if err != nil {
		t.Fatalf("DispatchReasoning(A show) err = %v", err)
	}
	if showA.Effort != ReasoningEffortHigh || showA.Scope != ReasoningSourceSession {
		t.Fatalf("chat A show = %+v, want effort=high scope=session", showA)
	}

	// Chat B is isolated — never touched, still unset.
	showB, err := m.DispatchReasoning("telegram:99", nil)
	if err != nil {
		t.Fatalf("DispatchReasoning(B show) err = %v", err)
	}
	if showB.Effort != ReasoningEffort("") || showB.Scope != ReasoningSourceUnset {
		t.Fatalf("chat B show = %+v, want effort=\"\" scope=unset (chat A leak)", showB)
	}

	// Global set on chat B calls the persist callback once with the requested effort.
	replyB, err := m.DispatchReasoning("telegram:99", []string{"low", "--global"})
	if err != nil {
		t.Fatalf("DispatchReasoning(B low --global) err = %v", err)
	}
	if replyB.Effort != ReasoningEffortLow || replyB.Scope != ReasoningSourceGlobal {
		t.Fatalf("chat B global reply = %+v, want effort=low scope=global", replyB)
	}
	if len(globalCalls) != 1 || globalCalls[0] != ReasoningEffortLow {
		t.Fatalf("persistGlobal calls = %v, want [low]", globalCalls)
	}

	// Chat A unaffected by chat B's global set — still session-scoped high.
	showAAfter, err := m.DispatchReasoning("telegram:42", nil)
	if err != nil {
		t.Fatalf("DispatchReasoning(A show after) err = %v", err)
	}
	if showAAfter.Effort != ReasoningEffortHigh || showAAfter.Scope != ReasoningSourceSession {
		t.Fatalf("chat A show after = %+v, want effort=high scope=session", showAAfter)
	}

	// Reset on chat A clears the session override.
	resetA, err := m.DispatchReasoning("telegram:42", []string{"reset"})
	if err != nil {
		t.Fatalf("DispatchReasoning(A reset) err = %v", err)
	}
	if resetA.Effort != ReasoningEffort("") || resetA.Scope != ReasoningSourceUnset {
		t.Fatalf("chat A reset = %+v, want effort=\"\" scope=unset", resetA)
	}

	// Invalid arg returns the parser error verbatim.
	if _, err := m.DispatchReasoning("telegram:42", []string{"bogus"}); !errors.Is(err, ErrInvalidEffort) {
		t.Fatalf("invalid effort err = %v, want ErrInvalidEffort", err)
	}
}

func TestGatewayManagerDispatchReasoningGlobalPersistFallback(t *testing.T) {
	persistErr := errors.New("config write failed")
	persist := func(ReasoningEffort) error { return persistErr }

	m := NewManager(ManagerConfig{
		PersistReasoningGlobal: persist,
	}, nil, slog.Default())

	reply, err := m.DispatchReasoning("telegram:42", []string{"medium", "--global"})
	if err != nil {
		t.Fatalf("DispatchReasoning err = %v, want nil (fallback should not propagate)", err)
	}
	if !reply.PersistFailed {
		t.Fatalf("reply.PersistFailed = false, want true on persistGlobal error")
	}
	if reply.Effort != ReasoningEffortMedium || reply.Scope != ReasoningSourceSession {
		t.Fatalf("fallback reply = %+v, want effort=medium scope=session", reply)
	}

	// Subsequent show reflects session fallback.
	show, err := m.DispatchReasoning("telegram:42", nil)
	if err != nil {
		t.Fatalf("DispatchReasoning(show) err = %v", err)
	}
	if show.Effort != ReasoningEffortMedium || show.Scope != ReasoningSourceSession {
		t.Fatalf("post-fallback show = %+v, want effort=medium scope=session", show)
	}
}
