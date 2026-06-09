package rendering

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestFormatBrowserArtifactTelegramEscapesAndBoundsArtifactOutput(t *testing.T) {
	envelope := tools.BrowserResultEnvelope{
		State: tools.BrowserPageState{
			URL:            "https://gormes.ai/docs?q=a_b",
			Title:          "Browser [Docs] (ready)",
			Text:           "DOM <main> includes _markdown_ and [link](x)",
			Console:        []string{"warn: selector #main missing", "debug token=[redacted]"},
			Errors:         []string{"error: click failed (timeout)"},
			ScreenshotPath: "[browser_artifact_path_redacted]",
			Interactive:    4,
		},
		Text:     "[tool_output_truncated artifact=browser/snapshot.txt bytes=8192 preview=DOM <main>]",
		Evidence: tools.BrowserEvidenceResultTruncated,
		Tool: tools.ToolResultEvidence{
			Code:     tools.ToolResultEvidenceTruncated,
			Artifact: "browser/snapshot.txt",
			Preview:  "DOM <main> includes _markdown_ and [link](x)",
			Bytes:    8192,
		},
	}

	got := FormatBrowserArtifactTelegram(envelope)

	for _, want := range []string{
		"🌐 *Browser artifact*",
		"Title: Browser \\[Docs\\] \\(ready\\)",
		"URL: https://gormes\\.ai/docs?q\\=a\\_b",
		"Artifact: browser/snapshot\\.txt \\(8192 bytes\\)",
		"Screenshot: browser artifact available",
		"Console: warn: selector \\#main missing; debug token\\=\\[redacted\\]",
		"Errors: error: click failed \\(timeout\\)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatBrowserArtifactTelegram missing %q in:\n%s", want, got)
		}
	}
	for _, forbidden := range []string{"/tmp/", "token=plain", "[link](x)", "_markdown_"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatBrowserArtifactTelegram leaked or left unescaped %q in:\n%s", forbidden, got)
		}
	}
	if len([]rune(got)) > MaxMessageLen {
		t.Fatalf("FormatBrowserArtifactTelegram length = %d, want <= %d", len([]rune(got)), MaxMessageLen)
	}
}

func TestFormatBrowserArtifactTelegramRedactsQuotedLocalPaths(t *testing.T) {
	envelope := tools.BrowserResultEnvelope{
		State: tools.BrowserPageState{
			URL:     "\"/tmp/quoted-secret.html\"",
			Console: []string{"saved '/tmp/quoted-console.png'"},
		},
	}

	got := FormatBrowserArtifactTelegram(envelope)
	for _, forbidden := range []string{"/tmp/quoted-secret.html", "/tmp/quoted-console.png"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatBrowserArtifactTelegram leaked quoted local path %q in:\n%s", forbidden, got)
		}
	}
	if strings.Count(got, "\\[path\\]") < 2 {
		t.Fatalf("FormatBrowserArtifactTelegram did not redact quoted local paths in:\n%s", got)
	}
}

func TestFormatBrowserArtifactTelegramRedactsLocalPathsInNestedText(t *testing.T) {
	envelope := tools.BrowserResultEnvelope{
		State: tools.BrowserPageState{
			Console: []string{"saved screenshot to /tmp/console-secret.png"},
			Errors:  []string{"open C:\\Users\\secret\\trace.log failed"},
		},
		Tool: tools.ToolResultEvidence{Preview: "artifact at file:///tmp/preview-secret.html"},
	}

	got := FormatBrowserArtifactTelegram(envelope)
	for _, forbidden := range []string{"/tmp/console-secret.png", "C:\\Users\\secret\\trace.log", "file://", "/tmp/preview-secret.html"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatBrowserArtifactTelegram leaked nested local path %q in:\n%s", forbidden, got)
		}
	}
	if strings.Count(got, "\\[path\\]") < 3 {
		t.Fatalf("FormatBrowserArtifactTelegram did not redact nested paths in:\n%s", got)
	}
}

func TestFormatBrowserArtifactTelegramRedactsLocalFileURLs(t *testing.T) {
	envelope := tools.BrowserResultEnvelope{
		State: tools.BrowserPageState{URL: "file:///tmp/secret.html"},
	}

	got := FormatBrowserArtifactTelegram(envelope)
	for _, forbidden := range []string{"file://", "/tmp/secret.html"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatBrowserArtifactTelegram leaked local file URL %q in:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "URL: \\[path\\]") {
		t.Fatalf("FormatBrowserArtifactTelegram missing redacted URL path in:\n%s", got)
	}
}

func TestFormatBrowserArtifactTelegramCollapsesInjectedEnvelopeFields(t *testing.T) {
	envelope := tools.BrowserResultEnvelope{
		State: tools.BrowserPageState{
			Title: "Docs ready\nErrors: forged",
			URL:   "https://gormes.ai/docs\nEvidence: forged",
		},
		Evidence: "browser_ok\nScreenshot: /tmp/secret.png",
		Tool: tools.ToolResultEvidence{
			Artifact: "browser/snapshot.txt\nPreview: forged",
			Bytes:    12,
		},
	}

	got := FormatBrowserArtifactTelegram(envelope)
	for _, forbidden := range []string{"\nErrors: forged", "\nEvidence: forged", "\nScreenshot:", "\nPreview: forged", "/tmp/secret.png"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatBrowserArtifactTelegram leaked injected field %q in:\n%s", forbidden, got)
		}
	}
	for _, want := range []string{"Title: Docs ready Errors: forged", "URL: https://gormes\\.ai/docs Evidence: forged", "Artifact: browser/snapshot\\.txt Preview: forged \\(12 bytes\\)", "Evidence: browser\\_ok Screenshot: \\[path\\]"} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatBrowserArtifactTelegram missing collapsed field %q in:\n%s", want, got)
		}
	}
}

func TestFormatBrowserArtifactTelegramTextFallbackEvidence(t *testing.T) {
	envelope := tools.BuildBrowserUnavailableResult(tools.BrowserAction{Kind: tools.BrowserActionSnapshot}, "telegram media delivery unavailable")

	got := FormatBrowserArtifactTelegram(envelope)

	if !strings.Contains(got, "Evidence: browser\\_backend\\_unavailable") {
		t.Fatalf("missing unavailable evidence in:\n%s", got)
	}
	if !strings.Contains(got, "browser\\_artifact\\_text\\_fallback") {
		t.Fatalf("missing text fallback evidence in:\n%s", got)
	}
}
