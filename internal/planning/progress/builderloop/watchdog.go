package builderloop

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/builderloop/watchdog"
)

type Verdict = watchdog.Verdict

const (
	VerdictHealthy = watchdog.VerdictHealthy
	VerdictSlow    = watchdog.VerdictSlow
	VerdictDead    = watchdog.VerdictDead
)

type WorkerVitals = watchdog.WorkerVitals

func Diagnose(now time.Time, v WorkerVitals, deadAfter, slowAfter time.Duration) Verdict {
	return watchdog.Diagnose(now, v, deadAfter, slowAfter)
}

type Decision = watchdog.Decision

const (
	DecisionFirst = watchdog.DecisionFirst
	DecisionAmend = watchdog.DecisionAmend
	DecisionNoop  = watchdog.DecisionNoop
)

type CheckpointState = watchdog.CheckpointState

type CoalesceConfig = watchdog.CoalesceConfig

func DecideCheckpoint(now time.Time, st CheckpointState, cfg CoalesceConfig) (Decision, CheckpointState) {
	return watchdog.DecideCheckpoint(now, st, cfg)
}
