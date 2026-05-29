package cron

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	hermesclient "github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"go.etcd.io/bbolt"
)

// fakeKernel implements KernelAPI. On Submit, schedules a single
// RenderFrame with Phase=PhaseIdle and History containing one
// assistant message with the configured response.
type fakeKernel struct {
	resp   string
	delay  time.Duration
	render chan kernel.RenderFrame
	mu     sync.Mutex
	events []kernel.PlatformEvent
}

func newFakeKernel(resp string, delay time.Duration) *fakeKernel {
	return &fakeKernel{
		resp:   resp,
		delay:  delay,
		render: make(chan kernel.RenderFrame, 4),
	}
}

func (fk *fakeKernel) Submit(e kernel.PlatformEvent) error {
	fk.mu.Lock()
	fk.events = append(fk.events, e)
	fk.mu.Unlock()

	go func() {
		if fk.delay > 0 {
			time.Sleep(fk.delay)
		}
		fk.render <- kernel.RenderFrame{
			Phase:     kernel.PhaseIdle,
			SessionID: e.SessionID,
			History: []hermesclient.Message{
				{Role: "user", Content: e.Text},
				{Role: "assistant", Content: fk.resp},
			},
		}
	}()
	return nil
}

func (fk *fakeKernel) Render() <-chan kernel.RenderFrame { return fk.render }

type erroringKernel struct{ err error }

func (e *erroringKernel) Submit(_ kernel.PlatformEvent) error { return e.err }
func (e *erroringKernel) Render() <-chan kernel.RenderFrame {
	ch := make(chan kernel.RenderFrame)
	close(ch)
	return ch
}

func newTestExecutorEnv(t *testing.T, fk KernelAPI) (*Executor, *atomic.Value, func()) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "session.db")
	db, _ := bbolt.Open(dbPath, 0o600, nil)
	js, _ := NewStore(db)
	msPath := filepath.Join(t.TempDir(), "memory.db")
	ms, _ := memory.OpenSqlite(msPath, 0, nil)
	rs := NewRunStore(ms.DB())

	var deliveries atomic.Value
	deliveries.Store([]string{})
	sink := FuncSink(func(_ context.Context, text string) error {
		cur := deliveries.Load().([]string)
		n := make([]string, len(cur), len(cur)+1)
		copy(n, cur)
		n = append(n, text)
		deliveries.Store(n)
		return nil
	})

	e := NewExecutor(ExecutorConfig{
		Kernel:      fk,
		JobStore:    js,
		RunStore:    rs,
		Sink:        sink,
		CallTimeout: 2 * time.Second,
	}, nil)

	cleanup := func() {
		_ = ms.Close(context.Background())
		_ = db.Close()
	}
	return e, &deliveries, cleanup
}

func TestExecutor_NormalResponseDelivers(t *testing.T) {
	fk := newFakeKernel("Morning report: all systems nominal.", 0)
	e, deliveries, cleanup := newTestExecutorEnv(t, fk)
	defer cleanup()

	job := NewJob("morning", "0 8 * * *", "status summary")
	_ = e.cfg.JobStore.Create(job)

	e.Run(context.Background(), job)

	got := deliveries.Load().([]string)
	if len(got) != 1 {
		t.Fatalf("deliveries = %d, want 1", len(got))
	}
	if got[0] != "Morning report: all systems nominal." {
		t.Errorf("delivery content = %q", got[0])
	}
	runs, _ := e.cfg.RunStore.LatestRuns(context.Background(), job.ID, 5)
	if len(runs) != 1 {
		t.Fatalf("runs = %d, want 1", len(runs))
	}
	if runs[0].Status != "success" || !runs[0].Delivered {
		t.Errorf("run = %+v, want success+delivered", runs[0])
	}
}

func TestCronExecutorDoesNotUseDeliveryOriginAsSessionIdentity(t *testing.T) {
	fk := newFakeKernel("Origin-routed cron output.", 0)
	e, deliveries, cleanup := newTestExecutorEnv(t, fk)
	defer cleanup()
	live := &fakeCronLiveAdapter{}
	e.cfg.LiveDelivery = live

	job := NewJob("origin-delivery", "@daily", "Summarize queue state.")
	job.Deliver = "origin"
	job.Origin = &DeliveryOrigin{
		Platform: "telegram",
		ChatID:   "-100777",
		ThreadID: "99",
	}
	if err := e.cfg.JobStore.Create(job); err != nil {
		t.Fatalf("Create job: %v", err)
	}

	e.Run(context.Background(), job)

	fk.mu.Lock()
	events := append([]kernel.PlatformEvent(nil), fk.events...)
	fk.mu.Unlock()
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if got := events[0].SessionContext; got != "" {
		t.Fatalf("SessionContext = %q, want empty cron-internal context", got)
	}
	for _, leaked := range []string{"telegram", "-100777", "99"} {
		if strings.Contains(events[0].Text, leaked) {
			t.Fatalf("cron prompt leaked delivery origin %q in %q", leaked, events[0].Text)
		}
	}
	if events[0].CronJobID != job.ID {
		t.Fatalf("CronJobID = %q, want %q", events[0].CronJobID, job.ID)
	}

	if got := deliveries.Load().([]string); len(got) != 0 {
		t.Fatalf("fallback deliveries = %d, want live origin delivery only: %#v", len(got), got)
	}
	if len(live.calls) != 1 {
		t.Fatalf("live deliveries = %d, want 1", len(live.calls))
	}
	call := live.calls[0]
	if got, want := call.target.Normalized(), "telegram:-100777:99"; got != want {
		t.Fatalf("live delivery target = %q, want %q", got, want)
	}
	if !call.target.Origin {
		t.Fatal("live delivery target Origin = false, want true")
	}
	if call.text != "Origin-routed cron output." {
		t.Fatalf("live delivery text = %q, want final assistant output", call.text)
	}
}

func TestCronExecutorSubmitsCronApprovalMode(t *testing.T) {
	t.Run("default deny", func(t *testing.T) {
		fk := newFakeKernel("ok", 0)
		e, _, cleanup := newTestExecutorEnv(t, fk)
		defer cleanup()
		job := NewJob("default-approval", "@daily", "p")
		_ = e.cfg.JobStore.Create(job)

		e.Run(context.Background(), job)

		fk.mu.Lock()
		defer fk.mu.Unlock()
		if len(fk.events) != 1 {
			t.Fatalf("events = %d, want 1", len(fk.events))
		}
		if got := fk.events[0].CronApprovalMode; got != "deny" {
			t.Fatalf("CronApprovalMode = %q, want deny", got)
		}
	})

	t.Run("configured approve", func(t *testing.T) {
		fk := newFakeKernel("ok", 0)
		e, _, cleanup := newTestExecutorEnv(t, fk)
		defer cleanup()
		e.cfg.CronApprovalMode = "approve"
		job := NewJob("approve-approval", "@daily", "p")
		_ = e.cfg.JobStore.Create(job)

		e.Run(context.Background(), job)

		fk.mu.Lock()
		defer fk.mu.Unlock()
		if len(fk.events) != 1 {
			t.Fatalf("events = %d, want 1", len(fk.events))
		}
		if got := fk.events[0].CronApprovalMode; got != "approve" {
			t.Fatalf("CronApprovalMode = %q, want approve", got)
		}
	})
}

func TestExecutor_SilentResponseSuppresses(t *testing.T) {
	fk := newFakeKernel("[SILENT]", 0)
	e, deliveries, cleanup := newTestExecutorEnv(t, fk)
	defer cleanup()
	job := NewJob("j", "@daily", "p")
	_ = e.cfg.JobStore.Create(job)
	e.Run(context.Background(), job)
	got := deliveries.Load().([]string)
	if len(got) != 0 {
		t.Errorf("deliveries = %d, want 0 (suppressed)", len(got))
	}
	runs, _ := e.cfg.RunStore.LatestRuns(context.Background(), job.ID, 5)
	if runs[0].Status != "suppressed" || runs[0].SuppressionReason != "silent" || runs[0].Delivered {
		t.Errorf("run = %+v, want suppressed/silent/!delivered", runs[0])
	}
}

func TestExecutor_EmptyResponseDeliversFailureNotice(t *testing.T) {
	fk := newFakeKernel("", 0)
	e, deliveries, cleanup := newTestExecutorEnv(t, fk)
	defer cleanup()
	job := NewJob("empty-job", "@daily", "p")
	_ = e.cfg.JobStore.Create(job)
	e.Run(context.Background(), job)
	got := deliveries.Load().([]string)
	if len(got) != 1 {
		t.Fatalf("deliveries = %d, want 1 (failure notice)", len(got))
	}
	if !strings.Contains(got[0], "empty-job") || !strings.Contains(got[0], "empty") {
		t.Errorf("notice = %q, want mention of job name + 'empty'", got[0])
	}
	runs, _ := e.cfg.RunStore.LatestRuns(context.Background(), job.ID, 5)
	if runs[0].Status != "error" || runs[0].SuppressionReason != "empty" || !runs[0].Delivered {
		t.Errorf("run = %+v, want error/empty/delivered", runs[0])
	}
}

func TestExecutor_TimeoutDeliversFailureNotice(t *testing.T) {
	fk := newFakeKernel("too late", 3*time.Second)
	e, deliveries, cleanup := newTestExecutorEnv(t, fk)
	e.cfg.CallTimeout = 100 * time.Millisecond
	defer cleanup()
	job := NewJob("slow", "@daily", "p")
	_ = e.cfg.JobStore.Create(job)
	e.Run(context.Background(), job)
	got := deliveries.Load().([]string)
	if len(got) != 1 {
		t.Fatalf("deliveries = %d, want 1 (timeout notice)", len(got))
	}
	if !strings.Contains(got[0], "slow") || !strings.Contains(got[0], "timed out") {
		t.Errorf("notice = %q, want mention of job name + 'timed out'", got[0])
	}
	runs, _ := e.cfg.RunStore.LatestRuns(context.Background(), job.ID, 5)
	if runs[0].Status != "timeout" || !runs[0].Delivered {
		t.Errorf("run = %+v, want timeout+delivered", runs[0])
	}
}

func TestExecutor_SubmitErrorRecordsWithoutDelivery(t *testing.T) {
	e, deliveries, cleanup := newTestExecutorEnv(t, &erroringKernel{err: errors.New("mailbox full")})
	defer cleanup()
	job := NewJob("x", "@daily", "p")
	_ = e.cfg.JobStore.Create(job)
	e.Run(context.Background(), job)
	got := deliveries.Load().([]string)
	if len(got) != 0 {
		t.Errorf("deliveries = %d, want 0 on kernel error", len(got))
	}
	runs, _ := e.cfg.RunStore.LatestRuns(context.Background(), job.ID, 5)
	if runs[0].Status != "error" || runs[0].Delivered {
		t.Errorf("run = %+v, want error/!delivered", runs[0])
	}
}

func TestExecutor_UpdatesJobLastRunStatus(t *testing.T) {
	fk := newFakeKernel("ok", 0)
	e, _, cleanup := newTestExecutorEnv(t, fk)
	defer cleanup()
	job := NewJob("update-test", "@daily", "p")
	_ = e.cfg.JobStore.Create(job)
	e.Run(context.Background(), job)
	got, _ := e.cfg.JobStore.Get(job.ID)
	if got.LastRunUnix == 0 {
		t.Error("LastRunUnix not updated")
	}
	if got.LastStatus != "success" {
		t.Errorf("LastStatus = %q, want success", got.LastStatus)
	}
}

func TestExecutorRecordAndUpdateJob_UsesRunCompletionState(t *testing.T) {
	e, _, cleanup := newTestExecutorEnv(t, newFakeKernel("unused", 0))
	defer cleanup()

	job := NewJob("completion-test", "every 30m", "p")
	job.Repeat = 5
	if err := e.cfg.JobStore.Create(job); err != nil {
		t.Fatalf("Create job: %v", err)
	}

	startedAt := time.Date(2026, 4, 27, 8, 0, 0, 0, time.UTC).Unix()
	for i, status := range []string{"success", "timeout", "suppressed"} {
		current, err := e.cfg.JobStore.Get(job.ID)
		if err != nil {
			t.Fatalf("Get job before %s run: %v", status, err)
		}
		run := Run{
			JobID:      job.ID,
			StartedAt:  startedAt + int64(i*60),
			FinishedAt: startedAt + int64(i*60) + 10,
			PromptHash: "hash",
			Status:     status,
		}
		if status == "timeout" {
			run.Delivered = true
			run.ErrorMsg = "context deadline exceeded"
		}
		if status == "suppressed" {
			run.SuppressionReason = "silent"
		}

		e.recordAndUpdateJob(context.Background(), current, run)

		got, err := e.cfg.JobStore.Get(job.ID)
		if err != nil {
			t.Fatalf("Get job after %s run: %v", status, err)
		}
		if got.LastRunUnix != run.StartedAt {
			t.Fatalf("%s LastRunUnix = %d, want %d", status, got.LastRunUnix, run.StartedAt)
		}
		if got.LastStatus != status {
			t.Fatalf("%s LastStatus = %q, want %q", status, got.LastStatus, status)
		}
		if got.RepeatCompleted != i+1 {
			t.Fatalf("%s RepeatCompleted = %d, want %d", status, got.RepeatCompleted, i+1)
		}
		if got.Paused {
			t.Fatalf("%s Paused = true, want recurring job active", status)
		}
	}

	runs, err := e.cfg.RunStore.LatestRuns(context.Background(), job.ID, 5)
	if err != nil {
		t.Fatalf("LatestRuns: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("runs = %d, want 3", len(runs))
	}
}
