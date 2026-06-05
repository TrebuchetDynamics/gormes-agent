// Package cron provides the scheduler, job store, and execution primitives for
// time-based automation inside gormes-agent. The paste sweep module offers a
// cron tick helper that runs the debug paste sweeper on an hourly cadence and a
// standalone SweepExpired wrapper for CLI-only opportunistic cleanup.
package cron

import (
	"context"
	"log/slog"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

// PasteSweepEvery is the default number of seconds between paste sweeps.
// 3600 seconds = 1 hour, matching Hermes behavior.
const PasteSweepEvery = 3600

// PasteSweepTickInterval is the cron tick interval in seconds.
// The paste sweep fires every PasteSweepEvery / PasteSweepTickInterval ticks.
const PasteSweepTickInterval = 60

// PasteSweepJobName is the canonical name used when registering the paste
// sweep as a cron Job via Store.Create.
const PasteSweepJobName = "paste_sweep"

// PasteSweepSchedule is the cron expression for hourly paste sweeps.
// Matches Hermes: sweep every 60 ticks at 60s interval = 1 hour.
const PasteSweepSchedule = "@every 1h"

// Sweeper defines the interface for paste sweep implementations.
// Both *cli.PasteSweeper and test fakes satisfy this interface.
type Sweeper interface {
	SweepExpired() (*cli.SweepResult, error)
}

// PasteSweepCronJob creates a cron.Job record for the paste sweep.
// The job's Prompt is empty because PasteSweepRun is a system housekeeping
// task rather than an agent prompt job. The job is stored via Store.Create
// before Scheduler.Start is called.
func PasteSweepCronJob() Job {
	return Job{
		ID:       newID(),
		Name:     PasteSweepJobName,
		Schedule: PasteSweepSchedule,
		Prompt:   "", // system job — not an agent prompt
		Paused:   false,
	}
}

// PasteSweepRun executes one paste sweep cycle and logs evidence.
// The sweeper may be nil in degraded mode (see contract).
func PasteSweepRun(ctx context.Context, log *slog.Logger, sweeper Sweeper) {
	if sweeper == nil {
		log.Warn("paste_sweep: sweeper unavailable — degraded mode")
		return
	}
	result, err := sweeper.SweepExpired()
	if err != nil {
		log.Warn("paste_sweep: sweep error", "err", err)
		return
	}
	if result.Deleted > 0 {
		log.Info("paste_sweep: deleted expired pastes",
			"deleted", result.Deleted,
			"remaining", result.Remaining)
	}
}

// PasteSweepCadenceTicks returns true when the given tickCount should trigger
// a paste sweep. Default: every PasteSweepEvery/PasteSweepTickInterval ticks.
// This matches Hermes behavior where _start_cron_ticker ticks every 60s and
// calls _sweep_expired_pastes every 60 ticks (1 hour).
func PasteSweepCadenceTicks(tickCount int) bool {
	interval := PasteSweepEvery / PasteSweepTickInterval
	return tickCount > 0 && tickCount%interval == 0
}

// RunPasteSweepTick runs one paste sweep cycle for use in a cron tick loop.
// It checks the current time against the sweep schedule and executes the
// sweeper if the cadence condition is met. The tickCount should increment
// each tick; SweepExpired fires when PasteSweepCadenceTicks returns true.
//
// Example integration with an external ticker:
//
//	for {
//	    tickCount++
//	    cron.RunPasteSweepTick(ctx, log, tickCount, sweeper)
//	    time.Sleep(PasteSweepTickInterval * time.Second)
//	}
func RunPasteSweepTick(ctx context.Context, log *slog.Logger, tickCount int, sweeper Sweeper) {
	if !PasteSweepCadenceTicks(tickCount) {
		return
	}
	PasteSweepRun(ctx, log, sweeper)
}

// PasteSweepCadenceDescription returns a human-readable description of the
// sweep cadence for debug-share status reporting.
func PasteSweepCadenceDescription() string {
	return "hourly (every 60 ticks at 60s interval)"
}
