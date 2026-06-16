package cron

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	cronlock "github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron/lock"
	"go.etcd.io/bbolt"
)

// fakeExecutor implements Runner.
type fakeExecutor struct {
	onRun func(context.Context, Job)
}

func (f *fakeExecutor) Run(ctx context.Context, j Job) {
	if f.onRun != nil {
		f.onRun(ctx, j)
	}
}

func TestScheduler_FiresJobOnTick(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "session.db")
	db, _ := bbolt.Open(dbPath, 0o600, nil)
	defer db.Close()
	js, _ := NewStore(db)

	j := NewJob("fast", "@every 1s", "tick")
	_ = js.Create(j)

	var fires atomic.Int32
	fe := &fakeExecutor{onRun: func(_ context.Context, _ Job) { fires.Add(1) }}

	s := NewScheduler(SchedulerConfig{Store: js, Executor: fe}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	<-ctx.Done()
	s.Stop(context.Background())

	if fires.Load() < 2 {
		t.Errorf("fires = %d, want at least 2 in 2.5s with @every 1s", fires.Load())
	}
}

func TestScheduler_PausedJobsAreIgnored(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "session.db")
	db, _ := bbolt.Open(dbPath, 0o600, nil)
	defer db.Close()
	js, _ := NewStore(db)

	j := NewJob("paused", "@every 500ms", "x")
	j.Paused = true
	_ = js.Create(j)

	var fires atomic.Int32
	fe := &fakeExecutor{onRun: func(_ context.Context, _ Job) { fires.Add(1) }}

	s := NewScheduler(SchedulerConfig{Store: js, Executor: fe}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	_ = s.Start(ctx)
	<-ctx.Done()
	s.Stop(context.Background())

	if fires.Load() != 0 {
		t.Errorf("paused job fired %d times, want 0", fires.Load())
	}
}

func TestScheduler_DefaultTickLockPathUsesGormesHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	want := filepath.Join(home, "cron", ".tick.lock")
	if got := cronlock.DefaultPath(); got != want {
		t.Fatalf("cronlock.DefaultPath() = %q, want %q", got, want)
	}
}

func TestScheduler_TickFileLockSkipsOverlappingProcess(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "cron", ".tick.lock")
	job := NewJob("locked", "@every 1s", "tick")

	var firstEntered atomic.Int32
	firstRelease := make(chan struct{})
	firstDone := make(chan struct{})
	fe1 := &fakeExecutor{onRun: func(_ context.Context, _ Job) {
		firstEntered.Add(1)
		<-firstRelease
	}}
	fe2 := &fakeExecutor{onRun: func(_ context.Context, _ Job) {
		t.Fatal("second scheduler ran while first tick held the file lock")
	}}

	s1 := NewScheduler(SchedulerConfig{Executor: fe1, LockPath: lockPath, MCPOrphanCleanup: func() {}}, nil)
	s2 := NewScheduler(SchedulerConfig{Executor: fe2, LockPath: lockPath, MCPOrphanCleanup: func() {}}, nil)

	go func() {
		s1.runTick(context.Background(), []Job{job})
		close(firstDone)
	}()
	deadline := time.After(2 * time.Second)
	for firstEntered.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("first tick did not enter job")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	s2.runTick(context.Background(), []Job{job})
	if firstEntered.Load() != 1 {
		t.Fatalf("first scheduler run count = %d, want 1", firstEntered.Load())
	}
	close(firstRelease)
	select {
	case <-firstDone:
	case <-time.After(2 * time.Second):
		t.Fatal("first tick did not release")
	}
}

func TestScheduler_TickFileLockSerializesSameSchedulerTicks(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "cron", ".tick.lock")
	first := NewJob("first", "@every 1s", "tick")
	second := NewJob("second", "@every 1s", "tick")
	firstEntered := make(chan struct{})
	firstRelease := make(chan struct{})
	secondDone := make(chan struct{})
	var ranSecond atomic.Int32

	runner := RunnerFunc(func(_ context.Context, job Job) {
		switch job.Name {
		case "first":
			close(firstEntered)
			<-firstRelease
		case "second":
			ranSecond.Add(1)
		}
	})
	s := NewScheduler(SchedulerConfig{Executor: runner, LockPath: lockPath, MCPOrphanCleanup: func() {}}, nil)

	go s.runTick(context.Background(), []Job{first})
	select {
	case <-firstEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("first tick did not enter job")
	}

	go func() {
		s.runTick(context.Background(), []Job{second})
		close(secondDone)
	}()
	select {
	case <-secondDone:
		t.Fatal("second tick returned while first tick held the same-scheduler lock")
	case <-time.After(50 * time.Millisecond):
	}

	close(firstRelease)
	select {
	case <-secondDone:
	case <-time.After(2 * time.Second):
		t.Fatal("second tick did not run after first released")
	}
	if ranSecond.Load() != 1 {
		t.Fatalf("second tick run count = %d, want 1", ranSecond.Load())
	}
}

func TestScheduler_InvalidScheduleSkippedButOthersRun(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "session.db")
	db, _ := bbolt.Open(dbPath, 0o600, nil)
	defer db.Close()
	js, _ := NewStore(db)

	bad := NewJob("bad", "not a cron", "x")
	good := NewJob("good", "@every 500ms", "y")
	_ = js.Create(bad)
	_ = js.Create(good)

	var fires atomic.Int32
	fe := &fakeExecutor{onRun: func(_ context.Context, j Job) {
		if j.Name == "good" {
			fires.Add(1)
		}
	}}

	s := NewScheduler(SchedulerConfig{Store: js, Executor: fe}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	<-ctx.Done()
	s.Stop(context.Background())

	if fires.Load() < 1 {
		t.Errorf("good job fires = %d, want >= 1 (bad job shouldn't block)", fires.Load())
	}
}
