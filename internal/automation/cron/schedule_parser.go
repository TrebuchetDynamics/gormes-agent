package cron

import (
	"time"

	cronschedule "github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron/schedule"
)

// ErrInvalidSchedule marks operator schedule strings that cannot be parsed.
var ErrInvalidSchedule = cronschedule.ErrInvalidSchedule

type ScheduleKind = cronschedule.ScheduleKind

const (
	ScheduleKindOnce     = cronschedule.ScheduleKindOnce
	ScheduleKindInterval = cronschedule.ScheduleKindInterval
	ScheduleKindCron     = cronschedule.ScheduleKindCron
)

// ParsedSchedule is the pure read model for operator cron schedule strings.
type ParsedSchedule = cronschedule.ParsedSchedule

// CronUnavailableEvidence is stable degraded-mode evidence for schedule
// decisions that should skip one job without stopping the scheduler loop.
type CronUnavailableEvidence = cronschedule.CronUnavailableEvidence

// ScheduleParseError is a typed invalid-schedule error with unavailable
// evidence attached for future cron tool/API envelopes.
type ScheduleParseError = cronschedule.ScheduleParseError

// CronRunDecision reports what a scheduler read path should do for one job.
type CronRunDecision = cronschedule.CronRunDecision

// ParseCronSchedule parses Hermes-compatible operator schedule strings without
// touching stores, clocks, goroutines, or public cron tool handlers.
func ParseCronSchedule(input string, now time.Time) (ParsedSchedule, error) {
	return cronschedule.ParseCronSchedule(input, now)
}

// CronNextRunDecision reports whether a parsed schedule should run at now and
// the next due timestamp after now.
func CronNextRunDecision(parsed ParsedSchedule, lastRunUnix int64, repeatCompleted int, now time.Time) CronRunDecision {
	return cronschedule.CronNextRunDecision(parsed, lastRunUnix, repeatCompleted, now)
}

// NewUnavailableEvidence builds stable degraded-mode evidence for cron schedule decisions.
func NewUnavailableEvidence(code, message string) *CronUnavailableEvidence {
	return cronschedule.NewUnavailableEvidence(code, message)
}
