package kernel

import (
	"context"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
)

func TestSelectTurnModelRoutesOnlySimpleTurns(t *testing.T) {
	routing := SmartModelRouting{
		Enabled:     true,
		SimpleModel: "hermes-agent-mini",
	}

	if got := selectTurnModel("hermes-agent", "what time is it in tokyo?", routing); got != "hermes-agent-mini" {
		t.Fatalf("simple turn model = %q, want hermes-agent-mini", got)
	}

	if got := selectTurnModel("hermes-agent", "implement a patch for this docker error", routing); got != "hermes-agent" {
		t.Fatalf("complex turn model = %q, want hermes-agent", got)
	}
}

func TestKernel_SelectsConfiguredModelPerTurnWithoutMutatingPrimary(t *testing.T) {
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{{Kind: hermes.EventDone, FinishReason: "stop"}}, "sess-routing")
	mc.Script([]hermes.Event{{Kind: hermes.EventDone, FinishReason: "stop"}}, "sess-routing")

	k := New(Config{
		Model:     "hermes-agent",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
		ModelRouting: SmartModelRouting{
			Enabled:     true,
			SimpleModel: "hermes-agent-mini",
		},
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go k.Run(ctx)

	<-k.Render()

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "what time is it in tokyo?"}); err != nil {
		t.Fatal(err)
	}
	first := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.Telemetry.TurnsCompleted == 1
	}, 2*time.Second)
	if first.Model != "hermes-agent-mini" {
		t.Fatalf("first frame model = %q, want hermes-agent-mini", first.Model)
	}
	if first.Telemetry.Model != "hermes-agent-mini" {
		t.Fatalf("first telemetry model = %q, want hermes-agent-mini", first.Telemetry.Model)
	}

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "implement a patch for this docker error"}); err != nil {
		t.Fatal(err)
	}
	second := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.Telemetry.TurnsCompleted == 2
	}, 2*time.Second)
	if second.Model != "hermes-agent" {
		t.Fatalf("second frame model = %q, want hermes-agent", second.Model)
	}
	if second.Telemetry.Model != "hermes-agent" {
		t.Fatalf("second telemetry model = %q, want hermes-agent", second.Telemetry.Model)
	}

	reqs := mc.Requests()
	if len(reqs) != 2 {
		t.Fatalf("request count = %d, want 2", len(reqs))
	}
	if reqs[0].Model != "hermes-agent-mini" {
		t.Fatalf("first request model = %q, want hermes-agent-mini", reqs[0].Model)
	}
	if reqs[1].Model != "hermes-agent" {
		t.Fatalf("second request model = %q, want hermes-agent", reqs[1].Model)
	}
}
