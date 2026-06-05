package session

import (
	"context"
	"testing"
	"time"
)

func TestGoalStatePersistsPerSession(t *testing.T) {
	ctx := context.Background()
	store := NewMemMap()
	now := time.Unix(1700000000, 0)

	if err := store.PutMetadata(ctx, Metadata{
		SessionID: "sess-one",
		Source:    "telegram",
		ChatID:    "42",
		UserID:    "u1",
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
	}); err != nil {
		t.Fatalf("PutMetadata: %v", err)
	}

	state, err := SetGoal(ctx, store, "sess-one", "ship the gateway loop", 3, now)
	if err != nil {
		t.Fatalf("SetGoal: %v", err)
	}
	if state.Status != GoalStatusActive || state.TurnsUsed != 0 || state.MaxTurns != 3 {
		t.Fatalf("initial goal = %+v, want active zero-turn budget 3", state)
	}

	reloaded, ok, err := LoadGoal(ctx, store, "sess-one")
	if err != nil {
		t.Fatalf("LoadGoal after set: %v", err)
	}
	if !ok {
		t.Fatal("LoadGoal after set ok=false, want persisted goal")
	}
	if reloaded.Goal != "ship the gateway loop" || reloaded.Status != GoalStatusActive {
		t.Fatalf("reloaded goal = %+v, want active ship goal", reloaded)
	}

	paused, err := PauseGoal(ctx, store, "sess-one", "user-paused", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("PauseGoal: %v", err)
	}
	if paused.Status != GoalStatusPaused || paused.PausedReason != "user-paused" {
		t.Fatalf("paused goal = %+v, want paused user-paused", paused)
	}

	resumed, err := ResumeGoal(ctx, store, "sess-one", true, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("ResumeGoal: %v", err)
	}
	if resumed.Status != GoalStatusActive || resumed.PausedReason != "" || resumed.TurnsUsed != 0 {
		t.Fatalf("resumed goal = %+v, want active with reset budget", resumed)
	}

	done, err := DoneGoal(ctx, store, "sess-one", "finished", now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("DoneGoal: %v", err)
	}
	if done.Status != GoalStatusDone || done.LastVerdict != "done" || done.LastReason != "finished" {
		t.Fatalf("done goal = %+v, want done verdict and reason", done)
	}

	cleared, err := ClearGoal(ctx, store, "sess-one", now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("ClearGoal: %v", err)
	}
	if cleared.Status != GoalStatusCleared {
		t.Fatalf("cleared goal status = %q, want %q", cleared.Status, GoalStatusCleared)
	}

	secondManagerView, ok, err := LoadGoal(ctx, store, "sess-one")
	if err != nil {
		t.Fatalf("LoadGoal second manager: %v", err)
	}
	if !ok {
		t.Fatal("LoadGoal second manager ok=false, want cleared state to round-trip")
	}
	if secondManagerView.Status != GoalStatusCleared || secondManagerView.Goal != "ship the gateway loop" {
		t.Fatalf("second manager goal = %+v, want cleared original goal", secondManagerView)
	}

	other, err := SetGoal(ctx, store, "sess-two", "different session goal", 0, now)
	if err != nil {
		t.Fatalf("SetGoal other session: %v", err)
	}
	if other.MaxTurns != DefaultGoalMaxTurns {
		t.Fatalf("default max turns = %d, want %d", other.MaxTurns, DefaultGoalMaxTurns)
	}
	if leaked, ok, err := LoadGoal(ctx, store, "sess-one"); err != nil {
		t.Fatalf("LoadGoal first session after other set: %v", err)
	} else if !ok || leaked.Goal != "ship the gateway loop" {
		t.Fatalf("first session goal changed to %+v, want original goal isolated", leaked)
	}
}
