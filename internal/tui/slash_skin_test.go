package tui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

type recordingSkinConfig struct {
	calls  int
	gotReq SkinConfigRequest
	result SkinConfigResult
	err    error
}

func (r *recordingSkinConfig) call(req SkinConfigRequest) (SkinConfigResult, error) {
	r.calls++
	r.gotReq = req
	if r.err != nil {
		return SkinConfigResult{}, r.err
	}
	return r.result, nil
}

func TestSkinSlashGetSetAdapter(t *testing.T) {
	rec := &recordingSkinConfig{result: SkinConfigResult{Name: "default"}}
	sub := &nopSubmitter{}
	m := newSkinSlashModel(sub, rec.call, "default")
	m.frame.SessionID = "sess-skin"

	m = enterSlashDispatchBehavior(t, m, "/skin")

	if sub.calls != 0 {
		t.Fatalf("/skin reached Submitter %d time(s), want 0", sub.calls)
	}
	if rec.calls != 1 {
		t.Fatalf("SkinConfig calls = %d, want 1", rec.calls)
	}
	wantReq := SkinConfigRequest{SessionID: "sess-skin"}
	if !reflect.DeepEqual(rec.gotReq, wantReq) {
		t.Fatalf("SkinConfig request = %#v, want %#v", rec.gotReq, wantReq)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /skin = %q, want cleared", got)
	}
	assertSkinPageContains(t, m, "skin: default")

	rec.result = SkinConfigResult{Name: "ares"}
	m = enterSlashDispatchBehavior(t, m, "/skin ares")

	wantReq = SkinConfigRequest{Name: "ares", SessionID: "sess-skin"}
	if !reflect.DeepEqual(rec.gotReq, wantReq) {
		t.Fatalf("SkinConfig request after set = %#v, want %#v", rec.gotReq, wantReq)
	}
	assertSkinPageContains(t, m, "skin → ares")
	if m.activeSkinName != "ares" {
		t.Fatalf("activeSkinName = %q, want ares", m.activeSkinName)
	}
	if !strings.Contains(m.editor.Prompt, "⚔") {
		t.Fatalf("editor prompt after accepted ares skin = %q, want ares prompt glyph", m.editor.Prompt)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/skin fell through to fallback: %q", m.statusMessage)
	}
}

func TestSkinSlashRejectedAndUnavailableDoNotMutate(t *testing.T) {
	rec := &recordingSkinConfig{err: errors.New("invalid skin zeus")}
	sub := &nopSubmitter{}
	m := newSkinSlashModel(sub, rec.call, "ares")
	beforePrompt := m.editor.Prompt

	m = enterSlashDispatchBehavior(t, m, "/skin zeus")

	if sub.calls != 0 {
		t.Fatalf("/skin zeus reached Submitter %d time(s), want 0", sub.calls)
	}
	if rec.gotReq.Name != "zeus" {
		t.Fatalf("SkinConfig name = %q, want zeus", rec.gotReq.Name)
	}
	if m.activeSkinName != "ares" {
		t.Fatalf("rejected skin mutated activeSkinName = %q, want ares", m.activeSkinName)
	}
	if m.editor.Prompt != beforePrompt {
		t.Fatalf("rejected skin mutated prompt = %q, want %q", m.editor.Prompt, beforePrompt)
	}
	if !strings.Contains(m.statusMessage, "skin: invalid skin zeus") {
		t.Fatalf("status after rejected skin = %q, want invalid evidence", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, newSkinSlashModel(sub, nil, "ares"), "/skin mono")
	if m.activeSkinName != "ares" {
		t.Fatalf("nil adapter mutated activeSkinName = %q, want ares", m.activeSkinName)
	}
	if !strings.Contains(m.statusMessage, "skin: configuration unavailable") {
		t.Fatalf("status after nil skin adapter = %q, want unavailable evidence", m.statusMessage)
	}
}

func assertSkinPageContains(t *testing.T, m Model, wants ...string) {
	t.Helper()
	if m.transientPage == nil {
		t.Fatal("skin page = nil, want rendered skin evidence")
	}
	if m.transientPage.Title != "Skin" {
		t.Fatalf("skin page title = %q, want Skin", m.transientPage.Title)
	}
	body := m.transientPage.Body
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Fatalf("skin page body missing %q:\n%s", want, body)
		}
	}
}

func newSkinSlashModel(sub *nopSubmitter, fn SkinConfigFunc, skinName string) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, SkinName: skinName, SkinConfig: fn})
	m.frame.Phase = kernel.PhaseIdle
	return m
}
