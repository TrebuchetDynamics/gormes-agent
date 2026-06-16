package gateway

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const GatewayDetachedEnvName = "GORMES_GATEWAY_DETACHED"

type ShutdownManager interface {
	Shutdown(context.Context) error
}

type ReloadManager interface {
	Reload(context.Context) error
}

type PlannedStopConsumer func(context.Context) (PlannedStopConsumeResult, error)

type SignalLoopOptions struct {
	Signals                  <-chan os.Signal
	Budget                   time.Duration
	Manager                  ShutdownManager
	Cancel                   context.CancelFunc
	Log                      *slog.Logger
	ForceExit                func(int)
	WakeLockManager          tools.TermuxWakeLockManager
	ConsumePlannedStopMarker PlannedStopConsumer
}

func RunSignalLoop(opts SignalLoopOptions) {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	forceExit := opts.ForceExit
	if forceExit == nil {
		forceExit = os.Exit
	}
	cancel := opts.Cancel
	if cancel == nil {
		cancel = func() {}
	}

	for {
		sig, ok := <-opts.Signals
		if !ok {
			return
		}
		if sig == syscall.SIGHUP {
			reloader, ok := opts.Manager.(ReloadManager)
			if !ok {
				log.Warn("gateway config reload unavailable", "signal", sig.String())
				continue
			}
			if err := reloader.Reload(context.Background()); err != nil {
				log.Warn("gateway config reload failed; continuing with last good config")
			} else {
				log.Info("gateway config reloaded", "signal", sig.String())
			}
			continue
		}
		plannedStop, plannedStopStatus := classifyShutdownSignal(sig, opts.ConsumePlannedStopMarker)
		if plannedStop {
			log.Info("gateway shutdown requested", "signal", sig.String(), "planned_stop", true, "planned_stop_status", plannedStopStatus)
		} else {
			log.Warn("gateway shutdown requested", "signal", sig.String(), "planned_stop", false, "exit_class", "unexpected_signal_restartable", "planned_stop_status", plannedStopStatus)
		}

		timer := time.AfterFunc(opts.Budget, func() {
			log.Error("shutdown budget exceeded; forcing exit")
			forceExit(3)
		})
		defer timer.Stop()

		if err := opts.WakeLockManager.Release(context.Background()); err != nil {
			log.Warn("termux-wake-lock release failed", "err", err)
		} else {
			log.Info("termux-wake-lock released")
		}
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), opts.Budget)
		err := opts.Manager.Shutdown(shutdownCtx)
		shutdownCancel()
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			log.Warn("gateway shutdown drain", "err", err)
		} else if err != nil {
			log.Warn("gateway shutdown drain", "err", err)
		}

		cancel()
		return
	}
}

func ShutdownSignalPlan(goos string, lookupEnv func(string) string) ([]os.Signal, bool) {
	if lookupEnv == nil {
		lookupEnv = os.Getenv
	}
	absorbInterrupt := goos == "windows" && truthyEnv(lookupEnv(GatewayDetachedEnvName))
	signals := []os.Signal{syscall.SIGTERM, syscall.SIGHUP}
	if !absorbInterrupt {
		signals = append([]os.Signal{os.Interrupt}, signals...)
	}
	return signals, absorbInterrupt
}

func classifyShutdownSignal(sig os.Signal, consume PlannedStopConsumer) (bool, PlannedStopConsumeStatus) {
	if sig == os.Interrupt {
		return true, PlannedStopConsumeMatched
	}
	if sig != syscall.SIGTERM {
		return false, ""
	}
	if consume == nil {
		return false, ""
	}
	result, err := consume(context.Background())
	if err != nil {
		return false, PlannedStopConsumeInvalid
	}
	return result.Matched, result.Status
}

func truthyEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
