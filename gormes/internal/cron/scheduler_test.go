package cron

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

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
