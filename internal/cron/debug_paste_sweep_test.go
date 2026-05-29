package cron

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

// fakePasteSweeper implements cron.Sweeper for test observation.
type fakePasteSweeper struct {
	result *cli.SweepResult
	err    error
}

func (f *fakePasteSweeper) SweepExpired() (*cli.SweepResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func TestPasteSweepRun_LogsDeletedCount(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, nil))
	sweeper := &fakePasteSweeper{
		result: &cli.SweepResult{Deleted: 3, Remaining: 7},
	}

	PasteSweepRun(context.Background(), log, sweeper)

	output := buf.String()
	if !strings.Contains(output, "deleted expired pastes") {
		t.Errorf("expected 'deleted expired pastes' in log, got: %s", output)
	}
	if !strings.Contains(output, `deleted=3`) {
		t.Errorf("expected deleted=3 in log, got: %s", output)
	}
}

func TestPasteSweepRun_NoSweeperWarns(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, nil))

	PasteSweepRun(context.Background(), log, nil)

	output := buf.String()
	if !strings.Contains(output, "sweeper unavailable") {
		t.Errorf("expected 'sweeper unavailable' warning, got: %s", output)
	}
}

func TestPasteSweepRun_ErrorLogs(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, nil))
	sweeper := &fakePasteSweeper{
		err: errors.New("store unavailable"),
	}

	PasteSweepRun(context.Background(), log, sweeper)

	output := buf.String()
	if !strings.Contains(output, "sweep error") {
		t.Errorf("expected 'sweep error' warning, got: %s", output)
	}
}

func TestPasteSweepRun_NoDeletedNoInfoLog(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, nil))
	sweeper := &fakePasteSweeper{
		result: &cli.SweepResult{Deleted: 0, Remaining: 5},
	}

	PasteSweepRun(context.Background(), log, sweeper)

	output := buf.String()
	// No "deleted expired pastes" when nothing was deleted.
	if strings.Contains(output, "deleted expired pastes") {
		t.Errorf("expected no 'deleted expired pastes' when nothing deleted, got: %s", output)
	}
}

func TestPasteSweepCadenceTicks_HourlyAtDefault(t *testing.T) {
	// PasteSweepEvery/PasteSweepTickInterval = 3600/60 = 60 ticks per hour.
	tests := []struct {
		tick int
		want bool
		desc string
	}{
		{1, false, "first tick not a sweep"},
		{59, false, "tick 59 not a sweep"},
		{60, true, "tick 60 is hourly"},
		{120, true, "tick 120 is 2 hours"},
		{61, false, "tick 61 not a sweep"},
		{180, true, "tick 180 is 3 hours"},
	}
	for _, tt := range tests {
		got := PasteSweepCadenceTicks(tt.tick)
		if got != tt.want {
			t.Errorf("PasteSweepCadenceTicks(%d) = %v, want %v (%s)",
				tt.tick, got, tt.want, tt.desc)
		}
	}
}

func TestPasteSweepCadenceTicks_ZeroTick(t *testing.T) {
	// tick=0 should not trigger.
	got := PasteSweepCadenceTicks(0)
	if got {
		t.Errorf("PasteSweepCadenceTicks(0) = %v, want false", got)
	}
}

func TestPasteSweepCadenceTicks_NegativeTick(t *testing.T) {
	got := PasteSweepCadenceTicks(-1)
	if got {
		t.Errorf("PasteSweepCadenceTicks(-1) = %v, want false", got)
	}
}

func TestRunPasteSweepTick_NotHourlySkips(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, nil))
	sweeper := &fakePasteSweeper{
		result: &cli.SweepResult{Deleted: 1, Remaining: 0},
	}

	// tick=30 is not hourly
	RunPasteSweepTick(context.Background(), log, 30, sweeper)

	output := buf.String()
	// Should not have called the sweeper (no log output)
	if output != "" {
		t.Errorf("expected no log calls for non-hourly tick, got: %s", output)
	}
}

func TestRunPasteSweepTick_HourlyFires(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, nil))
	sweeper := &fakePasteSweeper{
		result: &cli.SweepResult{Deleted: 2, Remaining: 3},
	}

	// tick=60 is hourly
	RunPasteSweepTick(context.Background(), log, 60, sweeper)

	output := buf.String()
	if !strings.Contains(output, "deleted expired pastes") {
		t.Errorf("expected 'deleted expired pastes' log at hourly tick, got: %s", output)
	}
}

func TestPasteSweepCronJob(t *testing.T) {
	job := PasteSweepCronJob()

	if job.Name != PasteSweepJobName {
		t.Errorf("job.Name = %q, want %q", job.Name, PasteSweepJobName)
	}
	if job.Schedule != PasteSweepSchedule {
		t.Errorf("job.Schedule = %q, want %q", job.Schedule, PasteSweepSchedule)
	}
	if job.ID == "" {
		t.Error("job.ID should not be empty")
	}
	if job.Paused {
		t.Error("job.Paused should be false")
	}
}

func TestPasteSweepCadenceDescription(t *testing.T) {
	desc := PasteSweepCadenceDescription()
	if !strings.Contains(desc, "hourly") {
		t.Errorf("cadence description = %q, want something containing 'hourly'", desc)
	}
}

// Verify that *cli.PasteSweeper satisfies the cron.Sweeper interface.
var _ Sweeper = (*cli.PasteSweeper)(nil)
