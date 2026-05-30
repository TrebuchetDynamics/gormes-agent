package usagepage

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func TestHandleSlashOpensPageAndRequestsAccountFetch(t *testing.T) {
	result := HandleSlash(kernel.RenderFrame{
		SessionID: "frame-session",
		Model:     "gpt-usage",
		Telemetry: telemetry.Snapshot{TokensInTotal: 12, TokensOutTotal: 8},
	}, "explicit-session", true)
	if !result.OpenPage || !result.FetchAccount || result.Status != "usage opened" {
		t.Fatalf("HandleSlash result = %+v, want open usage page with account fetch", result)
	}
	if !strings.Contains(result.Page.Body, AccountLoadingLine) {
		t.Fatalf("usage page missing account loading marker:\n%s", result.Page.Body)
	}
}

func TestHandleSlashRejectsMissingTelemetry(t *testing.T) {
	result := HandleSlash(kernel.RenderFrame{}, "", true)
	if result.OpenPage || result.FetchAccount || result.Status != "no API calls yet" {
		t.Fatalf("HandleSlash result = %+v, want no page and no API calls status", result)
	}
}

func TestBuildFormatsFrameEvidence(t *testing.T) {
	page, ok := Build(kernel.RenderFrame{
		SessionID: "frame-session",
		Model:     "gpt-usage",
		Telemetry: telemetry.Snapshot{TokensInTotal: 12, TokensOutTotal: 8, LatencyMsLast: 44, TokensPerSec: 5.5},
		ContextStatus: &llm.ContextStatus{
			ContextLength:    100,
			LastTotalTokens:  20,
			UsagePercent:     20,
			CompressionCount: 2,
		},
	}, "explicit-session")
	if !ok {
		t.Fatal("Build ok = false, want usage page")
	}
	for _, want := range []string{"Usage source: local TUI frame", "Model: gpt-usage", "Session: explicit-session", "Input tokens: 12", "Output tokens: 8", "Total tokens: 20", "Last latency: 44 ms", "Speed: 5.50 tokens/sec", "Context: 20 / 100 tokens (20.0%)", "Compressions: 2"} {
		if !strings.Contains(page.Body, want) {
			t.Fatalf("usage page missing %q:\n%s", want, page.Body)
		}
	}
}

func TestBuildRejectsMissingTelemetry(t *testing.T) {
	if _, ok := Build(kernel.RenderFrame{}, ""); ok {
		t.Fatal("Build ok = true, want false without local token counters")
	}
}

func TestReplaceAccountLoading(t *testing.T) {
	body := AppendAccountLines("base", []string{AccountLoadingLine})
	got := ReplaceAccountLoading(body, []string{"Provider: openrouter"})
	if !strings.Contains(got, "Provider: openrouter") || strings.Contains(got, AccountLoadingLine) {
		t.Fatalf("ReplaceAccountLoading = %q, want provider line without loading marker", got)
	}
}
