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

func TestFormatStreamPlain_DoesNotInlineToolProgress(t *testing.T) {
	f := kernel.RenderFrame{
		DraftText: "draft",
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "tool: search_files: Approval mode config normalization"},
		},
	}
	got := FormatStreamPlain(f)
	if strings.Contains(got, "search_files") {
		t.Fatalf("FormatStreamPlain = %q, want assistant stream without inline tool progress", got)
	}
	if got != "draft" {
		t.Fatalf("FormatStreamPlain = %q, want draft only", got)
	}
}

func TestFormatToolProgressPlain_IncludesHermesStyleToolTrace(t *testing.T) {
	f := kernel.RenderFrame{
		DraftText: "draft",
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "tool: search_files: Approval mode config normalization"},
		},
	}
	got := FormatToolProgressPlain(f)
	if !strings.Contains(got, `🔎 search_files: "Approval mode config normalization"`) {
		t.Fatalf("FormatToolProgressPlain = %q, want Hermes-style search_files trace", got)
	}
}

func TestFormatToolProgressPlainMode_OffSuppressesToolTrace(t *testing.T) {
	f := kernel.RenderFrame{
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "tool: browser_navigate: https://example.test"},
		},
	}
	if got := FormatToolProgressPlainMode(f, "off"); got != "" {
		t.Fatalf("FormatToolProgressPlainMode(off) = %q, want no tool progress", got)
	}
}

func TestFormatToolProgressPlainMode_NewSuppressesConsecutiveSameTool(t *testing.T) {
	f := kernel.RenderFrame{SoulEvents: []kernel.SoulEntry{
		{At: time.Now(), Text: "tool: read_file: a.txt"},
		{At: time.Now(), Text: "tool: read_file: b.txt"},
		{At: time.Now(), Text: "tool: search_files: needle"},
		{At: time.Now(), Text: "tool: read_file: c.txt"},
	}}

	got := FormatToolProgressPlainMode(f, "new")
	for _, want := range []string{
		`📖 read_file: "a.txt"`,
		`🔎 search_files: "needle"`,
		`📖 read_file: "c.txt"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatToolProgressPlainMode(new) missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "b.txt") {
		t.Fatalf("FormatToolProgressPlainMode(new) rendered consecutive same-tool preview:\n%s", got)
	}
}

func TestFormatToolProgressPlain_ToolTraceFixtureMatrix(t *testing.T) {
	tests := []struct {
		name  string
		event string
		want  string
	}{
		{name: "memory", event: "tool: memory: recall Juan context", want: `🧠 memory: "recall Juan context"`},
		{name: "read", event: "tool: read_file: internal/gateway/render.go", want: `📖 read_file: "internal/gateway/render.go"`},
		{name: "patch", event: "tool: patch: replace render tail", want: `🔧 patch: "replace render tail"`},
		{name: "terminal", event: "tool: terminal: go test ./internal/gateway", want: `💻 terminal: "go test ./internal/gateway"`},
		{name: "browser", event: "tool: browser_navigate: https://gormes.ai", want: `🌐 browser_navigate: "https://gormes.ai"`},
		{name: "snapshot", event: "tool: browser_snapshot", want: `📸 browser_snapshot...`},
		{name: "skill_view", event: "tool: skill_view: gormes-hermes-parity", want: `📚 skill_view: "gormes-hermes-parity"`},
		{name: "skills_list", event: "tool: skills_list", want: `📚 skills_list...`},
		{name: "todo", event: "tool: todo: planning 5 task(s)", want: `📋 todo: "planning 5 task(s)"`},
		{name: "execute_code", event: "tool: execute_code: printf shell-output", want: `💻 execute_code: "printf shell-output"`},
		{name: "cronjob", event: "tool: cronjob: run", want: `⏰ cronjob: "run"`},
		{name: "transcribe_audio", event: "tool: transcribe_audio: audio_path=/tmp/voice.ogg", want: `🎙️ transcribe_audio...`},
		{name: "text_to_speech", event: "tool: text_to_speech: voice reply", want: `🔊 text_to_speech...`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatToolProgressPlain(kernel.RenderFrame{SoulEvents: []kernel.SoulEntry{{At: time.Now(), Text: tt.event}}})
			if !strings.Contains(got, tt.want) {
				t.Fatalf("FormatToolProgressPlain(%q) = %q, want %q", tt.event, got, tt.want)
			}
		})
	}
}

func TestFormatToolProgressPlain_TruncatedWebPreviewsKeepLeftEdge(t *testing.T) {
	f := kernel.RenderFrame{SoulEvents: []kernel.SoulEntry{
		{At: time.Now(), Text: "tool: web_extract: https://docs.openclaw.ai/concepts/multi-agent"},
		{At: time.Now(), Text: "tool: browser_navigate: https://docs.openclaw.ai/concepts/multi-agent"},
		{At: time.Now(), Text: "tool: web_search: site:docs.openclaw.ai/concepts/multi-agent strings in tools"},
	}}

	got := FormatToolProgressPlain(f)
	for _, want := range []string{
		`📄 web_extract: "https://docs.openclaw.ai/concepts/mul..."`,
		`🌐 browser_navigate: "https://docs.openclaw.ai/concepts/mul..."`,
		`🔍 web_search: "site:docs.openclaw.ai/concepts/multi-..."`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatToolProgressPlain missing left-edge preview %q in:\n%s", want, got)
		}
	}
}

func TestFormatToolProgressPlain_UnknownToolUsesGenericBoundedEvidence(t *testing.T) {
	got := FormatToolProgressPlain(kernel.RenderFrame{SoulEvents: []kernel.SoulEntry{{At: time.Now(), Text: "tool: custom_provider_debug: " + strings.Repeat("payload ", 40)}}})
	if !strings.Contains(got, `🔧 tool_progress: "payload payload`) {
		t.Fatalf("FormatToolProgressPlain unknown tool = %q, want generic bounded tool_progress evidence", got)
	}
	if strings.Contains(got, "custom_provider_debug") {
		t.Fatalf("FormatToolProgressPlain leaked unknown provider tool name in %q", got)
	}
	if len([]rune(got)) > 130 {
		t.Fatalf("FormatToolProgressPlain unknown tool trace too long: %d runes in %q", len([]rune(got)), got)
	}
}

func TestFormatToolProgressPlain_SuppressesLegacyCompletionNoise(t *testing.T) {
	got := FormatToolProgressPlain(kernel.RenderFrame{SoulEvents: []kernel.SoulEntry{
		{At: time.Now(), Text: "tool: execute_code: printf hi"},
		{At: time.Now(), Text: "tool done: execute_code"},
	}})
	if !strings.Contains(got, `💻 execute_code: "printf hi"`) {
		t.Fatalf("FormatToolProgressPlain = %q, want execute_code start event", got)
	}
	if strings.Contains(got, "tool done") || strings.Contains(got, `🔧 tool done`) {
		t.Fatalf("FormatToolProgressPlain leaked legacy completion noise: %q", got)
	}
}

func TestFormatToolProgressPlain_MineruGatewayTranscriptShape(t *testing.T) {
	f := kernel.RenderFrame{SoulEvents: []kernel.SoulEntry{
		{At: time.Now(), Text: "tool: skill_view: gormes-hermes-parity"},
		{At: time.Now(), Text: "tool: cronjob: list"},
		{At: time.Now(), Text: "tool: browser_navigate: https://www.reddit.com/r/WebAfterAI/s/example"},
		{At: time.Now(), Text: "tool: browser_navigate: https://old.reddit.com/r/WebAfterAI/s/example"},
		{At: time.Now(), Text: "tool: terminal: curl -L https://example.test/post.json"},
		{At: time.Now(), Text: "tool: browser_snapshot"},
		{At: time.Now(), Text: "tool: terminal: curl -L https://example.test/post.json"},
		{At: time.Now(), Text: "tool: terminal: curl -L https://example.test/post.json"},
	}}

	got := FormatToolProgressPlain(f)
	for _, want := range []string{
		`📚 skill_view: "gormes-hermes-parity"`,
		`⏰ cronjob: "list"`,
		`🌐 browser_navigate: "https://www.reddit.com/r/WebAfterAI/s..."`,
		`🌐 browser_navigate: "https://old.reddit.com/r/WebAfterAI/s..."`,
		`💻 terminal: "curl -L https://example.test/post.json"`,
		`📸 browser_snapshot...`,
		`💻 terminal: "curl -L https://example.test/post.json" (×2)`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatToolProgressPlain missing %q in:\n%s", want, got)
		}
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

func TestFormatFinalTelegramText_RendersMarkdownDocumentForTelegram(t *testing.T) {
	input := "```markdown\n# Multi-agent routing\n\nRun multiple _isolated_ agents with `agentDir`.\n\n- Workspace files\n1. Restart gateway\n\nSee [OpenClaw](https://docs.openclaw.ai/concepts/multi-agent) and [Skills](/tools/skills).\n\n```bash\nopenclaw agents list --bindings\n```\n```"

	got := FormatFinalTelegramText(input)

	for _, forbidden := range []string{"```markdown", "# Multi-agent routing", "- Workspace files", "[Skills](/tools/skills)"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatFinalTelegramText leaked raw Markdown %q in:\n%s", forbidden, got)
		}
	}
	for _, want := range []string{
		`*Multi\-agent routing*`,
		`_isolated_`,
		"`agentDir`",
		`• Workspace files`,
		`1\. Restart gateway`,
		`[OpenClaw](https://docs\.openclaw\.ai/concepts/multi\-agent)`,
		`Skills \(/tools/skills\)`,
		"```\nopenclaw agents list --bindings\n```",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatFinalTelegramText missing %q in:\n%s", want, got)
		}
	}
}

func TestFormatFinalTelegramText_EscapesLinkURLsForMarkdownV2(t *testing.T) {
	input := "Extracted docs from: https://docs.openclaw.ai/concepts/presence\n\n- [Typing indicators](https://docs.openclaw.ai/concepts/typing-indicators)"

	got := FormatFinalTelegramText(input)

	for _, want := range []string{
		`https://docs\.openclaw\.ai/concepts/presence`,
		`[Typing indicators](https://docs\.openclaw\.ai/concepts/typing\-indicators)`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatFinalTelegramText missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, `(https://docs.openclaw.ai`) {
		t.Fatalf("FormatFinalTelegramText left unescaped link URL in:\n%s", got)
	}
}

func TestFormatFinalTelegramText_EscapesPlainTextWithoutMarkdownEntities(t *testing.T) {
	got := FormatFinalTelegramText("Use a_b(c)! 3.14")
	want := `Use a\_b\(c\)\! 3\.14`
	if got != want {
		t.Fatalf("FormatFinalTelegramText plain = %q, want %q", got, want)
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

func TestFormatErrorPlain_SanitizesProviderUnauthorizedJSONBody(t *testing.T) {
	f := kernel.RenderFrame{LastError: `Unauthorized: {"detail":"Unauthorized"}`}
	got := FormatErrorPlain(f)
	for _, forbidden := range []string{`{"detail"`, `"Unauthorized"}`, "detail"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatErrorPlain leaked provider JSON marker %q in %q", forbidden, got)
		}
	}
	if got != "❌ Unauthorized: provider authentication failed" {
		t.Fatalf("FormatErrorPlain = %q, want actionable auth failure", got)
	}
}

func TestFormatErrorTelegram_SanitizesProviderUnauthorizedJSONBody(t *testing.T) {
	f := kernel.RenderFrame{LastError: `Unauthorized: {"detail":"Unauthorized"}`}
	got := FormatErrorTelegram(f)
	for _, forbidden := range []string{`\\{`, "detail"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatErrorTelegram leaked provider JSON marker %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "provider authentication failed") {
		t.Fatalf("FormatErrorTelegram = %q, want actionable auth failure", got)
	}
}
