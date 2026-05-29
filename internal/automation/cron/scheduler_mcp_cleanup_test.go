package cron

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCronSchedulerRunsMCPOrphanCleanupAfterTickJoin(t *testing.T) {
	longStarted := make(chan struct{})
	shortFinished := make(chan struct{})
	releaseLong := make(chan struct{})
	tickDone := make(chan struct{})
	cleanupCalled := make(chan struct{})

	var active atomic.Int32
	var completed atomic.Int32
	var cleanupActive atomic.Int32
	var cleanupCompleted atomic.Int32
	var cleanupCalls atomic.Int32
	var closeCleanup sync.Once

	fe := &fakeExecutor{onRun: func(_ context.Context, job Job) {
		active.Add(1)
		defer func() {
			active.Add(-1)
			completed.Add(1)
			if job.ID == "short" {
				close(shortFinished)
			}
		}()

		switch job.ID {
		case "long":
			close(longStarted)
			<-releaseLong
		case "short":
			<-longStarted
		}
	}}
	s := NewScheduler(SchedulerConfig{
		Executor: fe,
		MCPOrphanCleanup: func() {
			cleanupActive.Store(active.Load())
			cleanupCompleted.Store(completed.Load())
			cleanupCalls.Add(1)
			closeCleanup.Do(func() { close(cleanupCalled) })
		},
	}, nil)

	go func() {
		s.runTick(context.Background(), []Job{
			{ID: "long", Name: "long"},
			{ID: "short", Name: "short"},
		})
		close(tickDone)
	}()

	select {
	case <-shortFinished:
	case <-time.After(2 * time.Second):
		t.Fatal("short job did not finish while sibling job remained active")
	}
	if got := active.Load(); got != 1 {
		t.Fatalf("active jobs after short finish = %d; want 1 long sibling still running", got)
	}

	select {
	case <-cleanupCalled:
		t.Fatal("cleanup ran before the tick's sibling jobs joined")
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseLong)

	select {
	case <-tickDone:
	case <-time.After(2 * time.Second):
		t.Fatal("tick did not finish after releasing long job")
	}
	if got := cleanupCalls.Load(); got != 1 {
		t.Fatalf("cleanup calls = %d; want 1", got)
	}
	if got := cleanupActive.Load(); got != 0 {
		t.Fatalf("active jobs observed by cleanup = %d; want 0", got)
	}
	if got := cleanupCompleted.Load(); got != 2 {
		t.Fatalf("completed jobs observed by cleanup = %d; want 2", got)
	}
}
