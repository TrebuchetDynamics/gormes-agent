//go:build !gormes_lite && !slim

package tools

import (
	"context"
	"encoding/binary"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper"
)

func TestLocalSTTProvider_Available(t *testing.T) {
	p := NewLocalSTTProvider(t.TempDir())
	if !p.Available(context.Background()) {
		t.Fatal("LocalSTTProvider should be available with default artifact config")
	}
}

func TestLocalSTTProvider_Transcribe_RejectsEmptyPath(t *testing.T) {
	p := NewLocalSTTProvider(t.TempDir())
	_, err := p.Transcribe(context.Background(), TranscriptionProviderRequest{AudioPath: ""})
	if err == nil {
		t.Fatal("expected error for empty audio path")
	}
}

func TestLocalSTTProvider_Transcribe_RejectsMissingFile(t *testing.T) {
	p := NewLocalSTTProvider(t.TempDir())
	_, err := p.Transcribe(context.Background(), TranscriptionProviderRequest{AudioPath: "/nonexistent/audio.wav"})
	if err == nil {
		t.Fatal("expected error for missing audio file")
	}
}

func TestLocalSTTProvider_Transcribe_RejectsNonWAV(t *testing.T) {
	dir := t.TempDir()
	nonWAV := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(nonWAV, []byte("not audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := NewLocalSTTProvider(dir)
	_, err := p.Transcribe(context.Background(), TranscriptionProviderRequest{AudioPath: nonWAV})
	if err == nil {
		t.Fatal("expected error for non-WAV file")
	}
}

func TestLocalSTTProvider_Transcribe_ConvertsOggBeforeWASITranscribe(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "voice.ogg")
	if err := os.WriteFile(audioPath, []byte("OggS fake opus payload"), 0o600); err != nil {
		t.Fatal(err)
	}

	var convertedFrom string
	var transcribedPath string
	p := NewLocalSTTProvider(dir)
	p.ensureModel = func(context.Context) (string, error) {
		return filepath.Join(dir, "ggml-tiny.en.bin"), nil
	}
	p.convertToWAV = func(_ context.Context, inputPath, outputPath string) error {
		convertedFrom = inputPath
		return os.WriteFile(outputPath, testLocalSTTWAVPCM16Mono16k(t, []int16{1, 2, 3}), 0o600)
	}
	p.newTranscriber = func(context.Context, string) (localSTTWhisperTranscriber, error) {
		return fakeLocalSTTWhisperTranscriber{
			transcript: "hola mundo",
			onTranscribe: func(path string) {
				transcribedPath = path
			},
		}, nil
	}

	result, err := p.Transcribe(context.Background(), TranscriptionProviderRequest{
		AudioPath: audioPath,
		Language:  "es",
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if result.Transcript != "hola mundo" {
		t.Fatalf("Transcript = %q, want converted transcript", result.Transcript)
	}
	if !strings.HasSuffix(convertedFrom, ".ogg") {
		t.Fatalf("convertedFrom = %q, want OGG input path", convertedFrom)
	}
	if !strings.HasSuffix(transcribedPath, ".wav") {
		t.Fatalf("transcribedPath = %q, want WAV temp file", transcribedPath)
	}
	if _, err := os.Stat(transcribedPath); !os.IsNotExist(err) {
		t.Fatalf("expected temp WAV cleanup, stat err: %v", err)
	}
}

func TestLocalSTTProvider_Transcribe_JFKFixture(t *testing.T) {
	jfkPath := filepath.Join("..", "wasi", "whisper", "testdata", "jfk.wav")
	if _, err := os.Stat(jfkPath); err != nil {
		t.Skip("jfk.wav test fixture not available:", err)
	}
	cacheDir := localSTTFixtureModelCacheDir(t)
	modelPath := filepath.Join(cacheDir, whisper.TinyEnModelArtifact.Filename)
	if _, err := os.Stat(modelPath); err != nil {
		if os.IsNotExist(err) {
			t.Skipf("WASI Whisper tiny.en model is not cached at %s; run internal/tools/whisper integration tests or set GORMES_WASI_WHISPER_MODEL_CACHE", modelPath)
		}
		t.Fatalf("stat cached model: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	p := NewLocalSTTProvider(cacheDir)
	p.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network disabled in local STT provider fixture test")
	})}
	result, err := p.Transcribe(ctx, TranscriptionProviderRequest{AudioPath: jfkPath})
	if err != nil {
		if modelDownloadUnavailable(err) {
			t.Skipf("WASI Whisper tiny.en model unavailable from cache %s: %v", cacheDir, err)
		}
		t.Fatalf("Transcribe jfk.wav: %v", err)
	}
	if result.Transcript == "" {
		t.Fatal("empty transcript from jfk.wav")
	}
	if result.Provider != "local" {
		t.Fatalf("provider = %q, want local", result.Provider)
	}
	if result.Model != "tiny.en" {
		t.Fatalf("model = %q, want tiny.en", result.Model)
	}

	normalized := strings.ToLower(result.Transcript)
	for _, want := range []string{"ask not", "your country", "what you can do"} {
		if !strings.Contains(normalized, want) {
			t.Fatalf("transcript missing %q:\n%s", want, result.Transcript)
		}
	}
}

func localSTTFixtureModelCacheDir(t *testing.T) string {
	t.Helper()
	if cacheDir := strings.TrimSpace(os.Getenv("GORMES_WASI_WHISPER_MODEL_CACHE")); cacheDir != "" {
		return cacheDir
	}
	userCache, err := os.UserCacheDir()
	if err != nil {
		t.Skipf("resolve user cache dir: %v", err)
	}
	return filepath.Join(userCache, "gormes", "wasi-whisper")
}

func modelDownloadUnavailable(err error) bool {
	var cacheErr *whisper.ModelCacheError
	return errors.As(err, &cacheErr) && cacheErr.Code == whisper.ModelCacheDownloadFailed
}

type fakeLocalSTTWhisperTranscriber struct {
	transcript   string
	onTranscribe func(string)
}

func (f fakeLocalSTTWhisperTranscriber) TranscribeWAV(_ context.Context, path string) (string, error) {
	if f.onTranscribe != nil {
		f.onTranscribe(path)
	}
	return f.transcript, nil
}

func (f fakeLocalSTTWhisperTranscriber) Close(context.Context) error {
	return nil
}

func testLocalSTTWAVPCM16Mono16k(t *testing.T, samples []int16) []byte {
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
