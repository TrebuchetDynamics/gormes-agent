package cron

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	hermesclient "github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestCronInactivityTimeoutInterruptsIdleRuns(t *testing.T) {
	k := &idleKernel{render: make(chan kernel.RenderFrame, 4)}
	e, deliveries, cleanup := newTestExecutorEnv(t, k)
	defer cleanup()
	e.cfg.CallTimeout = 2 * time.Second
	e.cfg.InactivityTimeout = 30 * time.Millisecond
	job := NewJob("idle", "@daily", "wait forever")
	if err := e.cfg.JobStore.Create(job); err != nil {
		t.Fatalf("Create job: %v", err)
	}

	_, err := e.RunWithRelease(context.Background(), job)

	if err == nil || !strings.Contains(err.Error(), "idle for") {
		t.Fatalf("RunWithRelease err = %v, want idle timeout", err)
	}
	got := deliveries.Load().([]string)
	if len(got) != 1 || !strings.Contains(got[0], "idle for") {
		t.Fatalf("deliveries = %#v, want idle timeout notice", got)
	}
	runs, err := e.cfg.RunStore.LatestRuns(context.Background(), job.ID, 1)
	if err != nil {
		t.Fatalf("LatestRuns: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != "timeout" || !strings.Contains(runs[0].ErrorMsg, "idle for") {
		t.Fatalf("runs = %+v, want timeout with idle evidence", runs)
	}
}

func TestCronInactivityTimeoutParsing(t *testing.T) {
	for _, tc := range []struct {
		name    string
		env     string
		want    time.Duration
		enabled bool
	}{
		{name: "default", want: 10 * time.Minute, enabled: true},
		{name: "seconds", env: "120", want: 120 * time.Second, enabled: true},
		{name: "zero disables", env: "0", enabled: false},
		{name: "negative disables", env: "-1", enabled: false},
		{name: "invalid default", env: "abc", want: 10 * time.Minute, enabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, enabled := ResolveCronInactivityTimeout(func(string) string { return tc.env })
			if enabled != tc.enabled || got != tc.want {
				t.Fatalf("ResolveCronInactivityTimeout(%q) = %v,%t want %v,%t", tc.env, got, enabled, tc.want, tc.enabled)
			}
		})
	}
}

func TestCronCodexExecutionPathRefreshes401(t *testing.T) {
	k := &codex401ThenSuccessKernel{render: make(chan kernel.RenderFrame, 2)}
	e, _, cleanup := newTestExecutorEnv(t, k)
	defer cleanup()
	var refreshes int
	e.cfg.Codex401Refresher = func(_ context.Context, job Job, err error) (bool, error) {
		refreshes++
		if job.Provider != "openai-codex" || !strings.Contains(err.Error(), "401") {
			t.Fatalf("refresh called with job.Provider=%q err=%v", job.Provider, err)
		}
		return true, nil
	}
	job := NewJob("codex", "@daily", "ping")
	job.Provider = "openai-codex"
	if err := e.cfg.JobStore.Create(job); err != nil {
		t.Fatalf("Create job: %v", err)
	}

	e.Run(context.Background(), job)

	if k.submits != 2 {
		t.Fatalf("submits = %d, want retry after 401", k.submits)
	}
	if refreshes != 1 {
		t.Fatalf("refreshes = %d, want 1", refreshes)
	}
	runs, err := e.cfg.RunStore.LatestRuns(context.Background(), job.ID, 1)
	if err != nil {
		t.Fatalf("LatestRuns: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != "success" {
		t.Fatalf("runs = %+v, want success after refresh retry", runs)
	}
}

type idleKernel struct {
	render chan kernel.RenderFrame
}

func (k *idleKernel) Submit(kernel.PlatformEvent) error { return nil }
func (k *idleKernel) Subscribe() (<-chan kernel.RenderFrame, func()) { return k.render, func() {} }
func (k *idleKernel) ConfigModel() string                          { return "fake-model" }

type codex401ThenSuccessKernel struct {
	mu      sync.Mutex
	submits int
	render  chan kernel.RenderFrame
}

func (k *codex401ThenSuccessKernel) Submit(e kernel.PlatformEvent) error {
	k.mu.Lock()
	k.submits++
	submits := k.submits
	k.mu.Unlock()
	if submits == 1 {
		return errCronTest401
	}
	k.render <- kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		SessionID: e.SessionID,
		History: []hermesclient.Message{
			{Role: "assistant", Content: "Recovered via refresh"},
		},
	}
	return nil
}

func (k *codex401ThenSuccessKernel) Subscribe() (<-chan kernel.RenderFrame, func()) {
	return k.render, func() {}
}
func (k *codex401ThenSuccessKernel) ConfigModel() string { return "fake-model" }
