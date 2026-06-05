//go:build !slim

package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// VoiceModeState represents the voice mode for a chat session.
type VoiceModeState int

const (
	VoiceModeOff       VoiceModeState = iota // Voice mode disabled
	VoiceModeVoiceOnly                       // Push-to-talk; responses not spoken
	VoiceModeAll                             // Full voice mode; responses spoken aloud
)

func (s VoiceModeState) String() string {
	switch s {
	case VoiceModeOff:
		return "off"
	case VoiceModeVoiceOnly:
		return "voice_only"
	case VoiceModeAll:
		return "all"
	default:
		return "unknown"
	}
}

// ParseVoiceModeState parses a voice mode string into VoiceModeState.
func ParseVoiceModeState(v string) (VoiceModeState, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "off", "0", "false":
		return VoiceModeOff, nil
	case "voice_only", "voiceonly", "1":
		return VoiceModeVoiceOnly, nil
	case "all", "2", "true":
		return VoiceModeAll, nil
	default:
		return VoiceModeOff, fmt.Errorf("unknown voice mode %q", v)
	}
}

// VoiceModeEvidence is stable operator-facing evidence for voice mode outcomes.
type VoiceModeEvidence string

const (
	VoiceModeEvidenceOK                  VoiceModeEvidence = "voice_mode_ok"
	VoiceModeEvidenceDisabled            VoiceModeEvidence = "voice_mode_disabled"
	VoiceModeEvidenceInvalidArguments    VoiceModeEvidence = "voice_mode_invalid_arguments"
	VoiceModeEvidenceProviderUnavailable VoiceModeEvidence = "voice_mode_provider_unavailable"
	VoiceModeEvidenceAudioNotAvailable   VoiceModeEvidence = "voice_mode_audio_not_available"
	VoiceModeEvidenceRecordingFailed     VoiceModeEvidence = "voice_mode_recording_failed"
	VoiceModeEvidenceTranscriptionFailed VoiceModeEvidence = "voice_mode_transcription_failed"
	VoiceModeEvidenceSynthesizeFailed    VoiceModeEvidence = "voice_mode_synthesize_failed"
	VoiceModeEvidencePlaybackFailed      VoiceModeEvidence = "voice_mode_playback_failed"
	VoiceModeEvidenceChatNotFound        VoiceModeEvidence = "voice_mode_chat_not_found"
	VoiceModeEvidenceStoreError          VoiceModeEvidence = "voice_mode_store_error"
)

// VoiceModeConfig controls global voice mode defaults.
type VoiceModeConfig struct {
	Disabled         bool
	SilenceThreshold int     // RMS threshold for silence detection (default 200)
	SilenceDuration  float64 // Seconds of silence before auto-stop (default 3.0)
	MaxRecordingAge  float64 // Max seconds to wait for speech before auto-stop (default 15.0)
	SampleRate       int     // Audio sample rate (default 16000)
}

// DefaultVoiceModeConfig returns a config with sensible defaults.
func DefaultVoiceModeConfig() VoiceModeConfig {
	return VoiceModeConfig{
		SilenceThreshold: 200,
		SilenceDuration:  3.0,
		MaxRecordingAge:  15.0,
		SampleRate:       16000,
	}
}

// VoiceModeResult is the redacted tool result envelope.
type VoiceModeResult struct {
	Success    bool              `json:"success"`
	Mode       string            `json:"mode,omitempty"`
	Transcript string            `json:"transcript,omitempty"`
	FilePath   string            `json:"file_path,omitempty"`
	MediaTag   string            `json:"media_tag,omitempty"`
	Evidence   VoiceModeEvidence `json:"evidence"`
	Error      string            `json:"error,omitempty"`
}

func FormatVoiceModeStatus(mode string, recordKey any) string {
	binding := ResolveVoiceRecordKey(recordKey, VoiceRecordKeyOptions{})
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = VoiceModeOff.String()
	}
	return fmt.Sprintf("voice: %s | record key: %s | evidence: %s", mode, binding.Display, binding.Evidence)
}

// VoiceModeChatState is the per-chat voice mode state.
type VoiceModeChatState struct {
	ChatID    string         `json:"chat_id"`
	Mode      VoiceModeState `json:"mode"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// VoiceModeStore persists per-chat voice mode state.
// Implementations must be safe for concurrent access.
type VoiceModeStore interface {
	// Get returns the voice mode state for a chat, or VoiceModeOff if unset.
	Get(ctx context.Context, chatID string) (VoiceModeChatState, error)
	// Set updates the voice mode state for a chat.
	Set(ctx context.Context, state VoiceModeChatState) error
	// List returns all known chat states.
	List(ctx context.Context) ([]VoiceModeChatState, error)
}

// InMemoryVoiceModeStore is a thread-safe in-memory store for testing.
type InMemoryVoiceModeStore struct {
	states map[string]VoiceModeChatState
}

func NewInMemoryVoiceModeStore() *InMemoryVoiceModeStore {
	return &InMemoryVoiceModeStore{states: make(map[string]VoiceModeChatState)}
}

func (s *InMemoryVoiceModeStore) Get(_ context.Context, chatID string) (VoiceModeChatState, error) {
	state, ok := s.states[chatID]
	if !ok {
		return VoiceModeChatState{ChatID: chatID, Mode: VoiceModeOff}, nil
	}
	return state, nil
}

func (s *InMemoryVoiceModeStore) Set(_ context.Context, state VoiceModeChatState) error {
	if state.ChatID == "" {
		return errors.New("chat_id is required")
	}
	s.states[state.ChatID] = state
	return nil
}

func (s *InMemoryVoiceModeStore) List(_ context.Context) ([]VoiceModeChatState, error) {
	result := make([]VoiceModeChatState, 0, len(s.states))
	for _, state := range s.states {
		result = append(result, state)
	}
	return result, nil
}

// FileVoiceModeStore persists state to a JSON file.
type FileVoiceModeStore struct {
	path string
}

func NewFileVoiceModeStore(path string) *FileVoiceModeStore {
	return &FileVoiceModeStore{path: path}
}

func (s *FileVoiceModeStore) Get(_ context.Context, chatID string) (VoiceModeChatState, error) {
	states, err := s.readAll()
	if err != nil {
		if os.IsNotExist(err) {
			return VoiceModeChatState{ChatID: chatID, Mode: VoiceModeOff}, nil
		}
		return VoiceModeChatState{ChatID: chatID, Mode: VoiceModeOff}, err
	}
	if state, ok := states[chatID]; ok {
		return state, nil
	}
	return VoiceModeChatState{ChatID: chatID, Mode: VoiceModeOff}, nil
}

func (s *FileVoiceModeStore) Set(_ context.Context, state VoiceModeChatState) error {
	if state.ChatID == "" {
		return errors.New("chat_id is required")
	}
	states, err := s.readAll()
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		states = make(map[string]VoiceModeChatState)
	}
	states[state.ChatID] = state
	return s.writeAll(states)
}

func (s *FileVoiceModeStore) List(_ context.Context) ([]VoiceModeChatState, error) {
	states, err := s.readAll()
	if err != nil {
		if os.IsNotExist(err) {
			return []VoiceModeChatState{}, nil
		}
		return nil, err
	}
	result := make([]VoiceModeChatState, 0, len(states))
	for _, state := range states {
		result = append(result, state)
	}
	return result, nil
}

func (s *FileVoiceModeStore) readAll() (map[string]VoiceModeChatState, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	var states map[string]VoiceModeChatState
	if err := json.Unmarshal(data, &states); err != nil {
		return nil, err
	}
	return states, nil
}

func (s *FileVoiceModeStore) writeAll(states map[string]VoiceModeChatState) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

// VoiceModeProvider captures the audio I/O seam. Production implementations
// use hardware (sounddevice/PortAudio); tests use fakes.
type VoiceModeProvider interface {
	// Available reports whether audio I/O is available in this environment.
	Available(ctx context.Context) bool
	// StartRecording begins audio capture. Returns a recording handle.
	StartRecording(ctx context.Context, cfg VoiceModeConfig, onSilenceStop func()) (RecordingHandle, error)
	// StopRecording ends capture and returns the recorded audio path.
	StopRecording(ctx context.Context, handle RecordingHandle) (string, error)
	// CancelRecording discards the current recording.
	CancelRecording(ctx context.Context, handle RecordingHandle)
	// PlayAudio plays an audio file through the default output device.
	PlayAudio(ctx context.Context, filePath string) error
	// StopPlayback interrupts any ongoing playback.
	StopPlayback(ctx context.Context)
	// DetectEnvironment returns audio environment diagnostics.
	DetectEnvironment(ctx context.Context) VoiceModeEnvironment
}

// RecordingHandle is an opaque handle returned by StartRecording.
type RecordingHandle interface{}

// VoiceModeRunner orchestrates voice mode operations.
type VoiceModeRunner struct {
	cfg           VoiceModeConfig
	store         VoiceModeStore
	audioProvider VoiceModeProvider
	ttsRunner     *TTSRunner
	sttRunner     *TranscriptionRunner
}

// NewVoiceModeRunner creates a runner with injected dependencies.
func NewVoiceModeRunner(
	cfg VoiceModeConfig,
	store VoiceModeStore,
	audioProvider VoiceModeProvider,
	ttsRunner *TTSRunner,
	sttRunner *TranscriptionRunner,
) *VoiceModeRunner {
	return &VoiceModeRunner{
		cfg:           cfg,
		store:         store,
		audioProvider: audioProvider,
		ttsRunner:     ttsRunner,
		sttRunner:     sttRunner,
	}
}

// GetMode returns the current voice mode for a chat.
func (r *VoiceModeRunner) GetMode(ctx context.Context, chatID string) (VoiceModeResult, error) {
	if r == nil || r.store == nil {
		return voiceModeFailure("", VoiceModeEvidenceStoreError, "no voice mode store configured"), nil
	}
	state, err := r.store.Get(ctx, chatID)
	if err != nil {
		return voiceModeFailure("", VoiceModeEvidenceStoreError, err.Error()), err
	}
	return VoiceModeResult{
		Success:  true,
		Mode:     state.Mode.String(),
		Evidence: VoiceModeEvidenceOK,
	}, nil
}

// SetMode updates the voice mode for a chat.
func (r *VoiceModeRunner) SetMode(ctx context.Context, chatID string, modeStr string) (VoiceModeResult, error) {
	if r == nil {
		return voiceModeFailure("", VoiceModeEvidenceProviderUnavailable, "no voice mode runner configured"), nil
	}
	cfg := r.cfg
	if cfg.Disabled {
		return voiceModeFailure(modeStr, VoiceModeEvidenceDisabled, "voice mode is disabled"), nil
	}
	if chatID == "" {
		return voiceModeFailure(modeStr, VoiceModeEvidenceInvalidArguments, "chat_id is required"), nil
	}
	mode, err := ParseVoiceModeState(modeStr)
	if err != nil {
		return voiceModeFailure(modeStr, VoiceModeEvidenceInvalidArguments, err.Error()), nil
	}
	state := VoiceModeChatState{
		ChatID:    chatID,
		Mode:      mode,
		UpdatedAt: time.Now().UTC(),
	}
	if err := r.store.Set(ctx, state); err != nil {
		return voiceModeFailure(modeStr, VoiceModeEvidenceStoreError, err.Error()), err
	}
	return VoiceModeResult{
		Success:  true,
		Mode:     mode.String(),
		Evidence: VoiceModeEvidenceOK,
	}, nil
}

// RecordAndTranscribe starts recording, waits for silence, and returns the transcript.
func (r *VoiceModeRunner) RecordAndTranscribe(ctx context.Context, chatID string) (VoiceModeResult, error) {
	if r == nil {
		return voiceModeFailure("", VoiceModeEvidenceProviderUnavailable, "no voice mode runner configured"), nil
	}
	cfg := r.cfg
	if cfg.Disabled {
		return voiceModeFailure("", VoiceModeEvidenceDisabled, "voice mode is disabled"), nil
	}
	// Check current chat mode
	state, err := r.store.Get(ctx, chatID)
	if err != nil {
		return voiceModeFailure("", VoiceModeEvidenceStoreError, err.Error()), err
	}
	if state.Mode == VoiceModeOff {
		return voiceModeFailure("", VoiceModeEvidenceDisabled, "voice mode is off for this chat"), nil
	}
	if r.audioProvider == nil {
		return voiceModeFailure("", VoiceModeEvidenceProviderUnavailable, "no audio provider configured"), nil
	}
	if !r.audioProvider.Available(ctx) {
		return voiceModeFailure("", VoiceModeEvidenceAudioNotAvailable, "audio not available in this environment"), nil
	}
	if r.sttRunner == nil {
		return voiceModeFailure("", VoiceModeEvidenceProviderUnavailable, "no STT runner configured"), nil
	}

	handle, err := r.audioProvider.StartRecording(ctx, cfg, nil)
	if err != nil {
		return voiceModeFailure("", VoiceModeEvidenceRecordingFailed, err.Error()), err
	}

	audioPath, err := r.audioProvider.StopRecording(ctx, handle)
	if err != nil {
		return voiceModeFailure("", VoiceModeEvidenceRecordingFailed, err.Error()), err
	}
	if audioPath == "" {
		return voiceModeFailure("", VoiceModeEvidenceRecordingFailed, "no audio recorded"), nil
	}

	// Transcribe via STT runner
	sttResult := r.sttRunner.Transcribe(ctx, TranscriptionRequest{
		AudioPath: audioPath,
	})
	if !sttResult.Success {
		return VoiceModeResult{
			Success:  false,
			Evidence: VoiceModeEvidenceTranscriptionFailed,
			Error:    sttResult.Error,
		}, nil
	}
	return VoiceModeResult{
		Success:    true,
		Transcript: sttResult.Transcript,
		Evidence:   VoiceModeEvidenceOK,
	}, nil
}

// PlayText synthesizes text to speech and plays it.
func (r *VoiceModeRunner) PlayText(ctx context.Context, chatID string, text string, platform string) (VoiceModeResult, error) {
	if r == nil {
		return voiceModeFailure("", VoiceModeEvidenceProviderUnavailable, "no voice mode runner configured"), nil
	}
	cfg := r.cfg
	if cfg.Disabled {
		return voiceModeFailure("", VoiceModeEvidenceDisabled, "voice mode is disabled"), nil
	}
	state, err := r.store.Get(ctx, chatID)
	if err != nil {
		return voiceModeFailure("", VoiceModeEvidenceStoreError, err.Error()), err
	}
	if state.Mode != VoiceModeAll {
		// voice_only mode does not speak responses
		return voiceModeFailure("", VoiceModeEvidenceDisabled, "voice mode is not in all mode"), nil
	}
	if r.audioProvider == nil {
		return voiceModeFailure("", VoiceModeEvidenceProviderUnavailable, "no audio provider configured"), nil
	}
	if !r.audioProvider.Available(ctx) {
		return voiceModeFailure("", VoiceModeEvidenceAudioNotAvailable, "audio not available in this environment"), nil
	}
	if r.ttsRunner == nil {
		return voiceModeFailure("", VoiceModeEvidenceProviderUnavailable, "no TTS runner configured"), nil
	}

	// Synthesize via TTS runner
	ttsResult := r.ttsRunner.Synthesize(ctx, TTSRequest{
		Text:     text,
		Platform: platform,
	})
	if !ttsResult.Success {
		return VoiceModeResult{
			Success:  false,
			Evidence: VoiceModeEvidenceSynthesizeFailed,
			Error:    ttsResult.Error,
		}, nil
	}

	// Play the audio
	if err := r.audioProvider.PlayAudio(ctx, ttsResult.FilePath); err != nil {
		return VoiceModeResult{
			Success:  false,
			FilePath: ttsResult.FilePath,
			MediaTag: ttsResult.MediaTag,
			Evidence: VoiceModeEvidencePlaybackFailed,
			Error:    err.Error(),
		}, nil
	}
	return VoiceModeResult{
		Success:  true,
		FilePath: ttsResult.FilePath,
		MediaTag: ttsResult.MediaTag,
		Evidence: VoiceModeEvidenceOK,
	}, nil
}

// CheckRequirements returns whether voice mode can operate in this environment.
func (r *VoiceModeRunner) CheckRequirements(ctx context.Context) VoiceModeRequirementsResult {
	if r == nil || r.audioProvider == nil {
		return VoiceModeRequirementsResult{Available: false, Details: "no audio provider configured"}
	}
	env := r.audioProvider.DetectEnvironment(ctx)
	sttAvailable := r.sttRunner != nil
	ttsAvailable := r.ttsRunner != nil

	return VoiceModeRequirementsResult{
		Available:      env.Available && sttAvailable && ttsAvailable,
		AudioAvailable: env.Available,
		STTAvailable:   sttAvailable,
		TTSAvailable:   ttsAvailable,
		Warnings:       env.Warnings,
		Notices:        env.Notices,
		Details:        formatVoiceModeDetails(env, sttAvailable, ttsAvailable),
	}
}

// VoiceModeRequirementsResult summarizes voice mode prerequisites.
type VoiceModeRequirementsResult struct {
	Available      bool     `json:"available"`
	AudioAvailable bool     `json:"audio_available"`
	STTAvailable   bool     `json:"stt_available"`
	TTSAvailable   bool     `json:"tts_available"`
	Warnings       []string `json:"warnings,omitempty"`
	Notices        []string `json:"notices,omitempty"`
	Details        string   `json:"details"`
}

func formatVoiceModeDetails(env VoiceModeEnvironment, sttAvailable, ttsAvailable bool) string {
	var parts []string
	if env.Available {
		parts = append(parts, "Audio: OK")
	} else {
		parts = append(parts, fmt.Sprintf("Audio: UNAVAILABLE (%s)", strings.Join(env.Warnings, "; ")))
	}
	if sttAvailable {
		parts = append(parts, "STT: OK")
	} else {
		parts = append(parts, "STT: not configured")
	}
	if ttsAvailable {
		parts = append(parts, "TTS: OK")
	} else {
		parts = append(parts, "TTS: not configured")
	}
	return strings.Join(parts, " | ")
}

func voiceModeFailure(mode string, evidence VoiceModeEvidence, message string) VoiceModeResult {
	return VoiceModeResult{
		Success:  false,
		Mode:     mode,
		Evidence: evidence,
		Error:    message,
	}
}

// VoiceModeTool exposes voice mode through the standard Tool contract.
type VoiceModeTool struct {
	runner *VoiceModeRunner
}

// NewVoiceModeTool creates a voice mode tool.
func NewVoiceModeTool(runner *VoiceModeRunner) *VoiceModeTool {
	return &VoiceModeTool{runner: runner}
}

func (*VoiceModeTool) Name() string { return "voice_mode" }

func (*VoiceModeTool) Description() string {
	return "Manage voice mode for a chat session. Modes: off (disabled), voice_only (push-to-talk, no TTS playback), all (full voice with TTS playback)."
}

func (*VoiceModeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"chat_id":{"type":"string","description":"Chat or session identifier."},"mode":{"type":"string","enum":["off","voice_only","all"],"description":"Voice mode state: off disables voice, voice_only enables push-to-talk without TTS playback, all enables full voice mode with TTS."},"action":{"type":"string","enum":["get","set","record","speak","check"],"description":"Action to perform: get (read current mode), set (change mode), record (record and transcribe audio), speak (synthesize and play text), check (check requirements)."},"text":{"type":"string","description":"Text to synthesize and speak (used with action=speak)."},"platform":{"type":"string","description":"Platform hint for audio format (e.g., telegram)."}},"required":["chat_id","action"]}`)
}

func (*VoiceModeTool) Timeout() time.Duration { return 60 * time.Second }

func (t *VoiceModeTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		ChatID   string `json:"chat_id"`
		Mode     string `json:"mode"`
		Action   string `json:"action"`
		Text     string `json:"text"`
		Platform string `json:"platform"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		result := voiceModeFailure("", VoiceModeEvidenceInvalidArguments, "invalid args: "+err.Error())
		return json.Marshal(result)
	}
	if in.ChatID == "" {
		result := voiceModeFailure(in.Mode, VoiceModeEvidenceInvalidArguments, "chat_id is required")
		return json.Marshal(result)
	}

	var result VoiceModeResult
	switch in.Action {
	case "get":
		result, _ = t.runner.GetMode(ctx, in.ChatID)
	case "set":
		if in.Mode == "" {
			result = voiceModeFailure("", VoiceModeEvidenceInvalidArguments, "mode is required for set action")
		} else {
			result, _ = t.runner.SetMode(ctx, in.ChatID, in.Mode)
		}
	case "record":
		result, _ = t.runner.RecordAndTranscribe(ctx, in.ChatID)
	case "speak":
		if in.Text == "" {
			result = voiceModeFailure("", VoiceModeEvidenceInvalidArguments, "text is required for speak action")
		} else {
			result, _ = t.runner.PlayText(ctx, in.ChatID, in.Text, in.Platform)
		}
	case "check":
		reqResult := t.runner.CheckRequirements(ctx)
		return json.Marshal(reqResult)
	default:
		result = voiceModeFailure(in.Mode, VoiceModeEvidenceInvalidArguments, "unknown action: "+in.Action)
	}
	return json.Marshal(result)
}

// Ensure compile-time tool conformance.
var _ Tool = (*VoiceModeTool)(nil)

// NoopVoiceModeProvider is a voice mode provider that reports unavailable.
type NoopVoiceModeProvider struct{}

func (*NoopVoiceModeProvider) Available(context.Context) bool { return false }
func (*NoopVoiceModeProvider) StartRecording(context.Context, VoiceModeConfig, func()) (RecordingHandle, error) {
	return nil, errors.New("audio not available")
}
func (*NoopVoiceModeProvider) StopRecording(context.Context, RecordingHandle) (string, error) {
	return "", nil
}
func (*NoopVoiceModeProvider) CancelRecording(context.Context, RecordingHandle) {}
func (*NoopVoiceModeProvider) PlayAudio(context.Context, string) error {
	return errors.New("audio not available")
}
func (*NoopVoiceModeProvider) StopPlayback(context.Context) {}
func (*NoopVoiceModeProvider) DetectEnvironment(context.Context) VoiceModeEnvironment {
	return VoiceModeEnvironment{Available: false, Warnings: []string{"audio not available"}}
}
