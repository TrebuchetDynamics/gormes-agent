package navivox

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNavivoxRunRecordVoiceTurnPreservesEvidenceAndUnknownUsage(t *testing.T) {
	now := time.Unix(1_700_100_000, 0).UTC()
	record := NewRunRecord("req-voice", "s-voice", "transcribed voice command", map[string]any{
		"input_kind":          "voice",
		"audio_duration_ms":   2300,
		"audio_codec":         "audio/opus",
		"stt_provider":        "device",
		"stt_status":          "available",
		"server_stt_provider": "none",
		"tts_provider":        "local",
		"tts_voice_id":        "calm-en",
		"tts_status":          "queued",
		"raw_audio_bytes":     "raw-audio-bytes-must-not-persist",
		"provider_api_key":    "secret-token-must-not-persist",
		"audio_path":          "/private/raw.wav",
	}, now)

	if record.RunID != "req-voice" || record.SessionID != "s-voice" || record.Status != RunStatusInProgress {
		t.Fatalf("identity/status = %+v", record)
	}
	if len(record.Transcript) != 1 || record.Transcript[0].Role != "user" || record.Transcript[0].Text != "transcribed voice command" {
		t.Fatalf("transcript = %+v", record.Transcript)
	}
	if record.Voice == nil {
		t.Fatal("voice evidence missing")
	}
	if record.Voice.DeviceTranscript != "transcribed voice command" {
		t.Fatalf("device transcript = %q", record.Voice.DeviceTranscript)
	}
	if record.Voice.Audio.DurationMS != 2300 || record.Voice.Audio.Codec != "audio/opus" {
		t.Fatalf("audio metadata = %+v", record.Voice.Audio)
	}
	if record.Voice.Audio.RawAudioStored || record.Voice.Audio.Retention != "not_stored" {
		t.Fatalf("raw audio retention = %+v, want not_stored without raw persistence", record.Voice.Audio)
	}
	if record.ProviderUsage.Status != EvidenceUnknown || record.ProviderCost.Status != EvidenceUnknown {
		t.Fatalf("usage/cost = %+v/%+v, want explicit unknown evidence", record.ProviderUsage, record.ProviderCost)
	}

	body, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"raw-audio-bytes-must-not-persist", "secret-token-must-not-persist", "/private/raw.wav"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("run record leaked %q: %s", forbidden, body)
		}
	}
}

func TestNavivoxRunRecordToolMetadataRejectsCollapsedSensitiveKeys(t *testing.T) {
	now := time.Unix(1_700_100_050, 0).UTC()
	record := NewRunRecord("req-tool", "s-tool", "inspect workspace", nil, now)
	record.AppendToolEvent("tool-1", "read_file", "finished", "Read README", map[string]any{
		"artifact_ref":     "artifact://readme-summary",
		"raw audio bytes":  "raw-audio-bytes-must-not-persist",
		"audio bytes path": "/private/raw.wav",
	}, now.Add(time.Second))

	if len(record.ToolEvents) != 1 {
		t.Fatalf("tool events = %+v", record.ToolEvents)
	}
	if record.ToolEvents[0].Metadata["artifact_ref"] != "artifact://readme-summary" {
		t.Fatalf("tool metadata = %+v", record.ToolEvents[0].Metadata)
	}
	body, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"raw-audio-bytes-must-not-persist", "/private/raw.wav"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("tool metadata leaked %q: %s", forbidden, body)
		}
	}
}

func TestNavivoxRunRecordToolTimelineAndCompletion(t *testing.T) {
	now := time.Unix(1_700_100_100, 0).UTC()
	record := NewRunRecord("req-text", "s-text", "inspect workspace", map[string]any{}, now)
	record.AppendToolEvent("tool-1", "read_file", "finished", "Read README", map[string]any{
		"artifact_ref": "artifact://readme-summary",
		"secret_token": "must-not-persist",
	}, now.Add(time.Second))
	record.AppendAssistant("workspace inspected", now.Add(2*time.Second))
	record.Complete(now.Add(3 * time.Second))

	if record.Status != RunStatusCompleted {
		t.Fatalf("status = %q, want completed", record.Status)
	}
	if record.CompletedAt == nil || !record.CompletedAt.Equal(now.Add(3*time.Second)) {
		t.Fatalf("completed_at = %v", record.CompletedAt)
	}
	if len(record.Transcript) != 2 || record.Transcript[1].Role != "assistant" || record.Transcript[1].Text != "workspace inspected" {
		t.Fatalf("transcript = %+v", record.Transcript)
	}
	if len(record.ToolEvents) != 1 || record.ToolEvents[0].ToolCallID != "tool-1" || record.ToolEvents[0].Name != "read_file" || record.ToolEvents[0].Status != "finished" {
		t.Fatalf("tool events = %+v", record.ToolEvents)
	}
	if record.ToolEvents[0].Metadata["artifact_ref"] != "artifact://readme-summary" {
		t.Fatalf("tool metadata = %+v", record.ToolEvents[0].Metadata)
	}
	if _, leaked := record.ToolEvents[0].Metadata["secret_token"]; leaked {
		t.Fatalf("tool metadata leaked secret field: %+v", record.ToolEvents[0].Metadata)
	}
}
