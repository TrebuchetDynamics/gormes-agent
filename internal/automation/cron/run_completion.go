package cron

import (
	"fmt"
	"time"
)

// CronNextRunDecisionFunc is the injectable next-run decision used by the
// pure run-completion helper.
type CronNextRunDecisionFunc func(ParsedSchedule, int64, int, time.Time) CronRunDecision

// CronRunCompletionState is the post-run state that should be persisted after
// one executor fire.
type CronRunCompletionState struct {
	Job      Job
	Run      Run
	Decision CronRunDecision
	Evidence *CronUnavailableEvidence
	Terminal bool
}

// CronRunCompletion updates a job/run pair after one executor fire without
// touching stores, goroutines, scheduler ticks, or wall-clock time.
func CronRunCompletion(job Job, run Run, parsed ParsedSchedule, now time.Time, next CronNextRunDecisionFunc) CronRunCompletionState {
	if next == nil {
		next = CronNextRunDecision
	}
	job.LastRunUnix = run.StartedAt
	job.LastStatus = run.Status
	if job.RepeatCompleted < 0 {
		job.RepeatCompleted = 0
	}
	job.RepeatCompleted++
	if job.Repeat > 0 {
		parsed.Repeat = job.Repeat
	}

	decision := next(parsed, job.LastRunUnix, job.RepeatCompleted, now)
	state := CronRunCompletionState{
		Job:      job,
		Run:      run,
		Decision: decision,
	}

	if decision.Exhausted {
		return state.withTerminalEvidence(evidenceOrDefault(decision.Unavailable, "repeat_exhausted", "repeat limit exhausted"))
	}

	if parsed.Kind == ScheduleKindOnce && oneShotCompletionTerminal(decision, now) {
		return state.withTerminalEvidence(evidenceOrDefault(decision.Unavailable, "oneshot_completed", "one-shot schedule has no next run"))
	}

	if isRecurringScheduleKind(parsed.Kind) && recurringNextRunUnavailable(decision, now) {
		evidence := recurringNextRunUnavailableEvidence(parsed, decision)
		state.Evidence = evidence
		state.Job.LastStatus = "error"
		state.Run.Status = "error"
		state.Run.ErrorMsg = appendRunCompletionEvidence(state.Run.ErrorMsg, evidence, decision.Unavailable)
		return state
	}

	return state
}

func cronRunCompletionForJob(job Job, run Run, now time.Time, next CronNextRunDecisionFunc) CronRunCompletionState {
	parsed, err := ParseCronSchedule(job.Schedule, now)
	if err != nil {
		return baselineRunCompletion(job, run)
	}
	return CronRunCompletion(job, run, parsed, now, next)
}

func baselineRunCompletion(job Job, run Run) CronRunCompletionState {
	job.LastRunUnix = run.StartedAt
	job.LastStatus = run.Status
	if job.RepeatCompleted < 0 {
		job.RepeatCompleted = 0
	}
	job.RepeatCompleted++
	return CronRunCompletionState{
		Job: job,
		Run: run,
	}
}

func (s CronRunCompletionState) withTerminalEvidence(evidence *CronUnavailableEvidence) CronRunCompletionState {
	s.Terminal = true
	s.Evidence = evidence
	s.Job.Paused = true
	s.Job.LastStatus = "completed"
	s.Run.ErrorMsg = appendRunCompletionEvidence(s.Run.ErrorMsg, evidence, nil)
	return s
}

func isRecurringScheduleKind(kind ScheduleKind) bool {
	return kind == ScheduleKindCron || kind == ScheduleKindInterval
}

func recurringNextRunUnavailable(decision CronRunDecision, now time.Time) bool {
	if decision.Unavailable != nil {
		return true
	}
	if decision.NextRun.IsZero() {
		return true
	}
	return !decision.NextRun.After(now)
}

func oneShotCompletionTerminal(decision CronRunDecision, now time.Time) bool {
	if decision.Unavailable != nil {
		return true
	}
	if decision.NextRun.IsZero() {
		return true
	}
	return !decision.NextRun.After(now)
}

func recurringNextRunUnavailableEvidence(parsed ParsedSchedule, decision CronRunDecision) *CronUnavailableEvidence {
	detail := "next-run decision returned no future time"
	if decision.Unavailable != nil {
		switch {
		case decision.Unavailable.Message != "":
			detail = decision.Unavailable.Message
		case decision.Unavailable.Code != "":
			detail = decision.Unavailable.Code
		}
	}
	display := parsed.Display
	if display == "" {
		display = parsed.Expr
	}
	if display == "" {
		display = string(parsed.Kind)
	}
	return unavailableEvidence(
		"cron_next_run_unavailable",
		fmt.Sprintf("next run unavailable for %s schedule %q: %s", parsed.Kind, display, detail),
	)
}

func evidenceOrDefault(evidence *CronUnavailableEvidence, code, message string) *CronUnavailableEvidence {
	if evidence != nil {
		return evidence
	}
	return unavailableEvidence(code, message)
}

func appendRunCompletionEvidence(existing string, evidence, cause *CronUnavailableEvidence) string {
	if evidence == nil {
		return existing
	}
	text := evidence.Code
	if evidence.Message != "" {
		text = fmt.Sprintf("%s: %s", evidence.Code, evidence.Message)
	}
	if cause != nil && cause.Code != "" && cause.Code != evidence.Code {
		if cause.Message != "" {
			text = fmt.Sprintf("%s (cause=%s: %s)", text, cause.Code, cause.Message)
		} else {
			text = fmt.Sprintf("%s (cause=%s)", text, cause.Code)
		}
	}
	if existing == "" {
		return text
	}
	return existing + "; " + text
}
