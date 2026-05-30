package usagecmd

import (
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func TestRenderFrameLinesUnavailable(t *testing.T) {
	got := RenderFrameLines(FrameSnapshot{Source: FrameSourceNone})
	want := []string{"Usage source: unavailable", "Runtime usage unavailable: no running or cached turn telemetry"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RenderFrameLines = %#v, want %#v", got, want)
	}
}

func TestRenderFrameLinesIncludesTelemetry(t *testing.T) {
	got := RenderFrameLines(FrameSnapshot{
		Source: FrameSourceRunning,
		Frame:  kernel.RenderFrame{Model: " gpt-5 ", SessionID: " sess-1 ", Telemetry: telemetry.Snapshot{TokensInTotal: 3, TokensOutTotal: 4, LatencyMsLast: 25, TokensPerSec: 12.345}},
	})
	want := []string{"Usage source: running turn", "Model: gpt-5", "Session: sess-1", "Tokens: 3 in / 4 out", "Last latency: 25 ms", "Speed: 12.35 tokens/sec"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RenderFrameLines = %#v, want %#v", got, want)
	}
}
