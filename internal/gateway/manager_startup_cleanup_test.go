package gateway

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

// startupCloseChannel is a gateway.Channel fixture that reports a fixed
// Run error, implements StartupCloser, and records Close call counts.
type startupCloseChannel struct {
	name     string
	runErr   error
	closeErr error

	mu     sync.Mutex
	closes int
}

func (c *startupCloseChannel) Name() string { return c.name }

func (c *startupCloseChannel) Run(ctx context.Context, _ chan<- InboundEvent) error {
	if c.runErr != nil {
		return c.runErr
	}
	<-ctx.Done()
	return nil
}

func (c *startupCloseChannel) Send(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

// Close implements StartupCloser.
func (c *startupCloseChannel) Close(_ context.Context) error {
	c.mu.Lock()
	c.closes++
	c.mu.Unlock()
	return c.closeErr
}

func (c *startupCloseChannel) closeCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closes
}

// noCloserChannel is a gateway.Channel fixture that deliberately does NOT
// implement StartupCloser. It is used to prove that startup cleanup skips
// channels without the optional interface.
type noCloserChannel struct {
	name   string
	runErr error
}

func (c *noCloserChannel) Name() string { return c.name }

func (c *noCloserChannel) Run(ctx context.Context, _ chan<- InboundEvent) error {
	if c.runErr != nil {
		return c.runErr
	}
	<-ctx.Done()
	return nil
}

func (c *noCloserChannel) Send(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

func discardGatewayLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestManager_Run_StartupFailure_InvokesStartupCloser(t *testing.T) {
	ch := &startupCloseChannel{
		name:   "failing",
		runErr: errors.New("bind port: already in use"),
	}

	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, discardGatewayLogger())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runDone := make(chan struct{})
	go func() {
		_ = m.Run(ctx)
		close(runDone)
	}()

	waitFor(t, 500*time.Millisecond, func() bool {
		return ch.closeCalls() == 1
	})

	cancel()
	<-runDone

	if got := ch.closeCalls(); got != 1 {
		t.Errorf("Close call count = %d, want 1", got)
	}
}

func TestManager_Run_StartupFailure_CleanupErrorDoesNotMaskStartupErr(t *testing.T) {
	startupErr := errors.New("bind port: already in use")
	cleanupErr := errors.New("leaked socket close: broken pipe")

	ch := &startupCloseChannel{
		name:     "failing",
		runErr:   startupErr,
		closeErr: cleanupErr,
	}

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, log)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runDone := make(chan struct{})
	go func() {
		_ = m.Run(ctx)
		close(runDone)
	}()

	waitFor(t, 500*time.Millisecond, func() bool {
		return ch.closeCalls() == 1
	})

	cancel()
	<-runDone

	logs := buf.String()
	if !strings.Contains(logs, startupErr.Error()) {
		t.Errorf("startup error %q missing from logs: %s", startupErr, logs)
	}
	if !strings.Contains(logs, cleanupErr.Error()) {
		t.Errorf("cleanup error %q missing from logs: %s", cleanupErr, logs)
	}
	// Expect a dedicated cleanup-failure log line that carries both errors,
	// proving cleanup is surfaced but does not replace the original log.
	if !strings.Contains(logs, "adapter startup cleanup") {
		t.Errorf("missing dedicated cleanup-failure log line; got: %s", logs)
	}
	// Existing "channel exited with error" log must still fire for the
	// original startup failure.
	if !strings.Contains(logs, "channel exited with error") {
		t.Errorf("missing original channel-exit log line; got: %s", logs)
	}
}

func TestManager_Run_ContextCanceledExit_SkipsStartupCloser(t *testing.T) {
	ch := &startupCloseChannel{
		name:   "normal",
		runErr: context.Canceled,
	}

	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, discardGatewayLogger())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	runDone := make(chan struct{})
	go func() {
		_ = m.Run(ctx)
		close(runDone)
	}()

	// Give the manager ample time to invoke cleanup (which it must not).
	time.Sleep(75 * time.Millisecond)

	if got := ch.closeCalls(); got != 0 {
		t.Errorf("Close must not run for context.Canceled exit; got %d calls", got)
	}

	cancel()
	<-runDone
}

func TestManager_Run_NilExit_SkipsStartupCloser(t *testing.T) {
	ch := &startupCloseChannel{
		name:   "clean",
		runErr: nil, // returns from Run only once ctx is cancelled
	}

	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, discardGatewayLogger())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	runDone := make(chan struct{})
	go func() {
		_ = m.Run(ctx)
		close(runDone)
	}()

	// Cancel cleanly; Run returns nil; Close must not be invoked.
	time.Sleep(10 * time.Millisecond)
	cancel()
	<-runDone

	if got := ch.closeCalls(); got != 0 {
		t.Errorf("Close must not run for a clean nil Run exit; got %d calls", got)
	}
}

func TestManager_Run_StartupFailure_NoStartupCloser_NoPanic(t *testing.T) {
	ch := &noCloserChannel{
		name:   "failing",
		runErr: errors.New("handshake rejected"),
	}

	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, discardGatewayLogger())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	runDone := make(chan struct{})
	go func() {
		_ = m.Run(ctx)
		close(runDone)
	}()

	// Allow startup error to propagate; there must be no panic even though
	// the channel does not implement StartupCloser.
	time.Sleep(50 * time.Millisecond)

	cancel()
	<-runDone
}
