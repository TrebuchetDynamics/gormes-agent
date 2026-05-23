package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

type recordingToolsConfigure struct {
	calls  int
	gotReq ToolsConfigureRequest
	result ToolsConfigureResult
	err    error
}

func (r *recordingToolsConfigure) call(req ToolsConfigureRequest) (ToolsConfigureResult, error) {
	r.calls++
	r.gotReq = req
	if r.err != nil {
		return ToolsConfigureResult{}, r.err
	}
	return r.result, nil
}

func TestToolsSlashEnableDisableAdapter(t *testing.T) {
	rec := &recordingToolsConfigure{result: ToolsConfigureResult{
		Changed:        []string{"web", "terminal"},
		Unknown:        []string{"unknown_toolset"},
		MissingServers: []string{"github"},
		Reset:          true,
	}}
	sub := &nopSubmitter{}
	m := newToolsSlashModel(sub, rec.call)
	m.frame.SessionID = "sess-tools"

	m = enterSlashDispatchBehavior(t, m, "/tools enable web terminal")

	if sub.calls != 0 {
		t.Fatalf("/tools enable reached Submitter %d time(s), want 0", sub.calls)
	}
	if rec.calls != 1 {
		t.Fatalf("ToolsConfigure calls = %d, want 1", rec.calls)
	}
	wantReq := ToolsConfigureRequest{Action: "enable", Names: []string{"web", "terminal"}, SessionID: "sess-tools"}
	if !reflect.DeepEqual(rec.gotReq, wantReq) {
		t.Fatalf("ToolsConfigure request = %#v, want %#v", rec.gotReq, wantReq)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /tools enable = %q, want cleared", got)
	}
	assertToolsPageContains(t, m, []string{
		"enabled: web, terminal",
		"unknown toolsets: unknown_toolset",
		"missing MCP servers: github",
		"session reset. new tool configuration is active.",
	})
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/tools enable fell through to fallback: %q", m.statusMessage)
	}
}

func TestToolsSlashDisableRendersDisabledEvidence(t *testing.T) {
	rec := &recordingToolsConfigure{result: ToolsConfigureResult{Changed: []string{"web"}, Reset: true}}
	m := newToolsSlashModel(&nopSubmitter{}, rec.call)

	m = enterSlashDispatchBehavior(t, m, "/tools disable web")

	if rec.gotReq.Action != "disable" {
		t.Fatalf("ToolsConfigure action = %q, want disable", rec.gotReq.Action)
	}
	assertToolsPageContains(t, m, []string{"disabled: web", "session reset. new tool configuration is active."})
}

func TestToolsSlashUsageAndUnavailableEvidence(t *testing.T) {
	for _, input := range []string{"/tools enable", "/tools disable"} {
		t.Run(input, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := enterSlashDispatchBehavior(t, newToolsSlashModel(sub, nil), input)
			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", input, sub.calls)
			}
			for _, want := range []string{
				"usage: " + input + " <name> [name ...]",
				"built-in toolset: " + input + " web",
				"MCP tool: " + input + " github:create_issue",
			} {
				if !strings.Contains(m.statusMessage, want) {
					t.Fatalf("status after %s missing %q:\n%s", input, want, m.statusMessage)
				}
			}
		})
	}

	sub := &nopSubmitter{}
	m := enterSlashDispatchBehavior(t, newToolsSlashModel(sub, nil), "/tools enable web")
	if sub.calls != 0 {
		t.Fatalf("/tools enable with nil adapter reached Submitter %d time(s), want 0", sub.calls)
	}
	if !strings.Contains(m.statusMessage, "tools: configuration unavailable") {
		t.Fatalf("status after nil adapter = %q, want unavailable evidence", m.statusMessage)
	}
	if m.transientPage != nil {
		t.Fatalf("nil adapter tools page = %+v, want none", *m.transientPage)
	}
}

func assertToolsPageContains(t *testing.T, m Model, wants []string) {
	t.Helper()
	if m.transientPage == nil {
		t.Fatal("tools page = nil, want rendered tools evidence")
	}
	if m.transientPage.Title != "Tools" {
		t.Fatalf("tools page title = %q, want Tools", m.transientPage.Title)
	}
	body := m.transientPage.Body
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Fatalf("tools page body missing %q:\n%s", want, body)
		}
	}
}

func newToolsSlashModel(sub *nopSubmitter, fn ToolsConfigureFunc) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, ToolsConfigure: fn})
	m.frame.Phase = kernel.PhaseIdle
	return m
}
