//go:build !slim

package main

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// fakeTranscriptionProvider implements tools.TranscriptionProvider for tests
// without touching live HTTP endpoints. It records the AudioPath the
// adapter wrote so we can assert the bytes survived the trip through the
// tempfile bridge.
type fakeTranscriptionProvider struct {
	available     bool
	availableErr  error
	transcript    string
	err           error
	receivedReq   tools.TranscriptionProviderRequest
	receivedBytes []byte
}

func (f *fakeTranscriptionProvider) Available(ctx context.Context) bool {
	_ = ctx
	return f.available
}

func (f *fakeTranscriptionProvider) Transcribe(ctx context.Context, req tools.TranscriptionProviderRequest) (tools.TranscriptionProviderResult, error) {
	_ = ctx
	f.receivedReq = req
	if req.AudioPath != "" {
		if data, readErr := os.ReadFile(req.AudioPath); readErr == nil {
			f.receivedBytes = data
		}
	}
	if f.err != nil {
		return tools.TranscriptionProviderResult{}, f.err
	}
	return tools.TranscriptionProviderResult{Transcript: f.transcript}, nil
}

func TestHTTPSTTAudioTranscriber_BridgesBytesToProviderAndReturnsTranscript(t *testing.T) {
	provider := &fakeTranscriptionProvider{available: true, transcript: "hello world"}
	adapter := httpSTTAudioTranscriber{provider: provider}

	wantBytes := []byte("OggS\x00fake-opus-payload")
	got, err := adapter.Transcribe(context.Background(), telegram.AudioInput{
		Kind:      "voice",
		MediaType: "audio/ogg",
		FileName:  "voice-message.oga",
		Data:      wantBytes,
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if got != "hello world" {
		t.Fatalf("expected transcript %q, got %q", "hello world", got)
	}
	if string(provider.receivedBytes) != string(wantBytes) {
		t.Fatalf("provider received wrong bytes: want %q got %q", wantBytes, provider.receivedBytes)
	}
	if !strings.HasSuffix(provider.receivedReq.AudioPath, ".oga") {
		t.Fatalf("expected tempfile to keep .oga extension from FileName, got path %q", provider.receivedReq.AudioPath)
	}
	// Tempfile must be cleaned up after Transcribe returns.
	if _, statErr := os.Stat(provider.receivedReq.AudioPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected tempfile to be removed, stat err: %v", statErr)
	}
}

func TestHTTPSTTAudioTranscriber_ReturnsErrorWhenProviderUnavailable(t *testing.T) {
	provider := &fakeTranscriptionProvider{available: false}
	adapter := httpSTTAudioTranscriber{provider: provider}

	_, err := adapter.Transcribe(context.Background(), telegram.AudioInput{Data: []byte("anything")})
	if err == nil {
		t.Fatal("expected error when provider is unavailable, got nil")
	}
}

func TestHTTPSTTAudioTranscriber_ReturnsErrorWhenProviderTranscribeFails(t *testing.T) {
	wantErr := errors.New("provider boom")
	provider := &fakeTranscriptionProvider{available: true, err: wantErr}
	adapter := httpSTTAudioTranscriber{provider: provider}

	_, err := adapter.Transcribe(context.Background(), telegram.AudioInput{
		FileName: "x.ogg",
		Data:     []byte("payload"),
	})
	if err == nil {
		t.Fatal("expected propagated error, got nil")
	}
	if !errors.Is(err, wantErr) && !strings.Contains(err.Error(), "provider boom") {
		t.Fatalf("expected upstream error to surface, got %v", err)
	}
}

func TestHTTPSTTAudioTranscriber_FallsBackToOggExtensionWhenFileNameMissing(t *testing.T) {
	provider := &fakeTranscriptionProvider{available: true, transcript: "ok"}
	adapter := httpSTTAudioTranscriber{provider: provider}

	_, err := adapter.Transcribe(context.Background(), telegram.AudioInput{Data: []byte("x")})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if !strings.HasSuffix(provider.receivedReq.AudioPath, ".ogg") {
		t.Fatalf("expected default .ogg extension when FileName is empty, got %q", provider.receivedReq.AudioPath)
	}
}

func TestNewHTTPAudioTranscriberFromEnv_ReturnsNilWhenNoSTTKeySet(t *testing.T) {
	t.Setenv("GORMES_STT_OPENAI_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("VOICE_TOOLS_OPENAI_KEY", "")
	t.Setenv("GROQ_API_KEY", "")

	got := newHTTPAudioTranscriberFromEnv()
	if got != nil {
		t.Fatalf("expected nil when no STT key is set, got %T", got)
	}
}

func TestNewHTTPAudioTranscriberFromEnv_ReturnsOpenAIAdapterWhenKeySet(t *testing.T) {
	t.Setenv("GORMES_STT_OPENAI_KEY", "")
	t.Setenv("VOICE_TOOLS_OPENAI_KEY", "")
	t.Setenv("OPENAI_API_KEY", "sk-test-not-real")
	t.Setenv("GROQ_API_KEY", "")

	got := newHTTPAudioTranscriberFromEnv()
	if got == nil {
		t.Fatal("expected non-nil HTTP transcriber when OPENAI_API_KEY is set, got nil")
	}
	if _, ok := got.(httpSTTAudioTranscriber); !ok {
		t.Fatalf("expected httpSTTAudioTranscriber adapter, got %T", got)
	}
}

func TestNewHTTPAudioTranscriberFromEnv_PrefersGroqWhenBothKeysSet(t *testing.T) {
	// Groq is free-tier; prefer it over paid OpenAI when both are configured.
	t.Setenv("GORMES_STT_OPENAI_KEY", "")
	t.Setenv("VOICE_TOOLS_OPENAI_KEY", "")
	t.Setenv("OPENAI_API_KEY", "sk-test-not-real")
	t.Setenv("GROQ_API_KEY", "gsk-test-not-real")

	got := newHTTPAudioTranscriberFromEnv()
	if got == nil {
		t.Fatal("expected non-nil HTTP transcriber when both keys are set, got nil")
	}
	adapter, ok := got.(httpSTTAudioTranscriber)
	if !ok {
		t.Fatalf("expected httpSTTAudioTranscriber, got %T", got)
	}
	if _, isGroq := adapter.provider.(*tools.TranscriptionGroqProvider); !isGroq {
		t.Fatalf("expected resolver to bind Groq provider when GROQ_API_KEY was set; got %T", adapter.provider)
	}
}

func TestNewHTTPAudioTranscriberFromEnv_ReturnsGroqAdapterWhenOnlyGroqKeySet(t *testing.T) {
	t.Setenv("GORMES_STT_OPENAI_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("VOICE_TOOLS_OPENAI_KEY", "")
	t.Setenv("GROQ_API_KEY", "gsk-test-not-real")

	got := newHTTPAudioTranscriberFromEnv()
	if got == nil {
		t.Fatal("expected non-nil HTTP transcriber when GROQ_API_KEY is set, got nil")
	}
	if _, ok := got.(httpSTTAudioTranscriber); !ok {
		t.Fatalf("expected httpSTTAudioTranscriber adapter, got %T", got)
	}
}

func TestResolveTelegramAudioTranscriber_FallsBackToHTTPWhenNoLocalCLI(t *testing.T) {
	// Force NewWhisperTranscriberFromEnv to return nil by clearing the env
	// override and ensuring no whisper binary is on PATH for this test.
	t.Setenv("GORMES_WHISPER_COMMAND", "")
	t.Setenv("GORMES_WASI_WHISPER_MODEL", "")
	t.Setenv("PATH", "/nonexistent")
	// Configure HTTP fallback.
	t.Setenv("GORMES_STT_OPENAI_KEY", "")
	t.Setenv("VOICE_TOOLS_OPENAI_KEY", "")
	t.Setenv("OPENAI_API_KEY", "sk-test-not-real")

	got := resolveTelegramAudioTranscriber()
	if got == nil {
		t.Fatal("expected resolver to return HTTP fallback when no local whisper is available, got nil")
	}
	if _, ok := got.(httpSTTAudioTranscriber); !ok {
		t.Fatalf("expected httpSTTAudioTranscriber, got %T", got)
	}
}

func TestResolveTelegramAudioTranscriber_ReturnsNilWhenNeitherLocalNorHTTPConfigured(t *testing.T) {
	t.Setenv("GORMES_WHISPER_COMMAND", "")
	t.Setenv("GORMES_WASI_WHISPER_MODEL", "")
	t.Setenv("PATH", "/nonexistent")
	t.Setenv("GORMES_STT_OPENAI_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("VOICE_TOOLS_OPENAI_KEY", "")
	t.Setenv("GROQ_API_KEY", "")

	got := resolveTelegramAudioTranscriber()
	if got != nil {
		t.Fatalf("expected nil when neither local nor HTTP is configured, got %T", got)
	}
}

func TestWASIWhisperFromEnv_ReturnsNilWhenModelUnsetOrMissing(t *testing.T) {
	t.Setenv("GORMES_WASI_WHISPER_MODEL", "")
	if got := newWASIWhisperTranscriberFromEnv(); got != nil {
		t.Fatalf("expected nil when env var is unset, got %T", got)
	}

	t.Setenv("GORMES_WASI_WHISPER_MODEL", filepath.Join(t.TempDir(), "missing.bin"))
	if got := newWASIWhisperTranscriberFromEnv(); got != nil {
		t.Fatalf("expected nil when model is missing, got %T", got)
	}
}

func TestWASIWhisperFromEnv_ReturnsAdapterWhenModelFileExists(t *testing.T) {
	modelPath := filepath.Join(t.TempDir(), "ggml-tiny.en.bin")
	if err := os.WriteFile(modelPath, []byte("placeholder-model"), 0o600); err != nil {
		t.Fatalf("write model fixture: %v", err)
	}
	t.Setenv("GORMES_WASI_WHISPER_MODEL", modelPath)

	got := newWASIWhisperTranscriberFromEnv()
	adapter, ok := got.(*wasiWhisperAudioTranscriber)
	if !ok {
		t.Fatalf("expected *wasiWhisperAudioTranscriber, got %T", got)
	}
	if adapter.modelPath != modelPath {
		t.Fatalf("modelPath = %q, want %q", adapter.modelPath, modelPath)
	}
}

func TestWASIWhisperAudioTranscriber_WritesWAVTempfileAndReturnsTranscript(t *testing.T) {
	modelPath := filepath.Join(t.TempDir(), "ggml-tiny.en.bin")
	var capturedModel string
	var capturedPath string
	var capturedBytes []byte
	input := testTelegramWAVPCM16Mono16k(t, []int16{0, 1024, -1024})
	adapter := &wasiWhisperAudioTranscriber{
		modelPath: modelPath,
		newTranscriber: func(_ context.Context, gotModel string) (wasiWhisperTranscriber, error) {
			capturedModel = gotModel
			return fakeWASIWhisperTranscriber{
				transcript: "hello from wasi",
				onTranscribe: func(path string) {
					capturedPath = path
					capturedBytes, _ = os.ReadFile(path)
				},
			}, nil
		},
	}

	got, err := adapter.Transcribe(context.Background(), telegram.AudioInput{
		MediaType: "audio/wav",
		FileName:  "voice.wav",
		Data:      input,
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if got != "hello from wasi" {
		t.Fatalf("transcript = %q, want %q", got, "hello from wasi")
	}
	if capturedModel != modelPath {
		t.Fatalf("model path = %q, want %q", capturedModel, modelPath)
	}
	if !strings.HasSuffix(capturedPath, ".wav") {
		t.Fatalf("expected WAV tempfile, got %q", capturedPath)
	}
	if string(capturedBytes) != string(input) {
		t.Fatalf("transcriber received bytes %q", capturedBytes)
	}
	if _, statErr := os.Stat(capturedPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected tempfile cleanup, stat err: %v", statErr)
	}
}

func TestWASIWhisperAudioTranscriber_UsesInjectedConverterBeforeTranscribing(t *testing.T) {
	var convertedFrom string
	var transcribedPath string
	adapter := &wasiWhisperAudioTranscriber{
		modelPath: filepath.Join(t.TempDir(), "ggml-tiny.en.bin"),
		convertToWAV: func(_ context.Context, inputPath, outputPath string) error {
			convertedFrom = inputPath
			return os.WriteFile(outputPath, testTelegramWAVPCM16Mono16k(t, []int16{7, 8, 9}), 0o600)
		},
		newTranscriber: func(_ context.Context, _ string) (wasiWhisperTranscriber, error) {
			return fakeWASIWhisperTranscriber{
				transcript: "converted transcript",
				onTranscribe: func(path string) {
					transcribedPath = path
				},
			}, nil
		},
	}

	got, err := adapter.Transcribe(context.Background(), telegram.AudioInput{
		MediaType: "audio/ogg",
		FileName:  "voice.ogg",
		Data:      []byte("OggS"),
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if got != "converted transcript" {
		t.Fatalf("transcript = %q", got)
	}
	if !strings.HasSuffix(convertedFrom, ".ogg") {
		t.Fatalf("converted input = %q, want .ogg", convertedFrom)
	}
	if !strings.HasSuffix(transcribedPath, ".wav") {
		t.Fatalf("transcribedPath = %q, want .wav", transcribedPath)
	}
}

func TestWASIWhisperAudioTranscriber_ChunksLongPCMAndJoinsTranscripts(t *testing.T) {
	samples := make([]int16, 31*16000)
	for i := range samples {
		samples[i] = int16(i % 1024)
	}
	input := testTelegramWAVPCM16Mono16k(t, samples)
	var calls int
	var chunkPaths []string
	adapter := &wasiWhisperAudioTranscriber{
		modelPath: filepath.Join(t.TempDir(), "ggml-tiny.en.bin"),
		newTranscriber: func(context.Context, string) (wasiWhisperTranscriber, error) {
			return fakeWASIWhisperTranscriber{
				transcript: "chunk",
				onTranscribe: func(path string) {
					calls++
					chunkPaths = append(chunkPaths, path)
				},
			}, nil
		},
	}

	got, err := adapter.Transcribe(context.Background(), telegram.AudioInput{
		MediaType: "audio/wav",
		FileName:  "long.wav",
		Data:      input,
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if got != "chunk\nchunk" {
		t.Fatalf("transcript = %q, want joined chunk transcripts", got)
	}
	if calls != 2 {
		t.Fatalf("transcribe calls = %d, want 2 fixed-window chunks", calls)
	}
	for _, path := range chunkPaths {
		if !strings.HasSuffix(path, ".wav") {
			t.Fatalf("chunk path = %q, want .wav", path)
		}
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("expected chunk tempfile cleanup for %s, stat err: %v", path, statErr)
		}
	}
}

func TestWASIWhisperAudioTranscriber_NonWAVMissingFFmpegReturnsTypedPreprocessEvidence(t *testing.T) {
	t.Setenv("PATH", filepath.Join(t.TempDir(), "missing-bin"))
	var transcriberConstructed bool
	adapter := &wasiWhisperAudioTranscriber{
		modelPath: filepath.Join(t.TempDir(), "ggml-tiny.en.bin"),
		newTranscriber: func(context.Context, string) (wasiWhisperTranscriber, error) {
			transcriberConstructed = true
			return fakeWASIWhisperTranscriber{transcript: "should not run"}, nil
		},
	}

	_, err := adapter.Transcribe(context.Background(), telegram.AudioInput{
		MediaType: "audio/ogg",
		FileName:  "voice.ogg",
		Data:      []byte("OggS"),
	})
	if err == nil {
		t.Fatal("Transcribe returned nil error without ffmpeg")
	}
	if !strings.Contains(err.Error(), "audio_preprocess_unavailable") {
		t.Fatalf("error = %v, want audio_preprocess_unavailable evidence", err)
	}
	if transcriberConstructed {
		t.Fatal("transcriber was constructed even though preprocessing failed")
	}
}

func TestResolveTelegramAudioTranscriber_PriorityLocalWASIHTTP(t *testing.T) {
	local := fakeMainAudioTranscriber{transcript: "local"}
	wasi := fakeMainAudioTranscriber{transcript: "wasi"}
	http := fakeMainAudioTranscriber{transcript: "http"}
	restore := stubTranscriberConstructors(t, local, wasi, http)
	defer restore()

	got, err := resolveTelegramAudioTranscriber().Transcribe(context.Background(), telegram.AudioInput{})
	if err != nil {
		t.Fatalf("Transcribe local: %v", err)
	}
	if got != "local" {
		t.Fatalf("expected local first, got %q", got)
	}

	restore()
	restore = stubTranscriberConstructors(t, nil, wasi, http)
	defer restore()
	got, err = resolveTelegramAudioTranscriber().Transcribe(context.Background(), telegram.AudioInput{})
	if err != nil {
		t.Fatalf("Transcribe wasi: %v", err)
	}
	if got != "wasi" {
		t.Fatalf("expected WASI before HTTP, got %q", got)
	}

	restore()
	restore = stubTranscriberConstructors(t, nil, nil, http)
	defer restore()
	got, err = resolveTelegramAudioTranscriber().Transcribe(context.Background(), telegram.AudioInput{})
	if err != nil {
		t.Fatalf("Transcribe http: %v", err)
	}
	if got != "http" {
		t.Fatalf("expected HTTP fallback, got %q", got)
	}
}

type fakeWASIWhisperTranscriber struct {
	transcript   string
	onTranscribe func(path string)
}

func (f fakeWASIWhisperTranscriber) TranscribeWAV(_ context.Context, path string) (string, error) {
	if f.onTranscribe != nil {
		f.onTranscribe(path)
	}
	return f.transcript, nil
}

func (f fakeWASIWhisperTranscriber) Close(_ context.Context) error {
	return nil
}

type fakeMainAudioTranscriber struct {
	transcript string
}

func (f fakeMainAudioTranscriber) Transcribe(context.Context, telegram.AudioInput) (string, error) {
	return f.transcript, nil
}

func stubTranscriberConstructors(t *testing.T, local, wasi, http telegram.AudioTranscriber) func() {
	t.Helper()
	oldLocal := newLocalTelegramAudioTranscriber
	oldWASI := newWASIWhisperTelegramAudioTranscriber
	oldHTTP := newHTTPSTTTelegramAudioTranscriber
	newLocalTelegramAudioTranscriber = func() telegram.AudioTranscriber { return local }
	newWASIWhisperTelegramAudioTranscriber = func() telegram.AudioTranscriber { return wasi }
	newHTTPSTTTelegramAudioTranscriber = func() telegram.AudioTranscriber { return http }
	return func() {
		newLocalTelegramAudioTranscriber = oldLocal
		newWASIWhisperTelegramAudioTranscriber = oldWASI
		newHTTPSTTTelegramAudioTranscriber = oldHTTP
	}
}

func testTelegramWAVPCM16Mono16k(t *testing.T, samples []int16) []byte {
	t.Helper()
	dataSize := len(samples) * 2
	raw := make([]byte, 44+dataSize)
	copy(raw[0:4], "RIFF")
	binary.LittleEndian.PutUint32(raw[4:8], uint32(36+dataSize))
	copy(raw[8:12], "WAVE")
	copy(raw[12:16], "fmt ")
	binary.LittleEndian.PutUint32(raw[16:20], 16)
	binary.LittleEndian.PutUint16(raw[20:22], 1)
	binary.LittleEndian.PutUint16(raw[22:24], 1)
	binary.LittleEndian.PutUint32(raw[24:28], 16000)
	binary.LittleEndian.PutUint32(raw[28:32], 16000*2)
	binary.LittleEndian.PutUint16(raw[32:34], 2)
	binary.LittleEndian.PutUint16(raw[34:36], 16)
	copy(raw[36:40], "data")
	binary.LittleEndian.PutUint32(raw[40:44], uint32(dataSize))
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(raw[44+(i*2):46+(i*2)], uint16(sample))
	}
	return raw
}
