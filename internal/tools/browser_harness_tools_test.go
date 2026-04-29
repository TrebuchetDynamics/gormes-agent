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
	if got := strings.Join(runner.argv, "\x00"); !strings.Contains(got, "browser-harness\x00-c\x00") {
		t.Fatalf("argv = %q, want browser-harness -c", got)
	}
	code := runner.argv[2]
	if !strings.Contains(code, `new_tab("https://example.com")`) {
		t.Fatalf("navigate code = %s, want new_tab(url)", code)
	}
	if strings.Contains(code, "goto_url(") {
		t.Fatalf("navigate code must not use goto_url: %s", code)
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
		wantCode   []string
		shouldDeny bool
	}{
		{
			name:     "browser_click",
			args:     `{"ref":"@e3","task_id":"task-1"}`,
			wantCode: []string{`_gormes_ref_center("@e3")`, "click_at_xy("},
		},
		{
			name:     "browser_type",
			args:     `{"ref":"@e3","text":"hello","task_id":"task-1"}`,
			wantCode: []string{`_gormes_ref_center("@e3")`, `type_text("hello")`},
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
			for _, want := range tc.wantCode {
				if !strings.Contains(runner.argv[2], want) {
					t.Fatalf("code missing %q:\n%s", want, runner.argv[2])
				}
			}
		})
	}
}

func TestBrowserHarnessCDPConsoleAndVisionBuildExpectedSnippets(t *testing.T) {
	tests := []struct {
		name     string
		args     string
		contains []string
	}{
		{name: "browser_cdp", args: `{"method":"Target.getTargets","params":{"discover":true}}`, contains: []string{`cdp("Target.getTargets"`, `_gormes_params = json.loads(`}},
		{name: "browser_console", args: `{"expression":"document.title"}`, contains: []string{`js("document.title")`}},
		{name: "browser_vision", args: `{"question":"what is visible?"}`, contains: []string{"capture_screenshot(", "browser_harness_screenshot_captured"}},
		{name: "browser_dialog", args: `{"action":"accept","prompt_text":"ok"}`, contains: []string{`Page.handleJavaScriptDialog`, `promptText`}},
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
			for _, want := range tt.contains {
				if !strings.Contains(runner.argv[2], want) {
					t.Fatalf("%s code missing %q:\n%s", tt.name, want, runner.argv[2])
				}
			}
		})
	}
}
