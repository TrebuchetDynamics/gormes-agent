package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestHermesVoiceRuntimeRecordKeyStartStopTranscribesIntoComposer(t *testing.T) {
	sub := &nopSubmitter{}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1, SessionID: "sess-voice"}
	recorder := &fakeVoiceRecorder{stopArtifact: VoiceAudioArtifact{ID: "clip-1", Bytes: []byte("wav")}}
	stt := &fakeVoiceTranscriber{text: "transcribed voice prompt"}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{
		VoiceRecordKey: "ctrl+o",
		VoiceToggle: func(req VoiceToggleRequest) (VoiceToggleResult, error) {
			return VoiceToggleResult{Enabled: true, RecordKey: "ctrl+o", Details: "Audio: fake recorder\nSTT: fake"}, nil
		},
		VoiceRuntime: VoiceRuntime{Recorder: recorder, Transcriber: stt},
	})
	m.frame.SessionID = "sess-voice"
	m.editor.SetValue("draft stays")

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	updated := next.(Model)
	runTestCmd(t, cmd)
	if recorder.starts != 1 || recorder.stops != 0 || stt.calls != 0 {
		t.Fatalf("after start recorder starts=%d stops=%d stt=%d", recorder.starts, recorder.stops, stt.calls)
	}
	if !updated.voiceRecording || sub.calls != 0 || updated.editor.Value() != "draft stays" {
		t.Fatalf("start state recording=%v submit=%d draft=%q", updated.voiceRecording, sub.calls, updated.editor.Value())
	}
	if !strings.Contains(updated.statusMessage, "voice recording") {
		t.Fatalf("start status = %q", updated.statusMessage)
	}

	next, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	updated = next.(Model)
	runTestCmd(t, cmd)
	if recorder.stops != 1 || stt.calls != 1 || sub.calls != 0 {
		t.Fatalf("after stop recorder stops=%d stt=%d submit=%d", recorder.stops, stt.calls, sub.calls)
	}
	if updated.voiceRecording || updated.voiceProcessing {
		t.Fatalf("stop flags recording=%v processing=%v", updated.voiceRecording, updated.voiceProcessing)
	}
	if got := updated.editor.Value(); got != "draft stays\ntranscribed voice prompt" {
		t.Fatalf("editor transcript = %q", got)
	}
}

func TestHermesVoiceRuntimeDegradedPathsKeepDraftSafe(t *testing.T) {
	sub := &nopSubmitter{}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1, SessionID: "sess-voice"}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{
		VoiceRecordKey: "ctrl+o",
		VoiceToggle: func(req VoiceToggleRequest) (VoiceToggleResult, error) {
			return VoiceToggleResult{Enabled: true, RecordKey: "ctrl+o"}, nil
		},
		VoiceRuntime: VoiceRuntime{Transcriber: &fakeVoiceTranscriber{}},
	})
	m.editor.SetValue("safe draft")

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	updated := next.(Model)
	runTestCmd(t, cmd)
	if sub.calls != 0 || updated.editor.Value() != "safe draft" || updated.voiceRecording || updated.voiceProcessing {
		t.Fatalf("degraded state submit=%d draft=%q recording=%v processing=%v", sub.calls, updated.editor.Value(), updated.voiceRecording, updated.voiceProcessing)
	}
	if !strings.Contains(updated.statusMessage, "voice_recorder_unavailable") {
		t.Fatalf("status = %q, want recorder evidence", updated.statusMessage)
	}
}

func TestHermesVoiceRuntimeTTSPlaybackOnAssistantFrame(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	playback := &fakeVoicePlayback{}
	m := NewModelWithOptions(frames, func(string) {}, func() {}, Options{
		VoiceRuntime: VoiceRuntime{Playback: playback},
		VoiceToggle: func(req VoiceToggleRequest) (VoiceToggleResult, error) {
			return VoiceToggleResult{Enabled: true, TTS: true, RecordKey: "ctrl+b"}, nil
		},
	})

	next, _ := m.Update(frameMsg(kernel.RenderFrame{Seq: 2, Phase: kernel.PhaseIdle, History: []llm.Message{{Role: "assistant", Content: "hello out loud"}}}))
	updated := next.(Model)
	if playback.calls != 1 || playback.texts[0] != "hello out loud" {
		t.Fatalf("playback calls=%d texts=%v", playback.calls, playback.texts)
	}
	if !strings.Contains(updated.statusMessage, "voice_tts_played") {
		t.Fatalf("status = %q", updated.statusMessage)
	}

	next, _ = updated.Update(frameMsg(kernel.RenderFrame{Seq: 2, Phase: kernel.PhaseIdle, History: []llm.Message{{Role: "assistant", Content: "hello out loud"}}}))
	if playback.calls != 1 {
		t.Fatalf("duplicate frame replayed TTS: %d", playback.calls)
	}
}

type fakeVoiceRecorder struct {
	starts       int
	stops        int
	startErr     error
	stopErr      error
	stopArtifact VoiceAudioArtifact
}

func (f *fakeVoiceRecorder) Start(context.Context, VoiceRecordRequest) (VoiceRecordEvidence, error) {
	f.starts++
	if f.startErr != nil {
		return VoiceRecordEvidence{}, f.startErr
	}
	return VoiceRecordEvidence{Code: "voice_recording", Message: "voice recording started"}, nil
}

func (f *fakeVoiceRecorder) Stop(context.Context, VoiceRecordRequest) (VoiceAudioArtifact, VoiceRecordEvidence, error) {
	f.stops++
	if f.stopErr != nil {
		return VoiceAudioArtifact{}, VoiceRecordEvidence{}, f.stopErr
	}
	return f.stopArtifact, VoiceRecordEvidence{Code: "voice_recorded", Message: "voice recording stopped"}, nil
}

type fakeVoiceTranscriber struct {
	calls int
	text  string
	err   error
}

func (f *fakeVoiceTranscriber) Transcribe(context.Context, VoiceAudioArtifact) (VoiceTranscript, VoiceRecordEvidence, error) {
	f.calls++
	if f.err != nil {
		return VoiceTranscript{}, VoiceRecordEvidence{}, f.err
	}
	return VoiceTranscript{Text: f.text}, VoiceRecordEvidence{Code: "voice_transcribed", Message: "voice transcribed"}, nil
}

type fakeVoicePlayback struct {
	calls int
	texts []string
	err   error
}

func (f *fakeVoicePlayback) Speak(_ context.Context, text string) (VoiceRecordEvidence, error) {
	f.calls++
	f.texts = append(f.texts, text)
	if f.err != nil {
		return VoiceRecordEvidence{}, f.err
	}
	return VoiceRecordEvidence{Code: "voice_tts_played", Message: "voice_tts_played"}, nil
}

var _ VoiceRecorder = (*fakeVoiceRecorder)(nil)
var _ VoiceTranscriber = (*fakeVoiceTranscriber)(nil)
var _ VoicePlayback = (*fakeVoicePlayback)(nil)
