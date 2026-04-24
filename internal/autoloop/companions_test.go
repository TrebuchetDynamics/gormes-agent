package autoloop

import (
	"testing"
	"time"
)

func TestCompanionDuePlannerOnCadence(t *testing.T) {
	decision := CompanionDue(CompanionOptions{
		Name:         "planner",
		CurrentCycle: 8,
		EveryNCycles: 4,
		Now:          time.Unix(200, 0),
		LoopSleep:    time.Second,
	}, CompanionState{
		LastCycle: 4,
		LastEpoch: 190,
	})

	if !decision.Run {
		t.Fatal("Run = false, want true")
	}
	if decision.Reason != "cycle cadence reached" {
		t.Fatalf("Reason = %q, want cycle cadence reached", decision.Reason)
	}
}

func TestCompanionSkipsWhenDisabled(t *testing.T) {
	decision := CompanionDue(CompanionOptions{
		Name:         "planner",
		CurrentCycle: 10,
		EveryNCycles: 1,
		Now:          time.Unix(300, 0),
		Disabled:     true,
	}, CompanionState{
		LastCycle: 0,
		LastEpoch: 0,
	})

	if decision.Run {
		t.Fatal("Run = true, want false")
	}
	if decision.Reason != "disabled" {
		t.Fatalf("Reason = %q, want disabled", decision.Reason)
	}
}
