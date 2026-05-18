package gateway

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestInboundEvent_ChatKey(t *testing.T) {
	tests := []struct {
		name string
		e    InboundEvent
		want string
	}{
		{"telegram", InboundEvent{Platform: "telegram", ChatID: "42"}, "telegram:42"},
		{"discord", InboundEvent{Platform: "discord", ChatID: "987654321"}, "discord:987654321"},
		{"empty chat id", InboundEvent{Platform: "telegram", ChatID: ""}, "telegram:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.ChatKey(); got != tt.want {
				t.Errorf("ChatKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEventKind_String(t *testing.T) {
	tests := []struct {
		k    EventKind
		want string
	}{
		{EventUnknown, "unknown"},
		{EventSubmit, "submit"},
		{EventCancel, "cancel"},
		{EventReset, "reset"},
		{EventStart, "start"},
	}
	for _, tt := range tests {
		if got := tt.k.String(); got != tt.want {
			t.Errorf("EventKind(%d).String() = %q, want %q", tt.k, got, tt.want)
		}
	}
}

func TestInboundEvent_SubmitTextAddsChannelNeutralAudioTranscriptionHint(t *testing.T) {
	audioPath := filepath.Join(t.TempDir(), "voice.ogg")
	ev := InboundEvent{
		Platform: "discord",
		ChatID:   "channel-42",
		Kind:     EventSubmit,
		Text:     "please inspect this voice note",
		Attachments: []Attachment{{
			Kind:      "audio",
			URL:       audioPath,
			MediaType: "audio/ogg",
			FileName:  "voice.ogg",
			SourceID:  "discord-attachment-1",
			SizeBytes: 321,
		}},
	}

	got := ev.SubmitText()
	for _, want := range []string{
		"Attachments:",
		"- audio voice.ogg: " + audioPath + " (mediaType=audio/ogg, sourceId=discord-attachment-1, sizeBytes=321)",
		"Audio transcription:",
		"- transcribe_audio audio_path=" + strconv.Quote(audioPath) + " (kind=audio, fileName=voice.ogg, mediaType=audio/ogg)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("SubmitText() missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(strings.ToLower(got), "telegram") {
		t.Fatalf("SubmitText() = %q, want channel-neutral audio hint", got)
	}
}

func TestInboundEvent_SubmitTextDoesNotTranscribeRemoteAudioURL(t *testing.T) {
	ev := InboundEvent{
		Platform: "fakechannel",
		ChatID:   "channel-42",
		Kind:     EventSubmit,
		Text:     "please inspect",
		Attachments: []Attachment{{
			Kind:      "audio",
			URL:       "https://cdn.example.test/audio/voice.ogg",
			MediaType: "audio/ogg",
			FileName:  "voice.ogg",
		}},
	}

	got := ev.SubmitText()
	if !strings.Contains(got, "- audio voice.ogg: https://cdn.example.test/audio/voice.ogg") {
		t.Fatalf("SubmitText() = %q, want remote attachment evidence preserved", got)
	}
	if strings.Contains(got, "transcribe_audio") {
		t.Fatalf("SubmitText() = %q, must not present remote URL as local transcribe_audio path", got)
	}
}
