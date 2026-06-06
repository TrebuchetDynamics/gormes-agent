package gateway

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type fakeSignalShutdownManager struct {
	called  chan struct{}
	release chan struct{}
}

func (f *fakeSignalShutdownManager) Shutdown(context.Context) error {
	close(f.called)
	<-f.release
	return nil
}

type fakeSignalReloadShutdownManager struct {
	*fakeSignalShutdownManager
	reloads   chan struct{}
	reloadErr error
}

func (f *fakeSignalReloadShutdownManager) Reload(context.Context) error {
	f.reloads <- struct{}{}
	return f.reloadErr
}

func TestShutdownSignalPlanWindowsDetachedCtrlC(t *testing.T) {
	t.Run("foreground windows keeps interrupt subscribed", func(t *testing.T) {
		signals, absorbInterrupt := ShutdownSignalPlan("windows", mapGatewaySignalEnv(map[string]string{GatewayDetachedEnvName: "0"}))

		if absorbInterrupt {
			t.Fatal("foreground Windows gateway must not absorb Ctrl+C")
		}
		if !hasGatewaySignal(signals, os.Interrupt) {
			t.Fatalf("foreground Windows signals = %v, want os.Interrupt", signals)
		}
	})

	t.Run("detached windows absorbs interrupt", func(t *testing.T) {
		signals, absorbInterrupt := ShutdownSignalPlan("windows", mapGatewaySignalEnv(map[string]string{GatewayDetachedEnvName: "1"}))

		if !absorbInterrupt {
			t.Fatal("detached Windows gateway must absorb Ctrl+C broadcasts")
		}
		if hasGatewaySignal(signals, os.Interrupt) {
			t.Fatalf("detached Windows signals = %v, must omit os.Interrupt", signals)
		}
		if !hasGatewaySignal(signals, syscall.SIGTERM) || !hasGatewaySignal(signals, syscall.SIGHUP) {
			t.Fatalf("detached Windows signals = %v, want SIGTERM and SIGHUP", signals)
		}
	})

	t.Run("non-windows keeps interrupt even with detached env", func(t *testing.T) {
		signals, absorbInterrupt := ShutdownSignalPlan("linux", mapGatewaySignalEnv(map[string]string{GatewayDetachedEnvName: "1"}))

		if absorbInterrupt {
			t.Fatal("non-Windows gateway must not use the Windows detached interrupt absorber")
		}
		if !hasGatewaySignal(signals, os.Interrupt) {
			t.Fatalf("non-Windows signals = %v, want os.Interrupt", signals)
		}
	})
}

func TestSignalLoopConsumesPlannedStopMarker(t *testing.T) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	mgr := &fakeSignalShutdownManager{
		called:  make(chan struct{}),
		release: make(chan struct{}),
	}
	consumed := false

	done := make(chan struct{})
	forceExit := make(chan int, 1)
	go func() {
		defer close(done)
		RunSignalLoop(SignalLoopOptions{
			Signals:         sigCh,
			Budget:          200 * time.Millisecond,
			Manager:         mgr,
			Cancel:          cancel,
			ForceExit:       func(code int) { forceExit <- code },
			WakeLockManager: tools.TermuxWakeLockManager{},
			ConsumePlannedStopMarker: func(context.Context) (PlannedStopConsumeResult, error) {
				consumed = true
				return PlannedStopConsumeResult{Status: PlannedStopConsumeMatched, Matched: true}, nil
			},
		})
	}()

	sigCh <- syscall.SIGTERM
	select {
	case <-mgr.called:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Shutdown was not called after planned SIGTERM")
	}
	close(mgr.release)
	select {
	case <-rootCtx.Done():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("root context not canceled after planned shutdown")
	}
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("signal loop did not return")
	}
	if !consumed {
		t.Fatal("planned stop marker consumer was not called for SIGTERM")
	}
	select {
	case code := <-forceExit:
		t.Fatalf("planned stop forced exit code %d", code)
	default:
	}
}

func TestSignalLoopDrainsBeforeCancel(t *testing.T) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	mgr := &fakeSignalShutdownManager{
		called:  make(chan struct{}),
		release: make(chan struct{}),
	}

	done := make(chan struct{})
	forceExit := make(chan int, 1)
	go func() {
		defer close(done)
		RunSignalLoop(SignalLoopOptions{
			Signals:         sigCh,
			Budget:          200 * time.Millisecond,
			Manager:         mgr,
			Cancel:          cancel,
			Log:             slog.Default(),
			ForceExit:       func(code int) { forceExit <- code },
			WakeLockManager: tools.TermuxWakeLockManager{},
		})
	}()

	sigCh <- syscall.SIGTERM

	select {
	case <-mgr.called:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Shutdown was not called after signal")
	}

	select {
	case <-rootCtx.Done():
		t.Fatal("root context canceled before shutdown drain completed")
	default:
	}

	close(mgr.release)

	select {
	case <-rootCtx.Done():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("root context not canceled after shutdown drain completed")
	}

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("signal loop did not return")
	}

	select {
	case code := <-forceExit:
		t.Fatalf("unexpected force exit: %d", code)
	default:
	}
}

func TestSignalLoopReloadsOnSIGHUPWithoutCancel(t *testing.T) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 2)
	mgr := &fakeSignalReloadShutdownManager{
		fakeSignalShutdownManager: &fakeSignalShutdownManager{
			called:  make(chan struct{}),
			release: make(chan struct{}),
		},
		reloads: make(chan struct{}, 1),
	}

	done := make(chan struct{})
	forceExit := make(chan int, 1)
	go func() {
		defer close(done)
		RunSignalLoop(SignalLoopOptions{
			Signals:         sigCh,
			Budget:          200 * time.Millisecond,
			Manager:         mgr,
			Cancel:          cancel,
			Log:             slog.Default(),
			ForceExit:       func(code int) { forceExit <- code },
			WakeLockManager: tools.TermuxWakeLockManager{},
		})
	}()

	sigCh <- syscall.SIGHUP

	select {
	case <-mgr.reloads:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Reload was not called after SIGHUP")
	}

	select {
	case <-rootCtx.Done():
		t.Fatal("root context canceled after reload signal")
	default:
	}

	select {
	case <-mgr.called:
		t.Fatal("Shutdown was called for reload signal")
	default:
	}

	sigCh <- syscall.SIGTERM
	select {
	case <-mgr.called:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Shutdown was not called after SIGTERM")
	}
	close(mgr.release)

	select {
	case <-rootCtx.Done():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("root context not canceled after shutdown signal")
	}
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("signal loop did not return")
	}
	select {
	case code := <-forceExit:
		t.Fatalf("unexpected force exit: %d", code)
	default:
	}
}

func TestSignalLoopDoesNotLogReloadFailureSecrets(t *testing.T) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 2)
	mgr := &fakeSignalReloadShutdownManager{
		fakeSignalShutdownManager: &fakeSignalShutdownManager{
			called:  make(chan struct{}),
			release: make(chan struct{}),
		},
		reloads:   make(chan struct{}, 1),
		reloadErr: errors.New("parse config.toml: api_key=plain-secret-token"),
	}
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))

	done := make(chan struct{})
	go func() {
		defer close(done)
		RunSignalLoop(SignalLoopOptions{
			Signals:         sigCh,
			Budget:          200 * time.Millisecond,
			Manager:         mgr,
			Cancel:          cancel,
			Log:             log,
			ForceExit:       func(int) {},
			WakeLockManager: tools.TermuxWakeLockManager{},
		})
	}()

	sigCh <- syscall.SIGHUP
	select {
	case <-mgr.reloads:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Reload was not called after SIGHUP")
	}
	logText := logs.String()
	if bytes.Contains([]byte(logText), []byte("plain-secret-token")) || bytes.Contains([]byte(logText), []byte("api_key")) {
		t.Fatalf("reload failure log leaked secret material:\n%s", logText)
	}
	select {
	case <-rootCtx.Done():
		t.Fatal("root context canceled after failed reload signal")
	default:
	}

	sigCh <- syscall.SIGTERM
	select {
	case <-mgr.called:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Shutdown was not called after SIGTERM")
	}
	close(mgr.release)
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("signal loop did not return")
	}
}

func mapGatewaySignalEnv(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func hasGatewaySignal(signals []os.Signal, want os.Signal) bool {
	for _, sig := range signals {
		if sig == want {
			return true
		}
	}
	return false
}
