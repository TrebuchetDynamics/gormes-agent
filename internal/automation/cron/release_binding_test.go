package cron

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// bindingCloser is a minimal io.Closer used to prove the binding
// closes registered resources at run end without any real DB or HTTP.
type bindingCloser struct {
	mu        sync.Mutex
	closed    int
	returnErr error
}

func (c *bindingCloser) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed++
	return c.returnErr
}

func (c *bindingCloser) closedCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// bindingKiller is a fake SubprocessKiller used in lieu of os.Process.
type bindingKiller struct {
	mu     sync.Mutex
	called []int
	err    error
}

func (k *bindingKiller) Kill(pid int) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.called = append(k.called, pid)
	return k.err
}

func (k *bindingKiller) calls() []int {
	k.mu.Lock()
	defer k.mu.Unlock()
	out := make([]int, len(k.called))
	copy(out, k.called)
	return out
}

// hasEvidenceCode returns true when evidence contains at least one entry
// with the given Code.
func hasEvidenceCode(evidence []ReleaseEvidence, code ReleaseEvidenceCode) bool {
	for _, ev := range evidence {
		if ev.Code == code {
			return true
		}
	}
	return false
}

// TestExecutorRun_ReleasesRegisteredResourcesOnSuccess: a Run() that
// registers a session-db closer + an HTTP idle closable + a tool subprocess
// PID emits release evidence for all three after the kernel turn completes.
func TestExecutorRun_ReleasesRegisteredResourcesOnSuccess(t *testing.T) {
	fk := newFakeKernel("ok output", 0)
	e, _, cleanup := newTestExecutorEnv(t, fk)
	defer cleanup()

	dbCloser := &bindingCloser{}
	idleCloser := &bindingCloser{}
	killer := &bindingKiller{}

	e.cfg.SubprocessKiller = killer
	e.cfg.RegisterRelease = func(_ context.Context, ledger *RunReleaseLedger, _ Job) {
		ledger.RegisterCloser("session-db", dbCloser)
		ledger.RegisterIdleClosable("http-idle", idleCloser)
		ledger.RegisterSubprocess(4242)
	}

	job := NewJob("ok-job", "@daily", "p")
	_ = e.cfg.JobStore.Create(job)

	evidence, runErr := e.RunWithRelease(context.Background(), job)
	if runErr != nil {
		t.Fatalf("RunWithRelease error = %v, want nil on success", runErr)
	}
	if dbCloser.closedCount() != 1 {
		t.Errorf("session-db Close called %d times, want 1", dbCloser.closedCount())
	}
	if idleCloser.closedCount() != 1 {
		t.Errorf("http-idle Close called %d times, want 1", idleCloser.closedCount())
	}
	if got := killer.calls(); len(got) != 1 || got[0] != 4242 {
		t.Errorf("killer calls = %v, want [4242]", got)
	}
	if !hasEvidenceCode(evidence, ReleaseEvidenceSessionDBClosed) {
		t.Errorf("evidence missing %s; got %+v", ReleaseEvidenceSessionDBClosed, evidence)
	}
	if !hasEvidenceCode(evidence, ReleaseEvidenceHTTPIdleClosed) {
		t.Errorf("evidence missing %s; got %+v", ReleaseEvidenceHTTPIdleClosed, evidence)
	}
	if !hasEvidenceCode(evidence, ReleaseEvidenceSubprocessKilled) {
		t.Errorf("evidence missing %s; got %+v", ReleaseEvidenceSubprocessKilled, evidence)
	}
}

// TestExecutorRun_ReleasesOnKernelError: when the kernel rejects Submit,
// resources still get released, the kernel error is surfaced, and the
// release evidence for every registered resource is present.
func TestExecutorRun_ReleasesOnKernelError(t *testing.T) {
	kernelErr := errors.New("mailbox full")
	e, _, cleanup := newTestExecutorEnv(t, &erroringKernel{err: kernelErr})
	defer cleanup()

	dbCloser := &bindingCloser{}
	idleCloser := &bindingCloser{}
	killer := &bindingKiller{}

	e.cfg.SubprocessKiller = killer
	e.cfg.RegisterRelease = func(_ context.Context, ledger *RunReleaseLedger, _ Job) {
		ledger.RegisterCloser("session-db", dbCloser)
		ledger.RegisterIdleClosable("http-idle", idleCloser)
		ledger.RegisterSubprocess(7777)
	}

	job := NewJob("kernel-err", "@daily", "p")
	_ = e.cfg.JobStore.Create(job)

	evidence, runErr := e.RunWithRelease(context.Background(), job)
	if runErr == nil {
		t.Fatalf("RunWithRelease error = nil, want kernel error")
	}
	if !errors.Is(runErr, kernelErr) && !strings.Contains(runErr.Error(), kernelErr.Error()) {
		t.Errorf("runErr = %v, want kernel error %v", runErr, kernelErr)
	}
	if dbCloser.closedCount() != 1 {
		t.Errorf("session-db Close called %d times on kernel-error path, want 1", dbCloser.closedCount())
	}
	if idleCloser.closedCount() != 1 {
		t.Errorf("http-idle Close called %d times on kernel-error path, want 1", idleCloser.closedCount())
	}
	if got := killer.calls(); len(got) != 1 || got[0] != 7777 {
		t.Errorf("killer calls on kernel-error = %v, want [7777]", got)
	}
	if !hasEvidenceCode(evidence, ReleaseEvidenceSessionDBClosed) ||
		!hasEvidenceCode(evidence, ReleaseEvidenceHTTPIdleClosed) ||
		!hasEvidenceCode(evidence, ReleaseEvidenceSubprocessKilled) {
		t.Errorf("evidence on kernel-error path missing one of the release codes; got %+v", evidence)
	}
}

// TestExecutorRun_ReleasesOnContextCancel: cancelling the parent context
// mid-run still triggers Release for all registered resources.
func TestExecutorRun_ReleasesOnContextCancel(t *testing.T) {
	// kernel never delivers a frame quickly; we cancel the parent ctx so
	// the executor exits via the ctx-Done branch.
	fk := newFakeKernel("never reaches caller", 5*time.Second)
	e, _, cleanup := newTestExecutorEnv(t, fk)
	e.cfg.CallTimeout = 5 * time.Second
	defer cleanup()

	dbCloser := &bindingCloser{}
	idleCloser := &bindingCloser{}
	killer := &bindingKiller{}

	e.cfg.SubprocessKiller = killer
	e.cfg.RegisterRelease = func(_ context.Context, ledger *RunReleaseLedger, _ Job) {
		ledger.RegisterCloser("session-db", dbCloser)
		ledger.RegisterIdleClosable("http-idle", idleCloser)
		ledger.RegisterSubprocess(9001)
	}

	job := NewJob("ctx-cancel", "@daily", "p")
	_ = e.cfg.JobStore.Create(job)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel before invoking so the executor takes the cancelled-ctx exit
	// path deterministically without goroutine timing.
	cancel()

	evidence, _ := e.RunWithRelease(ctx, job)
	if dbCloser.closedCount() != 1 {
		t.Errorf("session-db Close called %d times on ctx-cancel, want 1", dbCloser.closedCount())
	}
	if idleCloser.closedCount() != 1 {
		t.Errorf("http-idle Close called %d times on ctx-cancel, want 1", idleCloser.closedCount())
	}
	if got := killer.calls(); len(got) != 1 || got[0] != 9001 {
		t.Errorf("killer calls on ctx-cancel = %v, want [9001]", got)
	}
	if !hasEvidenceCode(evidence, ReleaseEvidenceSessionDBClosed) ||
		!hasEvidenceCode(evidence, ReleaseEvidenceHTTPIdleClosed) ||
		!hasEvidenceCode(evidence, ReleaseEvidenceSubprocessKilled) {
		t.Errorf("evidence on ctx-cancel path missing one of the release codes; got %+v", evidence)
	}
}

// TestExecutorRun_NoResourcesEmitsSkippedEvidence: a Run() that registers
// nothing emits exactly one cron_release_skipped_no_resource entry.
func TestExecutorRun_NoResourcesEmitsSkippedEvidence(t *testing.T) {
	fk := newFakeKernel("ok", 0)
	e, _, cleanup := newTestExecutorEnv(t, fk)
	defer cleanup()

	job := NewJob("no-resources", "@daily", "p")
	_ = e.cfg.JobStore.Create(job)

	evidence, runErr := e.RunWithRelease(context.Background(), job)
	if runErr != nil {
		t.Fatalf("RunWithRelease error = %v, want nil", runErr)
	}
	skipped := 0
	for _, ev := range evidence {
		if ev.Code == ReleaseEvidenceSkippedNoResource {
			skipped++
		}
	}
	if skipped != 1 {
		t.Errorf("evidence = %+v, want exactly one cron_release_skipped_no_resource", evidence)
	}
}

// TestExecutorRun_ReleaseErrorsDoNotMaskRunError: when the kernel returns
// an error AND a registered closer errors at Release, Run returns the
// kernel error while still emitting the release evidence (including the
// failed-close code) in its result.
func TestExecutorRun_ReleaseErrorsDoNotMaskRunError(t *testing.T) {
	kernelErr := errors.New("submit blew up")
	e, _, cleanup := newTestExecutorEnv(t, &erroringKernel{err: kernelErr})
	defer cleanup()

	closeErr := errors.New("idle close failed")
	failingIdle := &bindingCloser{returnErr: closeErr}
	okDB := &bindingCloser{}
	killer := &bindingKiller{}

	e.cfg.SubprocessKiller = killer
	e.cfg.RegisterRelease = func(_ context.Context, ledger *RunReleaseLedger, _ Job) {
		ledger.RegisterCloser("session-db", okDB)
		ledger.RegisterIdleClosable("http-idle", failingIdle)
		ledger.RegisterSubprocess(31337)
	}

	job := NewJob("kernel-and-release-err", "@daily", "p")
	_ = e.cfg.JobStore.Create(job)

	evidence, runErr := e.RunWithRelease(context.Background(), job)
	if runErr == nil {
		t.Fatalf("RunWithRelease error = nil, want kernel error to surface")
	}
	if !strings.Contains(runErr.Error(), kernelErr.Error()) {
		t.Errorf("runErr = %v; want kernel error %v to be returned, not masked by release errors", runErr, kernelErr)
	}
	if okDB.closedCount() != 1 {
		t.Errorf("session-db Close called %d times, want 1", okDB.closedCount())
	}
	if failingIdle.closedCount() != 1 {
		t.Errorf("http-idle Close called %d times, want 1", failingIdle.closedCount())
	}
	if got := killer.calls(); len(got) != 1 || got[0] != 31337 {
		t.Errorf("killer calls = %v, want [31337]", got)
	}
	if !hasEvidenceCode(evidence, ReleaseEvidenceSessionDBClosed) {
		t.Errorf("evidence missing %s; got %+v", ReleaseEvidenceSessionDBClosed, evidence)
	}
	if !hasEvidenceCode(evidence, ReleaseEvidenceHTTPIdleClosedFailed) {
		t.Errorf("evidence missing %s for failing idle close; got %+v", ReleaseEvidenceHTTPIdleClosedFailed, evidence)
	}
	if !hasEvidenceCode(evidence, ReleaseEvidenceSubprocessKilled) {
		t.Errorf("evidence missing %s; got %+v", ReleaseEvidenceSubprocessKilled, evidence)
	}
}
