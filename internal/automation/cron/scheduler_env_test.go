package cron

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestCronSchedulerExpandsConfigEnvRefs(t *testing.T) {
	t.Setenv("GORMES_CRON_MODEL", "gpt-4o-mini-cron-test")
	t.Setenv("GORMES_CRON_SECRET_TOKEN", "should-not-appear-in-evidence")

	fk := newFakeKernel("ok", 0)
	e, _, cleanup := newTestExecutorEnv(t, fk)
	defer cleanup()
	job := NewJob("env-job", "@daily", "status")
	job.Model = "${GORMES_CRON_MODEL}"
	if err := e.cfg.JobStore.Create(job); err != nil {
		t.Fatalf("Create job: %v", err)
	}

	e.Run(context.Background(), job)

	fk.mu.Lock()
	events := append([]kernelEventSnapshot(nil), snapshotKernelEvents(fk.events)...)
	fk.mu.Unlock()
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Model != "gpt-4o-mini-cron-test" {
		t.Fatalf("PlatformEvent.Model = %q, want expanded env ref", events[0].Model)
	}

	unresolved := NewJob("missing-env-job", "@daily", "status")
	unresolved.Model = "${GORMES_CRON_MISSING_MODEL}"
	unresolved.Provider = "${GORMES_CRON_SECRET_TOKEN}"
	expanded, evidence := ExpandCronJobEnvRefs(unresolved, func(name string) string {
		if name == "GORMES_CRON_SECRET_TOKEN" {
			return "should-not-appear-in-evidence"
		}
		return ""
	})
	if expanded.Model != "${GORMES_CRON_MISSING_MODEL}" {
		t.Fatalf("unresolved model = %q, want literal passthrough", expanded.Model)
	}
	if expanded.Provider != "should-not-appear-in-evidence" {
		t.Fatalf("resolved provider = %q, want env value", expanded.Provider)
	}
	if len(evidence) != 1 {
		t.Fatalf("evidence = %+v, want one unresolved env ref", evidence)
	}
	rendered := evidence[0].String()
	if !strings.Contains(rendered, "cron_env_ref_unresolved") ||
		!strings.Contains(rendered, "GORMES_CRON_MISSING_MODEL") {
		t.Fatalf("evidence = %q, want unresolved env ref code and variable", rendered)
	}
	if strings.Contains(rendered, "should-not-appear-in-evidence") {
		t.Fatalf("evidence leaked resolved secret value: %q", rendered)
	}
}

type kernelEventSnapshot struct {
	Model string
}

func snapshotKernelEvents(events []kernel.PlatformEvent) []kernelEventSnapshot {
	out := make([]kernelEventSnapshot, 0, len(events))
	for _, event := range events {
		out = append(out, kernelEventSnapshot{Model: event.Model})
	}
	return out
}
