package runtimeproc

import (
	"strings"
	"testing"
)

func TestParseProcStatIdentityRejectsUnknownState(t *testing.T) {
	stat := "123 (gormes gateway) ? " + strings.Repeat("1 ", 19) + "42 43"
	if got, state, ok := parseProcStatIdentity(stat); ok {
		t.Fatalf("parseProcStatIdentity unknown state = (%d, %q, true), want rejected", got, state)
	}
}

func TestParseProcStatIdentityAcceptsLowercaseZombieState(t *testing.T) {
	stat := "123 (gormes gateway) z " + strings.Repeat("1 ", 18) + "42 43"
	got, state, ok := parseProcStatIdentity(stat)
	if !ok || got != 42 || state != "z" {
		t.Fatalf("parseProcStatIdentity lowercase zombie = (%d, %q, %v), want (42, z, true)", got, state, ok)
	}
}

func TestParseProcStatIdentityRejectsMultiCharacterState(t *testing.T) {
	stat := "123 (gormes gateway) SS " + strings.Repeat("1 ", 19) + "42 43"
	if got, state, ok := parseProcStatIdentity(stat); ok {
		t.Fatalf("parseProcStatIdentity multi-character state = (%d, %q, true), want rejected", got, state)
	}
}

func TestParseProcStatIdentityRejectsMissingStateSeparator(t *testing.T) {
	stat := "123 (gormes gateway)S " + strings.Repeat("1 ", 19) + "42 43"
	if got, state, ok := parseProcStatIdentity(stat); ok {
		t.Fatalf("parseProcStatIdentity missing separator = (%d, %q, true), want rejected", got, state)
	}
}

func TestParseProcStatIdentityRejectsNonPositiveStartTime(t *testing.T) {
	for _, startTime := range []string{"0", "-42"} {
		stat := "123 (gormes gateway) S " + strings.Repeat("1 ", 18) + startTime + " 21 22"
		if got, state, ok := parseProcStatIdentity(stat); ok {
			t.Fatalf("parseProcStatIdentity start=%s = (%d, %q, true), want rejected", startTime, got, state)
		}
	}
}

func TestStoppedProcStateIncludesZombie(t *testing.T) {
	for _, state := range []string{"T", "t", "Z", "z", "X", "x"} {
		if !stoppedProcState(state) {
			t.Fatalf("stoppedProcState(%q) = false, want true for non-live terminal/stopped process", state)
		}
	}
	for _, state := range []string{"R", "S", "D", "I"} {
		if stoppedProcState(state) {
			t.Fatalf("stoppedProcState(%q) = true, want false for live/sleeping process", state)
		}
	}
}
