package gateway

import (
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestFormatStreamPlain_DraftPassThrough(t *testing.T) {
	f := kernel.RenderFrame{DraftText: "hello world"}
	if got := FormatStreamPlain(f); got != "hello world" {
		t.Errorf("FormatStreamPlain = %q", got)
	}
}

func TestFormatStreamPlain_ActivePreviewAddsHermesCursor(t *testing.T) {
	f := kernel.RenderFrame{Phase: kernel.PhaseStreaming, DraftText: "hello world"}
	if got := FormatStreamPlain(f); got != "hello world ▉" {
		t.Fatalf("FormatStreamPlain active preview = %q, want Hermes cursor", got)
	}
}

func TestFormatStreamTelegram_ActivePreviewAddsHermesCursor(t *testing.T) {
	f := kernel.RenderFrame{Phase: kernel.PhaseStreaming, DraftText: "hello world"}
	if got := FormatStreamTelegram(f); got != "hello world ▉" {
		t.Fatalf("FormatStreamTelegram active preview = %q, want Hermes cursor", got)
	}
}

func TestFormatFinalPlain_DoesNotIncludeStreamingCursor(t *testing.T) {
	f := kernel.RenderFrame{
		History: []hermes.Message{{Role: "assistant", Content: "final answer"}},
	}
	if got := FormatFinalPlain(f); strings.Contains(got, "▉") {
		t.Fatalf("FormatFinalPlain = %q, want no streaming cursor", got)
	}
}

func TestFormatStreamPlain_IncludesHermesStyleToolTrace(t *testing.T) {
	f := kernel.RenderFrame{
		DraftText: "draft",
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "tool: search_files: Approval mode config normalization"},
		},
	}
	got := FormatStreamPlain(f)
	if !strings.Contains(got, "draft") {
		t.Fatalf("FormatStreamPlain lost draft body: %q", got)
	}
	if !strings.Contains(got, `🔎 search_files: "Approval mode config normalization"`) {
		t.Fatalf("FormatStreamPlain = %q, want Hermes-style search_files trace", got)
	}
}

func TestFormatStreamPlain_ToolTraceFixtureMatrix(t *testing.T) {
	tests := []struct {
		name  string
		event string
		want  string
	}{
		{name: "memory", event: "tool: memory: recall Juan context", want: `🧠 memory: "recall Juan context"`},
		{name: "read", event: "tool: read_file: internal/gateway/render.go", want: `📖 read_file: "internal/gateway/render.go"`},
		{name: "patch", event: "tool: patch: replace render tail", want: `🔧 patch: "replace render tail"`},
		{name: "terminal", event: "tool: terminal: go test ./internal/gateway", want: `🖥 terminal: "go test ./internal/gateway"`},
		{name: "browser", event: "tool: browser_navigate: https://gormes.ai", want: `🔎 browser_navigate: "https://gormes.ai"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatStreamPlain(kernel.RenderFrame{SoulEvents: []kernel.SoulEntry{{At: time.Now(), Text: tt.event}}})
			if !strings.Contains(got, tt.want) {
				t.Fatalf("FormatStreamPlain(%q) = %q, want %q", tt.event, got, tt.want)
			}
		})
	}
}

func TestFormatStreamPlain_UnknownToolUsesGenericBoundedEvidence(t *testing.T) {
	got := FormatStreamPlain(kernel.RenderFrame{SoulEvents: []kernel.SoulEntry{{At: time.Now(), Text: "tool: custom_provider_debug: " + strings.Repeat("payload ", 40)}}})
	if !strings.Contains(got, `🔧 tool_progress: "payload payload`) {
		t.Fatalf("FormatStreamPlain unknown tool = %q, want generic bounded tool_progress evidence", got)
	}
	if strings.Contains(got, "custom_provider_debug") {
		t.Fatalf("FormatStreamPlain leaked unknown provider tool name in %q", got)
	}
	if len([]rune(got)) > 130 {
		t.Fatalf("FormatStreamPlain unknown tool trace too long: %d runes in %q", len([]rune(got)), got)
	}
}

func TestFormatFinalPlain_LastAssistant(t *testing.T) {
	f := kernel.RenderFrame{
		History: []hermes.Message{
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "the answer"},
		},
	}
	if got := FormatFinalPlain(f); got != "the answer" {
		t.Errorf("FormatFinalPlain = %q", got)
	}
}

func TestFormatErrorPlain(t *testing.T) {
	f := kernel.RenderFrame{LastError: "boom"}
	if got := FormatErrorPlain(f); got != "❌ boom" {
		t.Errorf("FormatErrorPlain = %q", got)
	}
}

func TestFormatStreamTelegram_EscapesAndEmits(t *testing.T) {
	f := kernel.RenderFrame{DraftText: "wow!"}
	plain := FormatStreamPlain(f)
	tg := FormatStreamTelegram(f)
	if plain == tg {
		t.Fatalf("plain and telegram outputs should differ; both = %q", plain)
	}
	if !strings.Contains(tg, "wow") {
		t.Errorf("telegram output lost body: %q", tg)
	}
}

func TestFormatErrorPlain_SanitizesProviderHTMLBody(t *testing.T) {
	f := kernel.RenderFrame{LastError: "Forbidden: <html><body><svg>bad</svg> secret plain-provider-token</body></html>"}
	got := FormatErrorPlain(f)
	for _, forbidden := range []string{"<html", "<svg", "plain-provider-token", "secret"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatErrorPlain leaked provider HTML/secret marker %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "Forbidden: provider returned HTML error body") {
		t.Fatalf("FormatErrorPlain = %q, want sanitized provider HTML evidence", got)
	}
}

func TestFormatErrorTelegram_SanitizesProviderHTMLBody(t *testing.T) {
	f := kernel.RenderFrame{LastError: "Forbidden: <html><body><svg>bad</svg> secret plain-provider-token</body></html>"}
	got := FormatErrorTelegram(f)
	for _, forbidden := range []string{"<html", "<svg", "plain-provider-token", "secret"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatErrorTelegram leaked provider HTML/secret marker %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "provider returned HTML error body") {
		t.Fatalf("FormatErrorTelegram = %q, want sanitized provider HTML evidence", got)
	}
}
