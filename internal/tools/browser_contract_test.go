package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowserContractValidateAction(t *testing.T) {
	tests := []struct {
		name     string
		action   BrowserAction
		wantCode string
	}{
		{
			name: "navigate public url with ssrf guard",
			action: BrowserAction{
				Kind:    BrowserActionNavigate,
				TaskID:  "task-1",
				URL:     "https://example.com/docs",
				Options: BrowserSSRFGuardOptions{CloudConfigured: true, AutoLocalForPrivateURLs: true},
			},
		},
		{
			name: "navigate private url reuses local route guard",
			action: BrowserAction{
				Kind:    BrowserActionNavigate,
				TaskID:  "task-1",
				URL:     "http://localhost:3000/",
				Options: BrowserSSRFGuardOptions{CloudConfigured: true, AutoLocalForPrivateURLs: true},
			},
		},
		{
			name:   "snapshot accepted without backend",
			action: BrowserAction{Kind: BrowserActionSnapshot, TaskID: "task-2"},
		},
		{
			name:   "extract accepted without backend",
			action: BrowserAction{Kind: BrowserActionExtract, TaskID: "task-2"},
		},
		{
			name:   "wait accepted without backend",
			action: BrowserAction{Kind: BrowserActionWait, TaskID: "task-2", Selector: ".ready"},
		},
		{
			name:     "unknown action rejected",
			action:   BrowserAction{Kind: "hover", Selector: "#buy"},
			wantCode: BrowserEvidenceInvalidAction,
		},
		{
			name:     "navigate requires url",
			action:   BrowserAction{Kind: BrowserActionNavigate},
			wantCode: BrowserEvidenceMissingURL,
		},
		{
			name:     "click requires selector",
			action:   BrowserAction{Kind: BrowserActionClick},
			wantCode: BrowserEvidenceMissingSelector,
		},
		{
			name: "cloud private url blocked when no local sidecar",
			action: BrowserAction{
				Kind: BrowserActionNavigate,
				URL:  "http://127.0.0.1:8080/",
				Options: BrowserSSRFGuardOptions{
					CloudConfigured:         true,
					AutoLocalForPrivateURLs: false,
				},
			},
			wantCode: browserSSRFPrivateURLBlocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateBrowserAction(tt.action)
			if tt.wantCode == "" {
				if !got.Allowed || got.Evidence != BrowserEvidenceActionAccepted {
					t.Fatalf("ValidateBrowserAction() = %#v, want accepted", got)
				}
				if tt.action.URL != "" && strings.Contains(tt.action.URL, "localhost") && !got.Route.ForceLocal {
					t.Fatalf("private navigation did not keep local sidecar: %#v", got.Route)
				}
				return
			}
			if got.Allowed || got.Evidence != tt.wantCode {
				t.Fatalf("ValidateBrowserAction() = %#v, want denied evidence %q", got, tt.wantCode)
			}
		})
	}
}

func TestBrowserResultEnvelopeUsesArtifactPointer(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(strings.Repeat("browser body ", 20))
	result, err := BuildBrowserResultEnvelope(BrowserResultInput{
		Action: BrowserAction{Kind: BrowserActionSnapshot, TaskID: "session-1"},
		State: BrowserPageState{
			URL:            "https://example.com/dashboard",
			Title:          "Dashboard",
			Text:           "ready",
			Console:        []string{"console: loaded"},
			Errors:         []string{"content_none"},
			ScreenshotPath: "screens/s1.png",
		},
		Output:    raw,
		MediaType: "text/plain",
		Budget: ToolResultBudgetConfig{
			OutputDir:       dir,
			TextBudgetBytes: 32,
			PreviewBytes:    16,
		},
	})
	if err != nil {
		t.Fatalf("BuildBrowserResultEnvelope returned error: %v", err)
	}
	if result.Evidence != BrowserEvidenceResultTruncated {
		t.Fatalf("evidence = %q, want %q", result.Evidence, BrowserEvidenceResultTruncated)
	}
	if result.Tool.Code != ToolResultEvidenceTruncated || result.Tool.Artifact == "" {
		t.Fatalf("tool evidence = %#v, want truncated artifact", result.Tool)
	}
	if !strings.Contains(result.Text, "tool_output_artifact") || !strings.Contains(result.Text, "browser body") {
		t.Fatalf("bounded text missing pointer/preview: %q", result.Text)
	}
	if _, err := os.Stat(filepath.Join(dir, result.Tool.Artifact)); err != nil {
		t.Fatalf("artifact was not written: %v", err)
	}
	if result.State.URL != "https://example.com/dashboard" || result.State.Title != "Dashboard" || result.State.Text != "ready" || len(result.State.Console) != 1 || len(result.State.Errors) != 1 || result.State.ScreenshotPath == "" {
		t.Fatalf("state not preserved: %#v", result.State)
	}
}

func TestBrowserTranscript(t *testing.T) {
	transcript := NewBrowserTranscript("task-7")
	transcript.Record(BrowserTranscriptEvent{Kind: BrowserEventAction, Evidence: BrowserEvidenceActionAccepted, Action: BrowserAction{Kind: BrowserActionNavigate, URL: "https://example.com"}})
	transcript.Record(BrowserTranscriptEvent{Kind: BrowserEventResult, Evidence: BrowserEvidenceResultOK, State: BrowserPageState{URL: "https://example.com", Title: "Example", Text: "", Console: []string{"log"}, Errors: []string{"content_none"}}})
	events := transcript.Events()
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	events[0].Evidence = "mutated"
	if transcript.Events()[0].Evidence != BrowserEvidenceActionAccepted {
		t.Fatalf("Events did not return a defensive copy")
	}
	if transcript.TaskID != "task-7" {
		t.Fatalf("TaskID = %q", transcript.TaskID)
	}
}

func TestBrowserUnavailableBackendEnvelope(t *testing.T) {
	result := BuildBrowserUnavailableResult(BrowserAction{Kind: BrowserActionClick, Selector: "#submit"}, "no_browser_backend")
	if result.Evidence != BrowserEvidenceBackendUnavailable {
		t.Fatalf("evidence = %q", result.Evidence)
	}
	if !strings.Contains(result.Text, "browser_unavailable") || !strings.Contains(result.Text, "no_browser_backend") {
		t.Fatalf("unexpected unavailable text: %q", result.Text)
	}
	if result.Action.Kind != BrowserActionClick || result.Action.Selector != "#submit" {
		t.Fatalf("action not preserved: %#v", result.Action)
	}
}
