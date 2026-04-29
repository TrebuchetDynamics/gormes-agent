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

func TestFormatStreamPlain_IncludesSoulLine(t *testing.T) {
	f := kernel.RenderFrame{
		DraftText: "draft",
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "running tool foo"},
		},
	}
	got := FormatStreamPlain(f)
	if !strings.Contains(got, "draft") || !strings.Contains(got, "running tool foo") {
		t.Errorf("FormatStreamPlain = %q", got)
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
	f := kernel.RenderFrame{LastError: "Forbidden: <html><body><svg>bad</svg> secret sk-test-123</body></html>"}
	got := FormatErrorPlain(f)
	for _, forbidden := range []string{"<html", "<svg", "sk-test-123", "secret"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatErrorPlain leaked provider HTML/secret marker %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "Forbidden: provider returned HTML error body") {
		t.Fatalf("FormatErrorPlain = %q, want sanitized provider HTML evidence", got)
	}
}

func TestFormatErrorTelegram_SanitizesProviderHTMLBody(t *testing.T) {
	f := kernel.RenderFrame{LastError: "Forbidden: <html><body><svg>bad</svg> secret sk-test-123</body></html>"}
	got := FormatErrorTelegram(f)
	for _, forbidden := range []string{"<html", "<svg", "sk-test-123", "secret"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatErrorTelegram leaked provider HTML/secret marker %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "provider returned HTML error body") {
		t.Fatalf("FormatErrorTelegram = %q, want sanitized provider HTML evidence", got)
	}
}
