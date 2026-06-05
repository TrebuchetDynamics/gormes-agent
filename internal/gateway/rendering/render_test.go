package rendering

import (
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestFormatStreamPlainActivePreviewAddsCursor(t *testing.T) {
	frame := kernel.RenderFrame{Phase: kernel.PhaseStreaming, DraftText: "hello"}
	if got := FormatStreamPlain(frame); got != "hello ▉" {
		t.Fatalf("FormatStreamPlain() = %q, want active preview cursor", got)
	}
}

func TestFormatToolProgressEventsRedactsUnknownToolPayload(t *testing.T) {
	frame := kernel.RenderFrame{
		Phase:      kernel.PhaseStreaming,
		SoulEvents: []kernel.SoulEntry{{At: time.Now(), Text: "tool: custom_debug: secret-token https://example.invalid/raw"}},
	}

	events := FormatToolProgressEvents(frame, "all", "request 1")
	if len(events) != 1 {
		t.Fatalf("FormatToolProgressEvents() len = %d, want 1", len(events))
	}
	if events[0].ToolName != "tool_progress" || events[0].Summary != "tool_progress started" {
		t.Fatalf("FormatToolProgressEvents() = %#v, want generic redacted progress", events[0])
	}
	if strings.Contains(events[0].Summary, "secret-token") || strings.Contains(events[0].Summary, "example.invalid") {
		t.Fatalf("FormatToolProgressEvents() leaked raw payload in summary %q", events[0].Summary)
	}
}

func TestFormatFinalTelegramTextRewritesMarkdownTables(t *testing.T) {
	input := "| Name | Status |\n| --- | --- |\n| Gateway | green |"
	got := FormatFinalTelegramText(input)
	for _, want := range []string{"*Gateway*", "• Name: Gateway", "• Status: green"} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatFinalTelegramText() missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "| --- |") {
		t.Fatalf("FormatFinalTelegramText() left raw table syntax in:\n%s", got)
	}
}

func TestFormatBrowserArtifactTelegramBoundsOutput(t *testing.T) {
	envelope := tools.BrowserResultEnvelope{
		Text: "preview",
		State: tools.BrowserPageState{
			Title:          "A_[title](unsafe)",
			URL:            "https://example.com/a_b?x=(1)",
			ScreenshotPath: "/tmp/secret.png",
		},
		Tool: tools.ToolResultEvidence{Artifact: "text", Bytes: 42},
	}

	got := FormatBrowserArtifactTelegram(envelope)
	if len([]rune(got)) > MaxMessageLen {
		t.Fatalf("FormatBrowserArtifactTelegram() length = %d, want <= %d", len([]rune(got)), MaxMessageLen)
	}
	if strings.Contains(got, "/tmp/secret.png") {
		t.Fatalf("FormatBrowserArtifactTelegram() leaked screenshot path:\n%s", got)
	}
	if !strings.Contains(got, `browser\_artifact\_text\_fallback`) {
		t.Fatalf("FormatBrowserArtifactTelegram() missing fallback evidence:\n%s", got)
	}
}
