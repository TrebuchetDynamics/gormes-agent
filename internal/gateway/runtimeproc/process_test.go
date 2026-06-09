package runtimeproc

import (
	"strings"
	"testing"
)

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
