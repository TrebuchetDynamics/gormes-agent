package cron

import (
	"strings"
	"testing"
	"time"
)

func TestCronRunCompletion_RecurringCronNextRunUnavailablePreservesActiveStatus(t *testing.T) {
	now := time.Date(2026, 4, 27, 9, 0, 0, 0, time.UTC)
	job := Job{
		ID:              "cron-job",
		Schedule:        "0 9 * * *",
		RepeatCompleted: 4,
	}
	run := Run{
		JobID:      job.ID,
		StartedAt:  now.Unix(),
		FinishedAt: now.Add(3 * time.Second).Unix(),
		Status:     "success",
	}
	parsed := ParsedSchedule{
		Kind:    ScheduleKindCron,
		Display: "0 9 * * *",
		Expr:    "0 9 * * *",
	}

	got := CronRunCompletion(job, run, parsed, now, func(decisionParsed ParsedSchedule, lastRunUnix int64, repeatCompleted int, decisionNow time.Time) CronRunDecision {
		if decisionParsed.Kind != ScheduleKindCron {
			t.Fatalf("decision parsed kind = %q, want cron", decisionParsed.Kind)
		}
		if lastRunUnix != run.StartedAt {
			t.Fatalf("decision lastRunUnix = %d, want %d", lastRunUnix, run.StartedAt)
		}
		if repeatCompleted != 5 {
			t.Fatalf("decision repeatCompleted = %d, want 5", repeatCompleted)
		}
		if !decisionNow.Equal(now) {
			t.Fatalf("decision now = %s, want %s", decisionNow, now)
		}
		return CronRunDecision{
			Unavailable: &CronUnavailableEvidence{
				Code:    "croniter_missing",
				Message: "cron iterator is unavailable",
			},
		}
	})

	if got.Job.Paused {
		t.Fatal("Paused = true, want recurring job to remain active")
	}
	if got.Job.LastRunUnix != run.StartedAt {
		t.Fatalf("LastRunUnix = %d, want %d", got.Job.LastRunUnix, run.StartedAt)
	}
	if got.Job.LastStatus != "error" {
		t.Fatalf("LastStatus = %q, want error", got.Job.LastStatus)
	}
	if got.Job.RepeatCompleted != 5 {
		t.Fatalf("RepeatCompleted = %d, want 5", got.Job.RepeatCompleted)
	}
	if got.Evidence == nil || got.Evidence.Code != "cron_next_run_unavailable" {
		t.Fatalf("Evidence = %+v, want cron_next_run_unavailable", got.Evidence)
	}
	if got.Terminal {
		t.Fatal("Terminal = true, want recurring next-run failure to stay non-terminal")
	}
	if got.Run.Status != "error" {
		t.Fatalf("run Status = %q, want error", got.Run.Status)
	}
	if !strings.Contains(got.Run.ErrorMsg, "cron_next_run_unavailable") ||
		!strings.Contains(got.Run.ErrorMsg, "croniter_missing") {
		t.Fatalf("run ErrorMsg = %q, want typed next-run evidence", got.Run.ErrorMsg)
	}
}

func TestCronRunCompletion_RecurringIntervalNextRunUnavailablePreservesActiveStatus(t *testing.T) {
	now := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	job := Job{ID: "interval-job", Schedule: "every 30m"}
	run := Run{
		JobID:      job.ID,
		StartedAt:  now.Unix(),
		FinishedAt: now.Add(time.Second).Unix(),
		Status:     "success",
	}
	parsed := ParsedSchedule{
		Kind:    ScheduleKindInterval,
		Display: "every 30m",
		Minutes: 30,
	}

	got := CronRunCompletion(job, run, parsed, now, func(ParsedSchedule, int64, int, time.Time) CronRunDecision {
		return CronRunDecision{Runnable: true}
	})

	if got.Job.Paused {
		t.Fatal("Paused = true, want interval job to remain active")
	}
	if got.Job.LastStatus != "error" {
		t.Fatalf("LastStatus = %q, want error", got.Job.LastStatus)
	}
	if got.Job.RepeatCompleted != 1 {
		t.Fatalf("RepeatCompleted = %d, want 1", got.Job.RepeatCompleted)
	}
	if got.Evidence == nil || got.Evidence.Code != "cron_next_run_unavailable" {
		t.Fatalf("Evidence = %+v, want cron_next_run_unavailable", got.Evidence)
	}
	if got.Terminal {
		t.Fatal("Terminal = true, want recurring interval compute failure to stay non-terminal")
	}
	if !strings.Contains(got.Run.ErrorMsg, "cron_next_run_unavailable") {
		t.Fatalf("run ErrorMsg = %q, want cron_next_run_unavailable evidence", got.Run.ErrorMsg)
	}
}

func TestCronRunCompletion_OneShotNoNextRunBecomesTerminal(t *testing.T) {
	now := time.Date(2026, 4, 27, 11, 0, 0, 0, time.UTC)
	job := Job{ID: "one-shot", Schedule: "2026-04-27T11:00:00Z"}
	run := Run{
		JobID:      job.ID,
		StartedAt:  now.Unix(),
		FinishedAt: now.Add(time.Second).Unix(),
		Status:     "success",
	}
	parsed := ParsedSchedule{
		Kind:    ScheduleKindOnce,
		Display: "2026-04-27T11:00:00Z",
		RunAt:   now,
		Repeat:  1,
	}

	got := CronRunCompletion(job, run, parsed, now, func(ParsedSchedule, int64, int, time.Time) CronRunDecision {
		return CronRunDecision{
			Unavailable: &CronUnavailableEvidence{
				Code:    "oneshot_completed",
				Message: "one-shot schedule has no next run",
			},
		}
	})

	if !got.Terminal {
		t.Fatal("Terminal = false, want one-shot without next run to become terminal")
	}
	if !got.Job.Paused {
		t.Fatal("Paused = false, want terminal one-shot to be excluded from future scheduler loads")
	}
	if got.Job.LastStatus != "completed" {
		t.Fatalf("LastStatus = %q, want completed", got.Job.LastStatus)
	}
	if got.Run.Status != "success" {
		t.Fatalf("run Status = %q, want original success preserved", got.Run.Status)
	}
	if got.Evidence == nil || got.Evidence.Code != "oneshot_completed" {
		t.Fatalf("Evidence = %+v, want oneshot_completed", got.Evidence)
	}
	if !strings.Contains(got.Run.ErrorMsg, "oneshot_completed") {
		t.Fatalf("run ErrorMsg = %q, want terminal evidence", got.Run.ErrorMsg)
	}
}

func TestCronRunCompletion_FiniteRepeatExhaustionBecomesTerminal(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	parsed := ParsedSchedule{
		Kind:    ScheduleKindInterval,
		Display: "every 30m",
		Minutes: 30,
		Repeat:  3,
	}
	run := Run{
		JobID:      "finite",
		StartedAt:  now.Unix(),
		FinishedAt: now.Add(time.Second).Unix(),
		Status:     "success",
	}

	exhausted := CronRunCompletion(Job{
		ID:              "finite",
		Schedule:        "every 30m",
		Repeat:          3,
		RepeatCompleted: 2,
	}, run, parsed, now, CronNextRunDecision)
	if !exhausted.Terminal {
		t.Fatal("Terminal = false, want third finite repeat to become terminal")
	}
	if !exhausted.Job.Paused {
		t.Fatal("Paused = false, want exhausted finite repeat to be excluded from future scheduler loads")
	}
	if exhausted.Job.RepeatCompleted != 3 {
		t.Fatalf("RepeatCompleted = %d, want 3", exhausted.Job.RepeatCompleted)
	}
	if exhausted.Job.LastStatus != "completed" {
		t.Fatalf("LastStatus = %q, want completed", exhausted.Job.LastStatus)
	}
	if exhausted.Evidence == nil || exhausted.Evidence.Code != "repeat_exhausted" {
		t.Fatalf("Evidence = %+v, want repeat_exhausted", exhausted.Evidence)
	}

	remaining := CronRunCompletion(Job{
		ID:              "finite",
		Schedule:        "every 30m",
		Repeat:          3,
		RepeatCompleted: 1,
	}, run, parsed, now, CronNextRunDecision)
	if remaining.Terminal {
		t.Fatal("Terminal = true, want second of three finite repeats to remain active")
	}
	if remaining.Job.Paused {
		t.Fatal("Paused = true, want second of three finite repeats to remain active")
	}
	if remaining.Job.RepeatCompleted != 2 {
		t.Fatalf("RepeatCompleted = %d, want 2", remaining.Job.RepeatCompleted)
	}
	if remaining.Job.LastStatus != "success" {
		t.Fatalf("LastStatus = %q, want success", remaining.Job.LastStatus)
	}
}
