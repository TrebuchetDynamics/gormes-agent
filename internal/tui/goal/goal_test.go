package goal

import (
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

func testController() (*Controller, *session.MemMap, time.Time) {
	store := session.NewMemMap()
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	return &Controller{Store: store, Now: func() time.Time { return now }, MaxTurns: 20}, store, now
}

func TestBuildLineActiveAndPaused(t *testing.T) {
	active := &session.GoalState{Goal: "ship parity", Status: session.GoalStatusActive, TurnsUsed: 3, MaxTurns: 20}
	line, ok := BuildLine(active)
	if !ok {
		t.Fatal("active goal produced no line")
	}
	if line.Glyph != "⊙" || line.Label != "goal" || line.Detail != "3/20 turns" || line.Title != "ship parity" {
		t.Fatalf("line = %+v", line)
	}
	if got := RenderBar(line, 80); got != "⊙ goal · 3/20 turns · ship parity" {
		t.Fatalf("bar = %q", got)
	}

	paused := &session.GoalState{Goal: "ship parity", Status: session.GoalStatusPaused, TurnsUsed: 4, MaxTurns: 20, PausedReason: "budget"}
	line, ok = BuildLine(paused)
	if !ok {
		t.Fatal("paused goal produced no line")
	}
	if line.Glyph != "⏸" || line.Label != "goal paused" || line.Detail != "budget · 4/20 turns" {
		t.Fatalf("paused line = %+v", line)
	}

	for _, status := range []session.GoalStatus{session.GoalStatusDone, session.GoalStatusCleared} {
		if _, ok := BuildLine(&session.GoalState{Goal: "x", Status: status}); ok {
			t.Fatalf("status %q produced a bar", status)
		}
	}
}

func TestRenderBarTruncates(t *testing.T) {
	line := Line{Glyph: "⊙", Label: "goal", Detail: "1/20 turns", Title: "a long objective"}
	got := RenderBar(line, 20)
	if !strings.HasPrefix(got, "⊙ goal") {
		t.Fatalf("bar = %q", got)
	}
	if len([]rune(got)) > 20 {
		t.Fatalf("bar width = %d, want <= 20: %q", len([]rune(got)), got)
	}
}

func TestControllerSetStatusPauseResumeClearDone(t *testing.T) {
	c, _, _ := testController()

	reply, err := c.Handle("s1", "/goal ship the port")
	if err != nil || !strings.Contains(reply, "Goal set (0/20): ship the port") {
		t.Fatalf("set reply = %q err=%v", reply, err)
	}
	status, err := c.Handle("s1", "/goal status")
	if err != nil || !strings.Contains(status, "Goal active (0/20): ship the port") {
		t.Fatalf("status reply = %q err=%v", status, err)
	}

	reply, _ = c.Handle("s1", "/goal pause waiting on review")
	if reply != "Goal paused." {
		t.Fatalf("pause reply = %q", reply)
	}
	status, _ = c.Handle("s1", "/goal")
	if !strings.Contains(status, "Paused reason: waiting on review") {
		t.Fatalf("paused status = %q", status)
	}

	reply, _ = c.Handle("s1", "/goal resume")
	if reply != "Goal resumed." {
		t.Fatalf("resume reply = %q", reply)
	}
	reply, _ = c.Handle("s1", "/goal done finished")
	if reply != "Goal marked done." {
		t.Fatalf("done reply = %q", reply)
	}
	if line, ok := c.Status("s1"); ok {
		t.Fatalf("done goal still renders a bar: %+v", line)
	}

	reply, _ = c.Handle("s1", "/goal clear")
	if reply != "Goal cleared." {
		t.Fatalf("clear reply = %q", reply)
	}
	if status, _ = c.Handle("s1", "/goal status"); status != "No goal set." {
		t.Fatalf("cleared status = %q", status)
	}
}

func TestControllerSubgoals(t *testing.T) {
	c, _, _ := testController()
	if reply, _ := c.Handle("s1", "/goal subgoal list"); reply != "No goal set." {
		t.Fatalf("subgoal without goal = %q", reply)
	}
	if reply, _ := c.Handle("s1", "/goal ship it"); !strings.Contains(reply, "Goal set") {
		t.Fatalf("set reply = %q", reply)
	}
	if reply, _ := c.Handle("s1", "/goal subgoal add tests pass"); reply != "Added subgoal 1." {
		t.Fatalf("add reply = %q", reply)
	}
	if reply, _ := c.Handle("s1", "/goal subgoal add docs pass"); reply != "Added subgoal 2." {
		t.Fatalf("add reply = %q", reply)
	}
	if reply, _ := c.Handle("s1", "/goal subgoal list"); !strings.Contains(reply, "2. docs pass") {
		t.Fatalf("list reply = %q", reply)
	}
	if reply, _ := c.Handle("s1", "/goal subgoal remove 1"); reply != "Removed subgoal 1." {
		t.Fatalf("remove reply = %q", reply)
	}
	if reply, _ := c.Handle("s1", "/goal subgoal list"); strings.Contains(reply, "tests pass") {
		t.Fatalf("removed subgoal still listed: %q", reply)
	}
	if reply, _ := c.Handle("s1", "/goal subgoal clear"); reply != "Cleared subgoals." {
		t.Fatalf("clear reply = %q", reply)
	}
}

func TestControllerSessionIsolation(t *testing.T) {
	c, _, _ := testController()
	if _, err := c.Handle("s1", "/goal first session goal"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Handle("s2", "/goal second session goal"); err != nil {
		t.Fatal(err)
	}
	one, _ := c.Handle("s1", "/goal status")
	two, _ := c.Handle("s2", "/goal status")
	if !strings.Contains(one, "first session goal") || !strings.Contains(two, "second session goal") {
		t.Fatalf("session goals leaked: s1=%q s2=%q", one, two)
	}
}

func TestControllerEmptySession(t *testing.T) {
	c, _, _ := testController()
	reply, err := c.Handle("", "/goal status")
	if err != nil || reply != "goal: session unavailable" {
		t.Fatalf("empty session reply = %q err=%v", reply, err)
	}
}
