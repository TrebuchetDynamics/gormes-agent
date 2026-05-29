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
