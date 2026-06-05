package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestTreeSlashOpensLineagePageWithFiltersAndLabels(t *testing.T) {
	sub := &nopSubmitter{}
	var gotReq SessionTreeRequest
	m := newTreeSlashModel(sub, func(ctx context.Context, req SessionTreeRequest) (SessionTreeResult, error) {
		if ctx == nil {
			t.Fatal("tree context is nil")
		}
		gotReq = req
		return SessionTreeResult{Entries: []SessionTreeEntry{
			{ID: "sess-root", Title: "Root Plan", LineageKind: "primary", Labels: []string{"pinned"}, UpdatedAt: 100, Messages: []SessionTreeMessage{{ID: 1, Role: "user", Content: "root prompt"}, {ID: 2, Role: "tool", Content: "tool noise"}}},
			{ID: "sess-compress", ParentID: "sess-root", Title: "Compressed", LineageKind: "compression", Depth: 1, UpdatedAt: 110},
			{ID: "sess-fork", ParentID: "sess-root", Title: "Branch", LineageKind: "fork", Labels: []string{"review"}, Depth: 1, Active: true, UpdatedAt: 120, Messages: []SessionTreeMessage{{ID: 3, Role: "user", Content: "fork prompt"}}},
		}}, nil
	}, nil, nil)

	m = enterSlashDispatchBehavior(t, m, "/tree --filter user-only")
	if sub.calls != 0 {
		t.Fatalf("/tree reached Submitter %d time(s), want 0", sub.calls)
	}
	if gotReq.Filter != "user-only" || gotReq.ActiveSessionID != "sess-current" {
		t.Fatalf("tree request = %+v, want filter user-only and active session", gotReq)
	}
	assertTreePageContains(t, m, "Session Tree", "Root Plan", "labels: pinned", "↳ compression", "*   ↳ fork", "fork prompt")
	if strings.Contains(m.transientPage.Body, "tool noise") {
		t.Fatalf("tree page rendered tool noise despite user-only request:\n%s", m.transientPage.Body)
	}
	if !strings.Contains(m.statusMessage, "tree opened") {
		t.Fatalf("tree status = %q, want opened", m.statusMessage)
	}
}

func TestTreeSlashLabelSetClearAndRestoreEditablePrompt(t *testing.T) {
	sub := &nopSubmitter{}
	var labelReqs []SessionTreeLabelRequest
	var restoreReq SessionTreeRestoreRequest
	m := newTreeSlashModel(sub,
		func(context.Context, SessionTreeRequest) (SessionTreeResult, error) { return SessionTreeResult{}, nil },
		func(_ context.Context, req SessionTreeLabelRequest) (SessionTreeLabelResult, error) {
			labelReqs = append(labelReqs, req)
			return SessionTreeLabelResult{SessionID: req.SessionID, Labels: []string{"pinned"}}, nil
		},
		func(_ context.Context, req SessionTreeRestoreRequest) (SessionTreeRestoreResult, error) {
			restoreReq = req
			return SessionTreeRestoreResult{Text: "revise this old prompt", Editable: true}, nil
		},
	)

	m = enterSlashDispatchBehavior(t, m, "/tree label sess-root pinned")
	m = enterSlashDispatchBehavior(t, m, "/tree unlabel sess-root pinned")
	if len(labelReqs) != 2 || labelReqs[0].Action != "set" || labelReqs[0].Label != "pinned" || labelReqs[1].Action != "clear" {
		t.Fatalf("label requests = %+v, want set then clear", labelReqs)
	}
	if !strings.Contains(m.statusMessage, "tree: labels for sess-root: pinned") {
		t.Fatalf("label status = %q, want label evidence", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, m, "/tree restore sess-root 7")
	if restoreReq.SessionID != "sess-root" || restoreReq.MessageID != 7 {
		t.Fatalf("restore request = %+v, want sess-root turn 7", restoreReq)
	}
	if got := m.editor.Value(); got != "revise this old prompt" {
		t.Fatalf("editor after restore = %q, want restored prompt", got)
	}
	if got := m.SessionID(); got != "sess-current" {
		t.Fatalf("restore changed active session to %q, want sess-current", got)
	}
	if !strings.Contains(m.statusMessage, "tree: restored editable prompt from sess-root#7") {
		t.Fatalf("restore status = %q", m.statusMessage)
	}
}

func TestTreeSlashReplayUnavailableDoesNotCorruptActiveSession(t *testing.T) {
	m := newTreeSlashModel(&nopSubmitter{},
		func(context.Context, SessionTreeRequest) (SessionTreeResult, error) { return SessionTreeResult{}, nil },
		nil,
		func(context.Context, SessionTreeRestoreRequest) (SessionTreeRestoreResult, error) {
			return SessionTreeRestoreResult{Editable: false, Evidence: "non_user_entry"}, nil
		},
	)
	m = enterSlashDispatchBehavior(t, m, "/tree restore sess-root 2")
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor after replay-unavailable = %q, want empty", got)
	}
	if got := m.SessionID(); got != "sess-current" {
		t.Fatalf("replay-unavailable changed session = %q", got)
	}
	if !strings.Contains(m.statusMessage, "tree: replay unavailable: non_user_entry") {
		t.Fatalf("status = %q, want typed replay evidence", m.statusMessage)
	}
}

func newTreeSlashModel(sub *nopSubmitter, tree SessionTreeFunc, label SessionTreeLabelFunc, restore SessionTreeRestoreFunc) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-current"}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, SessionTree: tree, SessionTreeLabel: label, SessionTreeRestore: restore})
	m.frame = kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-current"}
	m.width = 96
	m.height = 28
	return m
}

func assertTreePageContains(t *testing.T, m Model, want ...string) {
	t.Helper()
	if m.transientPage == nil {
		t.Fatal("tree slash did not open transient page")
	}
	for _, item := range want {
		if !strings.Contains(m.transientPage.Title+"\n"+m.transientPage.Body, item) {
			t.Fatalf("tree page missing %q:\n%s\n%s", item, m.transientPage.Title, m.transientPage.Body)
		}
	}
}
