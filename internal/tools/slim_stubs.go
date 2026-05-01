//go:build slim

package tools

import (
	"context"
	"encoding/json"
	"time"
)

// slimTextToSpeechTool is a no-op TTS tool for slim builds.
type slimTextToSpeechTool struct{}

func (s *slimTextToSpeechTool) Name() string { return "text_to_speech" }
func (s *slimTextToSpeechTool) Description() string {
	return "Text-to-speech is not available in this slim build"
}
func (s *slimTextToSpeechTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`)
}
func (s *slimTextToSpeechTool) Timeout() time.Duration { return 0 }
func (s *slimTextToSpeechTool) Execute(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	result := map[string]any{
		"success":  false,
		"evidence": "tts_unavailable_slim_build",
		"error":    "TTS not available (slim build)",
	}
	return json.Marshal(result)
}

// NewTextToSpeechTool returns a no-op TTS tool for slim builds.
func NewTextToSpeechTool(any) *slimTextToSpeechTool {
	return &slimTextToSpeechTool{}
}

// slimTranscriptionTool is a no-op transcription tool for slim builds.
type slimTranscriptionTool struct{}

func (s *slimTranscriptionTool) Name() string { return "transcribe_audio" }
func (s *slimTranscriptionTool) Description() string {
	return "Audio transcription is not available in this slim build"
}
func (s *slimTranscriptionTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"audio_path":{"type":"string"}},"required":["audio_path"]}`)
}
func (s *slimTranscriptionTool) Timeout() time.Duration { return 0 }
func (s *slimTranscriptionTool) Execute(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	result := map[string]any{
		"success":  false,
		"evidence": "stt_unavailable_slim_build",
		"error":    "Transcription not available (slim build)",
	}
	return json.Marshal(result)
}

// NewTranscriptionTool returns a no-op transcription tool for slim builds.
func NewTranscriptionTool(any) *slimTranscriptionTool {
	return &slimTranscriptionTool{}
}

// slimVoiceModeTool reports voice mode is unavailable.
type slimVoiceModeTool struct{}

func (s *slimVoiceModeTool) Name() string    { return "voice_mode" }
func (s *slimVoiceModeTool) Description() string {
	return "Voice mode is not available in this slim build"
}
func (s *slimVoiceModeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"chat_id":{"type":"string"},"action":{"type":"string"}},"required":["chat_id","action"]}`)
}
func (s *slimVoiceModeTool) Timeout() time.Duration { return 0 }
func (s *slimVoiceModeTool) Execute(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	result := map[string]any{
		"success":  false,
		"evidence": "voice_mode_unavailable_slim_build",
		"error":    "Voice mode not available (slim build)",
	}
	return json.Marshal(result)
}

// NewVoiceModeTool returns a no-op voice mode tool for slim builds.
func NewVoiceModeTool(any) *slimVoiceModeTool {
	return &slimVoiceModeTool{}
}

// slimImageGenTool reports image generation is unavailable.
type slimImageGenTool struct{}

func (s *slimImageGenTool) Name() string    { return "image_generate" }
func (s *slimImageGenTool) Description() string {
	return "Image generation is not available in this slim build"
}
func (s *slimImageGenTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"prompt":{"type":"string"}},"required":["prompt"]}`)
}
func (s *slimImageGenTool) Timeout() time.Duration { return 0 }
func (s *slimImageGenTool) Execute(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	result := map[string]any{
		"success":  false,
		"evidence": "image_gen_unavailable_slim_build",
		"error":    "Image generation not available (slim build)",
	}
	return json.Marshal(result)
}

// NewImageGenTool returns a no-op image gen tool for slim builds.
func NewImageGenTool(any) *slimImageGenTool {
	return &slimImageGenTool{}
}

// Ensure compile-time tool conformance.
var _ Tool = (*slimTextToSpeechTool)(nil)
var _ Tool = (*slimTranscriptionTool)(nil)
var _ Tool = (*slimVoiceModeTool)(nil)
var _ Tool = (*slimImageGenTool)(nil)
