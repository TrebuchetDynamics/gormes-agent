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

	hermesclient "github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
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
