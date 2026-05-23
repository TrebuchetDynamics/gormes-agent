package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

type recordingVoiceToggle struct {
	calls  int
	gotReq VoiceToggleRequest
	result VoiceToggleResult
	err    error
}

func (r *recordingVoiceToggle) call(req VoiceToggleRequest) (VoiceToggleResult, error) {
	r.calls++
	r.gotReq = req
	if r.err != nil {
		return VoiceToggleResult{}, r.err
	}
	return r.result, nil
}

func TestVoiceSlashStatusUpdatesRecordKey(t *testing.T) {
	rec := &recordingVoiceToggle{result: VoiceToggleResult{
		Enabled:   true,
		TTS:       false,
		RecordKey: "ctrl+space",
		Details:   "Audio: OK\nSTT: not configured\nTTS: not configured",
	}}
	sub := &nopSubmitter{}
	m := newVoiceSlashModel(sub, rec.call, "ctrl+b")
	m.frame.SessionID = "sess-voice"

	m = enterSlashDispatchBehavior(t, m, "/voice status")

	if sub.calls != 0 {
		t.Fatalf("/voice status reached Submitter %d time(s), want 0", sub.calls)
	}
	if rec.calls != 1 {
		t.Fatalf("VoiceToggle calls = %d, want 1", rec.calls)
	}
	wantReq := VoiceToggleRequest{Action: "status", SessionID: "sess-voice"}
	if !reflect.DeepEqual(rec.gotReq, wantReq) {
		t.Fatalf("VoiceToggle request = %#v, want %#v", rec.gotReq, wantReq)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /voice status = %q, want cleared", got)
	}
	assertVoicePageContains(t, m,
		"Voice Mode Status",
		"Mode:       ON",
		"TTS:        OFF",
		"Record key: Ctrl+Space",
		"Requirements:",
		"Audio: OK",
		"STT: not configured",
		"TTS: not configured",
	)
	if m.voiceRecordKey != "ctrl+space" {
		t.Fatalf("model voiceRecordKey = %q, want ctrl+space", m.voiceRecordKey)
	}
	if decision := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeySpace, Ctrl: true}, HermesInputState{VoiceRecordKey: m.voiceRecordKey}); decision.Action != HermesActionToggleVoiceRecording {
		t.Fatalf("Ctrl+Space after /voice status = %v, want voice toggle", decision.Action)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/voice status fell through to fallback: %q", m.statusMessage)
	}
}

func TestVoiceSlashToggleAndTTSDoNotClobberMissingRecordKey(t *testing.T) {
	rec := &recordingVoiceToggle{result: VoiceToggleResult{Enabled: true, TTS: false, RecordKey: "alt+r"}}
	m := newVoiceSlashModel(&nopSubmitter{}, rec.call, "ctrl+b")

	m = enterSlashDispatchBehavior(t, m, "/voice on")

	if rec.gotReq.Action != "on" {
		t.Fatalf("VoiceToggle action = %q, want on", rec.gotReq.Action)
	}
	if m.voiceRecordKey != "alt+r" {
		t.Fatalf("model voiceRecordKey after /voice on = %q, want alt+r", m.voiceRecordKey)
	}
	assertVoicePageContains(t, m,
		"Voice mode enabled",
		"Alt+R to start/stop recording",
		"/voice tts  to toggle speech output",
		"/voice off  to disable voice mode",
	)

	rec.result = VoiceToggleResult{Enabled: true, TTS: true}
	m = enterSlashDispatchBehavior(t, m, "/voice tts")
	if rec.gotReq.Action != "tts" {
		t.Fatalf("VoiceToggle action = %q, want tts", rec.gotReq.Action)
	}
	if m.voiceRecordKey != "alt+r" {
		t.Fatalf("model voiceRecordKey after missing record_key = %q, want cached alt+r", m.voiceRecordKey)
	}
	assertVoicePageContains(t, m, "Voice TTS enabled.")

	rec.result = VoiceToggleResult{Enabled: false, TTS: false}
	m = enterSlashDispatchBehavior(t, m, "/voice off")
	assertVoicePageContains(t, m, "Voice mode disabled.")
}

func TestVoiceSlashUnavailableAdapterConsumesWithRequirements(t *testing.T) {
	sub := &nopSubmitter{}
	m := newVoiceSlashModel(sub, nil, "ctrl+o")

	m = enterSlashDispatchBehavior(t, m, "/voice status")

	if sub.calls != 0 {
		t.Fatalf("/voice status with nil adapter reached Submitter %d time(s), want 0", sub.calls)
	}
	if m.voiceRecordKey != "ctrl+o" {
		t.Fatalf("nil adapter clobbered voiceRecordKey = %q, want ctrl+o", m.voiceRecordKey)
	}
	assertVoicePageContains(t, m,
		"Voice Mode Status",
		"Mode:       OFF",
		"TTS:        OFF",
		"Record key: Ctrl+B",
		"Requirements:",
		"voice adapter unavailable",
	)
}

func assertVoicePageContains(t *testing.T, m Model, wants ...string) {
	t.Helper()
	if m.transientPage == nil {
		t.Fatal("voice page = nil, want rendered voice evidence")
	}
	if m.transientPage.Title != "Voice" {
		t.Fatalf("voice page title = %q, want Voice", m.transientPage.Title)
	}
	body := m.transientPage.Body
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Fatalf("voice page body missing %q:\n%s", want, body)
		}
	}
}

func newVoiceSlashModel(sub *nopSubmitter, fn VoiceToggleFunc, recordKey string) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, VoiceRecordKey: recordKey, VoiceToggle: fn})
	m.frame.Phase = kernel.PhaseIdle
	return m
}
