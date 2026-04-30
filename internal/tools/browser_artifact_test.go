package tools

import (
	"strings"
	"testing"
)

func TestBrowserArtifactEnvelopeSanitizesPreviewAndState(t *testing.T) {
	envelope, err := BuildBrowserResultEnvelope(BrowserResultInput{
		Action: BrowserAction{Kind: BrowserActionSnapshot, TaskID: "browser-task"},
		State: BrowserPageState{
			URL:            "http://127.0.0.1:9222/json/version?token=plain-local-token",
			Title:          "Local Debug Page",
			Text:           "page body",
			Console:        []string{"cookie=plain-cookie-token", "safe console line"},
			Errors:         []string{"cdp websocket ws://127.0.0.1/devtools/browser/plain-cdp-token failed"},
			ScreenshotPath: "/tmp/gormes/screenshots/private-session/screen.png",
			Interactive:    3,
		},
		Output:    []byte("snapshot includes plain-cookie-token and ws://127.0.0.1/devtools/browser/plain-cdp-token"),
		MediaType: "text/plain",
		Budget: ToolResultBudgetConfig{
			OutputDir:       t.TempDir(),
			TextBudgetBytes: 4096,
			PreviewBytes:    512,
		},
	})
	if err != nil {
		t.Fatalf("BuildBrowserResultEnvelope: %v", err)
	}
	joined := envelope.Text + "\n" + envelope.State.URL + "\n" + strings.Join(envelope.State.Console, "\n") + "\n" + strings.Join(envelope.State.Errors, "\n") + "\n" + envelope.State.ScreenshotPath
	for _, forbidden := range []string{"plain-cookie-token", "plain-cdp-token", "127.0.0.1", "ws://", "/tmp/gormes"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("browser envelope leaked %q in %#v", forbidden, envelope)
		}
	}
	if !strings.Contains(envelope.Text, "[redacted]") {
		t.Fatalf("envelope text = %q, want redacted marker", envelope.Text)
	}
	if envelope.State.URL != "[browser_private_url_redacted]" {
		t.Fatalf("state URL = %q, want private URL redacted", envelope.State.URL)
	}
	if envelope.State.ScreenshotPath != "[browser_artifact_path_redacted]" {
		t.Fatalf("screenshot path = %q, want redacted artifact path", envelope.State.ScreenshotPath)
	}
}

func TestBrowserArtifactEnvelopePersistsOversizedSnapshotsWithConsoleMetadata(t *testing.T) {
	envelope, err := BuildBrowserResultEnvelope(BrowserResultInput{
		Action: BrowserAction{Kind: BrowserActionSnapshot, TaskID: "browser-task"},
		State: BrowserPageState{
			URL:         "https://gormes.ai/docs",
			Title:       "Gormes Docs",
			Console:     []string{"hydrated route"},
			Errors:      []string{"missing optional image"},
			Interactive: 5,
		},
		Output:    []byte(strings.Repeat("dom node ", 128)),
		MediaType: "text/plain",
		Budget: ToolResultBudgetConfig{
			OutputDir:       t.TempDir(),
			TextBudgetBytes: 128,
			PreviewBytes:    32,
		},
	})
	if err != nil {
		t.Fatalf("BuildBrowserResultEnvelope: %v", err)
	}
	if envelope.Evidence != BrowserEvidenceResultTruncated {
		t.Fatalf("evidence = %q, want %q", envelope.Evidence, BrowserEvidenceResultTruncated)
	}
	if envelope.Tool.Artifact == "" {
		t.Fatalf("artifact empty for oversized snapshot: %#v", envelope.Tool)
	}
	if envelope.State.Title != "Gormes Docs" || envelope.State.Interactive != 5 {
		t.Fatalf("state metadata lost: %#v", envelope.State)
	}
	if !strings.Contains(strings.Join(envelope.State.Console, "\n"), "hydrated route") || !strings.Contains(strings.Join(envelope.State.Errors, "\n"), "missing optional image") {
		t.Fatalf("console/error metadata lost: %#v", envelope.State)
	}
}
