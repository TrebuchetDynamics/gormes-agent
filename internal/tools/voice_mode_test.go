//go:build !slim

package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVoiceModeStateParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected VoiceModeState
		wantErr  bool
	}{
		{"off", VoiceModeOff, false},
		{"voice_only", VoiceModeVoiceOnly, false},
		{"all", VoiceModeAll, false},
		{"OFF", VoiceModeOff, false},
		{"VOICE_ONLY", VoiceModeVoiceOnly, false},
		{"ALL", VoiceModeAll, false},
		{"0", VoiceModeOff, false},
		{"1", VoiceModeVoiceOnly, false},
		{"2", VoiceModeAll, false},
		{"true", VoiceModeAll, false},
		{"false", VoiceModeOff, false},
		{"invalid", VoiceModeOff, true},
		{"", VoiceModeOff, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseVoiceModeState(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVoiceModeState(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseVoiceModeState(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestVoiceModeStateString(t *testing.T) {
	tests := []struct {
		state  VoiceModeState
		expect string
	}{
		{VoiceModeOff, "off"},
		{VoiceModeVoiceOnly, "voice_only"},
		{VoiceModeAll, "all"},
		{VoiceModeState(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expect {
				t.Errorf("VoiceModeState(%d).String() = %q, want %q", tt.state, got, tt.expect)
			}
		})
	}
}

func TestInMemoryVoiceModeStore(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryVoiceModeStore()

	// Get non-existent chat should return VoiceModeOff
	state, err := store.Get(ctx, "chat1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if state.ChatID != "chat1" || state.Mode != VoiceModeOff {
		t.Fatalf("initial state = %+v, want chat1/off", state)
	}

	// Set mode to voice_only
	err = store.Set(ctx, VoiceModeChatState{ChatID: "chat1", Mode: VoiceModeVoiceOnly, UpdatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Get should return voice_only
	state, err = store.Get(ctx, "chat1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if state.Mode != VoiceModeVoiceOnly {
		t.Fatalf("state = %+v, want voice_only", state)
	}

	// Set mode to all for another chat
	err = store.Set(ctx, VoiceModeChatState{ChatID: "chat2", Mode: VoiceModeAll, UpdatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	// List should return both
	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List length = %d, want 2", len(list))
	}

	// Set with empty chatID should fail
	err = store.Set(ctx, VoiceModeChatState{ChatID: "", Mode: VoiceModeAll})
	if err == nil {
		t.Error("Set with empty chatID should fail")
	}
}

func TestFileVoiceModeStore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "voice_mode_store.json")
	store := NewFileVoiceModeStore(storePath)
	ctx := context.Background()

	// Get non-existent chat - should return default without error
	state, err := store.Get(ctx, "chat1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if state.ChatID != "chat1" || state.Mode != VoiceModeOff {
		t.Fatalf("initial state = %+v, want chat1/off", state)
	}

	// Set and persist
	err = store.Set(ctx, VoiceModeChatState{ChatID: "chat1", Mode: VoiceModeVoiceOnly, UpdatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	// New store instance should see persisted state
	store2 := NewFileVoiceModeStore(storePath)
	state, err = store2.Get(ctx, "chat1")
	if err != nil {
		t.Fatalf("Get from new store: %v", err)
	}
	if state.Mode != VoiceModeVoiceOnly {
		t.Fatalf("persisted state = %+v, want voice_only", state)
	}

	// List all chats
	list, err := store2.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List length = %d, want 1", len(list))
	}
}

func TestVoiceModeRunnerGetSet(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryVoiceModeStore()
	runner := NewVoiceModeRunner(DefaultVoiceModeConfig(), store, nil, nil, nil)

	// Get initial state
	result, err := runner.GetMode(ctx, "chat1")
	if err != nil {
		t.Fatalf("GetMode: %v", err)
	}
	if !result.Success || result.Mode != "off" {
		t.Fatalf("GetMode = %+v, want success/off", result)
	}

	// Set to voice_only
	result, err = runner.SetMode(ctx, "chat1", "voice_only")
	if err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	if !result.Success || result.Mode != "voice_only" {
		t.Fatalf("SetMode = %+v, want success/voice_only", result)
	}

	// Get should reflect change
	result, err = runner.GetMode(ctx, "chat1")
	if err != nil {
		t.Fatalf("GetMode: %v", err)
	}
	if !result.Success || result.Mode != "voice_only" {
		t.Fatalf("GetMode = %+v, want success/voice_only", result)
	}

	// Set to all
	result, err = runner.SetMode(ctx, "chat1", "all")
	if err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	if !result.Success || result.Mode != "all" {
		t.Fatalf("SetMode = %+v, want success/all", result)
	}

	// Invalid mode
	result, err = runner.SetMode(ctx, "chat1", "invalid_mode")
	if err != nil {
		t.Fatalf("SetMode with invalid: %v", err)
	}
	if result.Success || result.Evidence != VoiceModeEvidenceInvalidArguments {
		t.Fatalf("invalid mode result = %+v, want failure/invalid_arguments", result)
	}

	// Empty chatID
	result, err = runner.SetMode(ctx, "", "all")
	if err != nil {
		t.Fatalf("SetMode with empty chatID: %v", err)
	}
	if result.Success || result.Evidence != VoiceModeEvidenceInvalidArguments {
		t.Fatalf("empty chatID result = %+v, want failure/invalid_arguments", result)
	}
}

func TestVoiceModeRunnerDisabled(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryVoiceModeStore()
	cfg := DefaultVoiceModeConfig()
	cfg.Disabled = true
	runner := NewVoiceModeRunner(cfg, store, nil, nil, nil)

	result, err := runner.SetMode(ctx, "chat1", "all")
	if err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	if result.Success || result.Evidence != VoiceModeEvidenceDisabled {
		t.Fatalf("disabled result = %+v, want disabled evidence", result)
	}
}

func TestVoiceModeRunnerRecordNoProvider(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryVoiceModeStore()
	runner := NewVoiceModeRunner(DefaultVoiceModeConfig(), store, nil, nil, nil)

	// Set mode first
	runner.SetMode(ctx, "chat1", "voice_only")

	result, err := runner.RecordAndTranscribe(ctx, "chat1")
	if err != nil {
		t.Fatalf("RecordAndTranscribe: %v", err)
	}
	// Should fail because no audio provider
	if result.Success || result.Evidence != VoiceModeEvidenceProviderUnavailable {
		t.Fatalf("no provider result = %+v, want provider unavailable", result)
	}
}

func TestVoiceModeRunnerPlayNoProvider(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryVoiceModeStore()
	runner := NewVoiceModeRunner(DefaultVoiceModeConfig(), store, nil, nil, nil)

	// Set mode to all
	runner.SetMode(ctx, "chat1", "all")

	result, err := runner.PlayText(ctx, "chat1", "Hello", "telegram")
	if err != nil {
		t.Fatalf("PlayText: %v", err)
	}
	// Should fail because no audio provider
	if result.Success || result.Evidence != VoiceModeEvidenceProviderUnavailable {
		t.Fatalf("no provider result = %+v, want provider unavailable", result)
	}
}

func TestVoiceModeRunnerPlayVoiceOnlyMode(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryVoiceModeStore()
	runner := NewVoiceModeRunner(DefaultVoiceModeConfig(), store, nil, nil, nil)

	// Set mode to voice_only (should not play)
	runner.SetMode(ctx, "chat1", "voice_only")

	result, err := runner.PlayText(ctx, "chat1", "Hello", "telegram")
	if err != nil {
		t.Fatalf("PlayText: %v", err)
	}
	if result.Success || result.Evidence != VoiceModeEvidenceDisabled {
		t.Fatalf("voice_only play result = %+v, want disabled", result)
	}
}

func TestVoiceModeRunnerCheckRequirements(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryVoiceModeStore()
	runner := NewVoiceModeRunner(DefaultVoiceModeConfig(), store, nil, nil, nil)

	result := runner.CheckRequirements(ctx)
	if result.Available {
		t.Fatal("should not be available with nil providers")
	}
	if result.AudioAvailable {
		t.Fatal("audio should not be available with nil provider")
	}
}

func TestVoiceModeToolDescriptor(t *testing.T) {
	runner := NewVoiceModeRunner(DefaultVoiceModeConfig(), NewInMemoryVoiceModeStore(), nil, nil, nil)
	tool := NewVoiceModeTool(runner)

	if tool.Name() != "voice_mode" {
		t.Fatalf("tool name = %q, want voice_mode", tool.Name())
	}

	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("schema invalid JSON: %v", err)
	}
	for _, field := range []string{"chat_id", "action"} {
		if _, ok := schema.Properties[field]; !ok {
			t.Fatalf("schema missing %q", field)
		}
	}
	if len(schema.Required) != 2 || schema.Required[0] != "chat_id" || schema.Required[1] != "action" {
		t.Fatalf("required = %#v, want [chat_id action]", schema.Required)
	}
}

func TestVoiceModeToolExecute(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryVoiceModeStore()
	runner := NewVoiceModeRunner(DefaultVoiceModeConfig(), store, nil, nil, nil)
	tool := NewVoiceModeTool(runner)

	// Test get action
	raw, err := tool.Execute(ctx, json.RawMessage(`{"chat_id":"chat1","action":"get"}`))
	if err != nil {
		t.Fatalf("Execute get: %v", err)
	}
	var result VoiceModeResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("result JSON: %v", err)
	}
	if !result.Success || result.Mode != "off" {
		t.Fatalf("get result = %+v, want success/off", result)
	}

	// Test set action
	raw, err = tool.Execute(ctx, json.RawMessage(`{"chat_id":"chat1","action":"set","mode":"voice_only"}`))
	if err != nil {
		t.Fatalf("Execute set: %v", err)
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("result JSON: %v", err)
	}
	if !result.Success || result.Mode != "voice_only" {
		t.Fatalf("set result = %+v, want success/voice_only", result)
	}

	// Test check action
	raw, err = tool.Execute(ctx, json.RawMessage(`{"chat_id":"chat1","action":"check"}`))
	if err != nil {
		t.Fatalf("Execute check: %v", err)
	}
	var reqResult VoiceModeRequirementsResult
	if err := json.Unmarshal(raw, &reqResult); err != nil {
		t.Fatalf("check result JSON: %v", err)
	}
	if reqResult.Available {
		t.Fatal("check should report unavailable with no audio provider")
	}

	// Test invalid action
	raw, err = tool.Execute(ctx, json.RawMessage(`{"chat_id":"chat1","action":"invalid"}`))
	if err != nil {
		t.Fatalf("Execute invalid: %v", err)
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("result JSON: %v", err)
	}
	if result.Success || result.Evidence != VoiceModeEvidenceInvalidArguments {
		t.Fatalf("invalid action result = %+v, want failure", result)
	}

	// Test missing chat_id
	raw, err = tool.Execute(ctx, json.RawMessage(`{"action":"get"}`))
	if err != nil {
		t.Fatalf("Execute no chat_id: %v", err)
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("result JSON: %v", err)
	}
	if result.Success || result.Evidence != VoiceModeEvidenceInvalidArguments {
		t.Fatalf("no chat_id result = %+v, want failure", result)
	}
}

func TestVoiceModeToolModeSwitchSequence(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryVoiceModeStore()
	runner := NewVoiceModeRunner(DefaultVoiceModeConfig(), store, nil, nil, nil)
	tool := NewVoiceModeTool(runner)

	// Sequence: off -> voice_only -> all -> off
	modes := []string{"voice_only", "all", "off"}
	for _, mode := range modes {
		modeJSON, _ := json.Marshal(mode)
		raw, err := tool.Execute(ctx, json.RawMessage(`{"chat_id":"chat1","action":"set","mode":`+string(modeJSON)+`}`))
		if err != nil {
			t.Fatalf("SetMode %s: %v", mode, err)
		}
		var result VoiceModeResult
		if err := json.Unmarshal(raw, &result); err != nil {
			t.Fatalf("result JSON: %v", err)
		}
		if !result.Success || result.Mode != mode {
			t.Fatalf("set %s result = %+v, want success/%s", mode, result, mode)
		}
	}
}

func TestVoiceModeStorePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "persistence_test.json")
	ctx := context.Background()

	// Create store and set some state
	store1 := NewFileVoiceModeStore(storePath)
	err := store1.Set(ctx, VoiceModeChatState{ChatID: "chat1", Mode: VoiceModeVoiceOnly, UpdatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	err = store1.Set(ctx, VoiceModeChatState{ChatID: "chat2", Mode: VoiceModeAll, UpdatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Simulate restart: new store instance
	store2 := NewFileVoiceModeStore(storePath)
	list, err := store2.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("persisted list length = %d, want 2", len(list))
	}

	// Verify individual states
	state1, _ := store2.Get(ctx, "chat1")
	if state1.Mode != VoiceModeVoiceOnly {
		t.Fatalf("chat1 persisted mode = %v, want voice_only", state1.Mode)
	}
	state2, _ := store2.Get(ctx, "chat2")
	if state2.Mode != VoiceModeAll {
		t.Fatalf("chat2 persisted mode = %v, want all", state2.Mode)
	}

	// Update chat1
	err = store2.Set(ctx, VoiceModeChatState{ChatID: "chat1", Mode: VoiceModeAll, UpdatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Verify update persisted
	store3 := NewFileVoiceModeStore(storePath)
	state3, _ := store3.Get(ctx, "chat1")
	if state3.Mode != VoiceModeAll {
		t.Fatalf("updated chat1 persisted mode = %v, want all", state3.Mode)
	}
}

func TestNoopVoiceModeProvider(t *testing.T) {
	provider := &NoopVoiceModeProvider{}
	ctx := context.Background()

	if provider.Available(ctx) {
		t.Error("NoopVoiceModeProvider should not be available")
	}

	_, err := provider.StartRecording(ctx, DefaultVoiceModeConfig(), nil)
	if err == nil {
		t.Error("StartRecording should fail")
	}

	err = provider.PlayAudio(ctx, "/tmp/test.wav")
	if err == nil {
		t.Error("PlayAudio should fail")
	}

	env := provider.DetectEnvironment(ctx)
	if env.Available {
		t.Error("DetectEnvironment should report unavailable")
	}
}

func TestVoiceModeRunnerWithFakeProviders(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryVoiceModeStore()
	tmpDir := t.TempDir()

	// Create fake TTS provider - use "edge" name since that's in the auto-select list
	ttsOutput := filepath.Join(tmpDir, "tts_output.ogg")
	fakeTTS := &fakeVoiceModeTTSProvider{available: true, outputPath: ttsOutput}
	ttsRunner := NewTTSRunner(TTSConfig{Provider: "edge"}, map[string]TTSProvider{
		"edge": fakeTTS,
	})

	// Create fake STT provider - use "groq" name since that's in the auto-select list
	sttRunner := NewTranscriptionRunner(TranscriptionConfig{}, map[string]TranscriptionProvider{
		"groq": &fakeVoiceModeSTTProvider{available: true, transcript: "hello world"},
	})

	// Create fake audio provider
	fakeAudio := &fakeVoiceModeAudioProvider{available: true}

	runner := NewVoiceModeRunner(DefaultVoiceModeConfig(), store, fakeAudio, ttsRunner, sttRunner)

	// Set mode to all
	runner.SetMode(ctx, "chat1", "all")

	// Play text
	result, err := runner.PlayText(ctx, "chat1", "Hello world", "telegram")
	if err != nil {
		t.Fatalf("PlayText: %v", err)
	}
	if !result.Success || result.Evidence != VoiceModeEvidenceOK {
		t.Fatalf("PlayText result = %+v, want success", result)
	}
	if !fakeTTS.synthesizeCalled {
		t.Error("TTS Synthesize was not called")
	}
	if !fakeAudio.playCalled {
		t.Error("Audio PlayAudio was not called")
	}

	// Check requirements
	reqResult := runner.CheckRequirements(ctx)
	if !reqResult.Available {
		t.Error("CheckRequirements should report available with all providers")
	}
}

// fakeVoiceModeTTSProvider is a fake TTS provider for testing.
type fakeVoiceModeTTSProvider struct {
	available        bool
	outputPath       string
	synthesizeCalled bool
}

func (f *fakeVoiceModeTTSProvider) Available(context.Context) bool { return f.available }

func (f *fakeVoiceModeTTSProvider) Synthesize(_ context.Context, req TTSProviderRequest) (TTSProviderResult, error) {
	f.synthesizeCalled = true
	if err := os.MkdirAll(filepath.Dir(req.OutputPath), 0o700); err != nil {
		return TTSProviderResult{}, err
	}
	if err := os.WriteFile(req.OutputPath, []byte("fake audio"), 0o600); err != nil {
		return TTSProviderResult{}, err
	}
	return TTSProviderResult{
		FilePath:        req.OutputPath,
		Provider:        "edge",
		VoiceCompatible: true,
	}, nil
}

// fakeVoiceModeSTTProvider is a fake STT provider for testing.
type fakeVoiceModeSTTProvider struct {
	available        bool
	transcript       string
	transcribeCalled bool
}

func (f *fakeVoiceModeSTTProvider) Available(context.Context) bool { return f.available }

func (f *fakeVoiceModeSTTProvider) Transcribe(_ context.Context, req TranscriptionProviderRequest) (TranscriptionProviderResult, error) {
	f.transcribeCalled = true
	return TranscriptionProviderResult{
		Transcript: f.transcript,
		Provider:   "groq",
	}, nil
}

// fakeVoiceModeAudioProvider is a fake audio provider for testing.
type fakeVoiceModeAudioProvider struct {
	available    bool
	playCalled   bool
	stopCalled   bool
	recordStarted bool
}

func (f *fakeVoiceModeAudioProvider) Available(context.Context) bool { return f.available }

func (f *fakeVoiceModeAudioProvider) StartRecording(context.Context, VoiceModeConfig, func()) (RecordingHandle, error) {
	f.recordStarted = true
	return &fakeRecordingHandle{}, nil
}

func (f *fakeVoiceModeAudioProvider) StopRecording(context.Context, RecordingHandle) (string, error) {
	return "", nil
}

func (f *fakeVoiceModeAudioProvider) CancelRecording(context.Context, RecordingHandle) {}

func (f *fakeVoiceModeAudioProvider) PlayAudio(context.Context, string) error {
	f.playCalled = true
	return nil
}

func (f *fakeVoiceModeAudioProvider) StopPlayback(context.Context) {
	f.stopCalled = true
}

func (f *fakeVoiceModeAudioProvider) DetectEnvironment(context.Context) VoiceModeEnvironment {
	return VoiceModeEnvironment{Available: f.available}
}

type fakeRecordingHandle struct{}
