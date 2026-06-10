package usagecmd

import (
	"math"
	"reflect"
	"strings"
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

func TestRenderFrameLinesSanitizesFrameSource(t *testing.T) {
	got := RenderFrameLines(FrameSnapshot{
		Source: "cached turn\n**Injected source:** `x`",
		Frame:  kernel.RenderFrame{Model: "gpt-5", SessionID: "sess-1"},
	})
	joined := strings.Join(got, "\n")
	for _, forbidden := range []string{"**Injected source:**", "`x`", "cached turn\n"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("RenderFrameLines leaked unsafe source field %q in %#v", forbidden, got)
		}
	}
	if got[0] != "Usage source: cached turn ''Injected source:'' 'x'" {
		t.Fatalf("source line = %q, want sanitized source", got[0])
	}
}

func TestRenderFrameLinesRemovesHiddenFormattingOperatorFields(t *testing.T) {
	got := RenderFrameLines(FrameSnapshot{
		Source: "cached turn\u202e",
		Frame:  kernel.RenderFrame{Model: "gpt-5\u200d", SessionID: "sess-1\u2066"},
	})
	joined := strings.Join(got, "\n")
	for _, forbidden := range []string{"\u202e", "\u200d", "\u2066"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("RenderFrameLines leaked hidden formatting rune %q in %#v", forbidden, got)
		}
	}
	want := []string{"Usage source: cached turn", "Model: gpt-5", "Session: sess-1", "Tokens: 0 in / 0 out"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RenderFrameLines = %#v, want hidden formatting removed as %#v", got, want)
	}
}

func TestRenderFrameLinesRedactsHiddenFormattingSecretLabels(t *testing.T) {
	got := RenderFrameLines(FrameSnapshot{
		Source: "cached to\u200dken=plain-secret-token",
		Frame: kernel.RenderFrame{
			Model:     "model api\u200d_key=plain-model-secret",
			SessionID: "session bearer\u200d=plain-session-secret",
		},
	})
	joined := strings.Join(got, "\n")
	for _, forbidden := range []string{"plain-secret-token", "plain-model-secret", "plain-session-secret", "token=", "api_key=", "bearer="} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("RenderFrameLines leaked hidden-formatting-obscured secret field %q in %#v", forbidden, got)
		}
	}
	if !strings.Contains(joined, "[redacted]") {
		t.Fatalf("RenderFrameLines missing redaction marker in %#v", got)
	}
}

func TestRenderFrameLinesRedactsAuthorizationOperatorFields(t *testing.T) {
	got := RenderFrameLines(FrameSnapshot{
		Source: "cached authorization=Bearer plain-secret-token",
		Frame: kernel.RenderFrame{
			Model:     "model authorization=Bearer model-secret-token",
			SessionID: "session authorization=Bearer session-secret-token",
		},
	})
	joined := strings.Join(got, "\n")
	for _, forbidden := range []string{"plain-secret-token", "model-secret-token", "session-secret-token", "authorization", "Bearer", "bearer"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("RenderFrameLines leaked authorization field %q in %#v", forbidden, got)
		}
	}
	if !strings.Contains(joined, "[redacted]") {
		t.Fatalf("RenderFrameLines missing redaction marker in %#v", got)
	}
}

func TestRenderFrameLinesSanitizesOperatorFields(t *testing.T) {
	got := RenderFrameLines(FrameSnapshot{
		Source: FrameSourceCached,
		Frame:  kernel.RenderFrame{Model: "gpt-5\n**Injected model:** `x`", SessionID: "sess`bad\nnext"},
	})
	joined := strings.Join(got, "\n")
	for _, forbidden := range []string{"**Injected model:**", "`x`", "sess`bad", "\nnext"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("RenderFrameLines leaked operator injection %q in %#v", forbidden, got)
		}
	}
	want := []string{"Usage source: cached turn", "Model: gpt-5 ''Injected model:'' 'x'", "Session: sess'bad next", "Tokens: 0 in / 0 out"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RenderFrameLines = %#v, want %#v", got, want)
	}
}

func TestRenderFrameLinesOmitsNonFiniteTelemetrySpeed(t *testing.T) {
	for _, speed := range []float64{math.Inf(1), math.NaN()} {
		got := RenderFrameLines(FrameSnapshot{
			Source: FrameSourceCached,
			Frame: kernel.RenderFrame{
				Model:     "gpt-5",
				SessionID: "sess-1",
				Telemetry: telemetry.Snapshot{TokensInTotal: 3, TokensOutTotal: 4, TokensPerSec: speed},
			},
		})
		joined := strings.Join(got, "\n")
		if strings.Contains(joined, "Inf") || strings.Contains(joined, "NaN") || strings.Contains(joined, "Speed:") {
			t.Fatalf("RenderFrameLines leaked non-finite speed %v in %#v", speed, got)
		}
	}
}

func TestRenderFrameLinesClampsNegativeTelemetry(t *testing.T) {
	got := RenderFrameLines(FrameSnapshot{
		Source: FrameSourceCached,
		Frame: kernel.RenderFrame{
			Model:     "gpt-5",
			SessionID: "sess-1",
			Telemetry: telemetry.Snapshot{TokensInTotal: -3, TokensOutTotal: -4, LatencyMsLast: -25, TokensPerSec: -12.345},
		},
	})
	joined := strings.Join(got, "\n")
	for _, forbidden := range []string{"-3", "-4", "-25", "-12"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("RenderFrameLines leaked negative telemetry %q in %#v", forbidden, got)
		}
	}
	want := []string{"Usage source: cached turn", "Model: gpt-5", "Session: sess-1", "Tokens: 0 in / 0 out"}
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
