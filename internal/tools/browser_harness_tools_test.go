package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestBrowserHarnessToolsExposeHermesNames(t *testing.T) {
	got := NewBrowserHarnessTools(BrowserHarnessToolsConfig{})
	names := make([]string, 0, len(got))
	for _, tool := range got {
		names = append(names, tool.Name())
		if !strings.HasPrefix(tool.Name(), "browser_") {
			t.Fatalf("tool name = %q, want Hermes browser_* name", tool.Name())
		}
		if !json.Valid(tool.Schema()) {
			t.Fatalf("%s schema is invalid JSON: %s", tool.Name(), tool.Schema())
		}
	}
	want := []string{
		"browser_back",
		"browser_cdp",
		"browser_click",
		"browser_console",
		"browser_dialog",
		"browser_get_images",
		"browser_navigate",
		"browser_press",
		"browser_scroll",
		"browser_snapshot",
		"browser_type",
		"browser_vision",
	}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("tool names = %v, want %v", names, want)
	}
}

func TestBrowserHarnessNavigateUsesNewTabAndEnvelope(t *testing.T) {
	runner := &recordingHarnessRunner{
		result: BrowserHarnessProcessResult{Stdout: []byte(`{"url":"https://example.com","title":"Example","interactive":[{"ref":"@e1","text":"Docs"}]}`)},
	}
	tool := NewBrowserHarnessTool("browser_navigate", BrowserHarnessToolsConfig{
		Runner: runner,
		Budget: ToolResultBudgetConfig{OutputDir: t.TempDir(), TextBudgetBytes: 4096, PreviewBytes: 512},
	})

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.com","task_id":"Browser Task"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got, wantPrefix := strings.Join(runner.argv[:2], "\x00"), "go-browser-harness\x00--action-json"; got != wantPrefix {
		t.Fatalf("argv prefix = %q, want %q", got, wantPrefix)
	}
	action := decodeHarnessAction(t, runner.argv[2])
	if action.SchemaVersion != browserHarnessActionSchemaVersion || action.Kind != BrowserActionNavigate || action.URL != "https://example.com" {
		t.Fatalf("navigate action = %#v", action)
	}
	if !action.NewTab {
		t.Fatalf("navigate action must request a new tab: %#v", action)
	}
	if runner.env["BU_NAME"] != "gormes_Browser_Task" {
		t.Fatalf("BU_NAME = %q, want sanitized task id", runner.env["BU_NAME"])
	}
	var out BrowserHarnessToolResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, raw)
	}
	if out.Tool != "browser_navigate" || out.Evidence != BrowserHarnessEvidenceCommandOK || !strings.Contains(out.Text, "Example") {
		t.Fatalf("response = %#v", out)
	}
}

func TestBrowserHarnessClickAndTypeUseSnapshotRefs(t *testing.T) {
	for _, tc := range []struct {
		name       string
		args       string
		wantAction BrowserHarnessActionRequest
		shouldDeny bool
	}{
		{
			name:       "browser_click",
			args:       `{"ref":"@e3","task_id":"task-1"}`,
			wantAction: BrowserHarnessActionRequest{Kind: BrowserActionClick, Ref: "@e3", TaskID: "task-1"},
		},
		{
			name:       "browser_type",
			args:       `{"ref":"@e3","text":"hello","task_id":"task-1"}`,
			wantAction: BrowserHarnessActionRequest{Kind: BrowserActionType, Ref: "@e3", Text: "hello", TaskID: "task-1"},
		},
		{
			name:       "browser_click",
			args:       `{"task_id":"task-1"}`,
			shouldDeny: true,
		},
	} {
		t.Run(tc.name+"_"+strings.ReplaceAll(tc.args, `"`, ""), func(t *testing.T) {
			runner := &recordingHarnessRunner{result: BrowserHarnessProcessResult{Stdout: []byte(`{"ok":true}`)}}
			tool := NewBrowserHarnessTool(tc.name, BrowserHarnessToolsConfig{
				Runner: runner,
				Budget: ToolResultBudgetConfig{OutputDir: t.TempDir(), TextBudgetBytes: 4096, PreviewBytes: 512},
			})
			_, err := tool.Execute(context.Background(), json.RawMessage(tc.args))
			if tc.shouldDeny {
				if err == nil || !strings.Contains(err.Error(), "ref is required") {
					t.Fatalf("Execute err = %v, want missing ref", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			action := decodeHarnessAction(t, runner.argv[2])
			if action.Kind != tc.wantAction.Kind || action.Ref != tc.wantAction.Ref || action.Text != tc.wantAction.Text || action.TaskID != tc.wantAction.TaskID {
				t.Fatalf("action = %#v, want %#v", action, tc.wantAction)
			}
		})
	}
}

func TestBrowserHarnessCDPConsoleAndVisionBuildExpectedActions(t *testing.T) {
	tests := []struct {
		name       string
		args       string
		wantKind   string
		wantFields func(t *testing.T, action BrowserHarnessActionRequest)
	}{
		{
			name:     "browser_cdp",
			args:     `{"method":"Target.getTargets","params":{"discover":true}}`,
			wantKind: "cdp",
			wantFields: func(t *testing.T, action BrowserHarnessActionRequest) {
				t.Helper()
				if action.Method != "Target.getTargets" || action.Params["discover"] != true {
					t.Fatalf("cdp action = %#v", action)
				}
			},
		},
		{
			name:     "browser_console",
			args:     `{"expression":"document.title"}`,
			wantKind: "console",
			wantFields: func(t *testing.T, action BrowserHarnessActionRequest) {
				t.Helper()
				if action.Expression != "document.title" {
					t.Fatalf("console action = %#v", action)
				}
			},
		},
		{
			name:     "browser_vision",
			args:     `{"question":"what is visible?"}`,
			wantKind: "vision",
			wantFields: func(t *testing.T, action BrowserHarnessActionRequest) {
				t.Helper()
				if action.Question != "what is visible?" {
					t.Fatalf("vision action = %#v", action)
				}
			},
		},
		{
			name:     "browser_dialog",
			args:     `{"action":"accept","prompt_text":"ok"}`,
			wantKind: "dialog",
			wantFields: func(t *testing.T, action BrowserHarnessActionRequest) {
				t.Helper()
				if action.DialogAction != "accept" || action.PromptText != "ok" {
					t.Fatalf("dialog action = %#v", action)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &recordingHarnessRunner{result: BrowserHarnessProcessResult{Stdout: []byte(`{"ok":true}`)}}
			tool := NewBrowserHarnessTool(tt.name, BrowserHarnessToolsConfig{
				Runner: runner,
				Budget: ToolResultBudgetConfig{OutputDir: t.TempDir(), TextBudgetBytes: 4096, PreviewBytes: 512},
			})
			if _, err := tool.Execute(context.Background(), json.RawMessage(tt.args)); err != nil {
				t.Fatalf("Execute: %v", err)
			}
			action := decodeHarnessAction(t, runner.argv[2])
			if action.Kind != tt.wantKind {
				t.Fatalf("action kind = %q, want %q: %#v", action.Kind, tt.wantKind, action)
			}
			if tt.wantFields != nil {
				tt.wantFields(t, action)
			}
		})
	}
}

func TestBrowserHarnessLegacyPythonCommandStillExplicit(t *testing.T) {
	runner := &recordingHarnessRunner{result: BrowserHarnessProcessResult{Stdout: []byte(`{"ok":true}`)}}
	tool := NewBrowserHarnessTool("browser_navigate", BrowserHarnessToolsConfig{
		Command:  legacyBrowserHarnessCommand,
		Protocol: BrowserHarnessProtocolLegacy,
		Runner:   runner,
		Budget:   ToolResultBudgetConfig{OutputDir: t.TempDir(), TextBudgetBytes: 4096, PreviewBytes: 512},
	})
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.com","task_id":"Browser Task"}`)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got, want := strings.Join(runner.argv[:2], "\x00"), "browser-harness\x00-c"; got != want {
		t.Fatalf("argv prefix = %q, want %q", got, want)
	}
	if code := runner.argv[2]; !strings.Contains(code, `new_tab("https://example.com")`) || strings.Contains(code, "goto_url(") {
		t.Fatalf("legacy navigate code should preserve new_tab and avoid goto_url:\n%s", code)
	}
}

func decodeHarnessAction(t *testing.T, raw string) BrowserHarnessActionRequest {
	t.Helper()
	var action BrowserHarnessActionRequest
	if err := json.Unmarshal([]byte(raw), &action); err != nil {
		t.Fatalf("decode action JSON: %v\n%s", err, raw)
	}
	return action
}
