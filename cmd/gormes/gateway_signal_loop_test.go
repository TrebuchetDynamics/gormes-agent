package main

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestGatewayWindowsDetachedCtrlCSignalPlan(t *testing.T) {
	t.Run("foreground windows keeps interrupt subscribed", func(t *testing.T) {
		restoreGOOS := gatewayRuntimeGOOSForTest(t, "windows")
		defer restoreGOOS()
		t.Setenv("GORMES_GATEWAY_DETACHED", "0")

		signals, absorbInterrupt := gatewayShutdownSignalPlan()

		if absorbInterrupt {
			t.Fatal("foreground Windows gateway must not absorb Ctrl+C")
		}
		if !hasGatewaySignal(signals, os.Interrupt) {
			t.Fatalf("foreground Windows signals = %v, want os.Interrupt", signals)
		}
	})

	t.Run("detached windows absorbs interrupt", func(t *testing.T) {
		restoreGOOS := gatewayRuntimeGOOSForTest(t, "windows")
		defer restoreGOOS()
		t.Setenv("GORMES_GATEWAY_DETACHED", "1")

		signals, absorbInterrupt := gatewayShutdownSignalPlan()

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
		restoreGOOS := gatewayRuntimeGOOSForTest(t, "linux")
		defer restoreGOOS()
		t.Setenv("GORMES_GATEWAY_DETACHED", "1")

		signals, absorbInterrupt := gatewayShutdownSignalPlan()

		if absorbInterrupt {
			t.Fatal("non-Windows gateway must not use the Windows detached interrupt absorber")
		}
		if !hasGatewaySignal(signals, os.Interrupt) {
			t.Fatalf("non-Windows signals = %v, want os.Interrupt", signals)
		}
	})
}

func TestGatewayStopSignalLoopConsumesPlannedMarker(t *testing.T) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	mgr := &fakeShutdownManager{
		called:  make(chan struct{}),
		release: make(chan struct{}),
	}
	consumed := false
	restoreConsumer := gatewayPlannedStopConsumerForTest(t, func(context.Context) (gateway.PlannedStopConsumeResult, error) {
		consumed = true
		return gateway.PlannedStopConsumeResult{Status: gateway.PlannedStopConsumeMatched, Matched: true}, nil
	})
	defer restoreConsumer()

	done := make(chan struct{})
	forceExit := make(chan int, 1)
	go func() {
		defer close(done)
		runGatewaySignalLoop(sigCh, 200*time.Millisecond, mgr, cancel, nil, func(code int) {
			forceExit <- code
		}, tools.TermuxWakeLockManager{})
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

func gatewayRuntimeGOOSForTest(t *testing.T, goos string) func() {
	t.Helper()
	previous := gatewayRuntimeGOOS
	gatewayRuntimeGOOS = goos
	return func() {
		gatewayRuntimeGOOS = previous
	}
}

func gatewayPlannedStopConsumerForTest(t *testing.T, consume func(context.Context) (gateway.PlannedStopConsumeResult, error)) func() {
	t.Helper()
	previous := consumeGatewayPlannedStopMarkerForSelf
	consumeGatewayPlannedStopMarkerForSelf = consume
	return func() {
		consumeGatewayPlannedStopMarkerForSelf = previous
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
