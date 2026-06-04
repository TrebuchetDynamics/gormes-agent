package tts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPiperSynthesizerRunsConfiguredLocalNeuralCommand(t *testing.T) {
	out := filepath.Join(t.TempDir(), "speech.wav")
	runner := &fakePiperRunner{writeOutput: true}
	synth := &PiperSynthesizer{Binary: "piper", ModelPath: "/models/en.onnx", Runner: runner, MaxTextLength: 100}

	result, err := synth.Synthesize(context.Background(), Request{Text: "hello piper", OutputPath: out})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if result.FilePath != out || result.Format != "wav" || result.Bytes == 0 {
		t.Fatalf("result = %+v, want WAV output evidence", result)
	}
	if runner.last.Binary != "piper" || runner.last.ModelPath != "/models/en.onnx" || runner.last.Text != "hello piper" || runner.last.OutputPath != out {
		t.Fatalf("runner request = %+v", runner.last)
	}
}

func TestNewPiperSynthesizerFromEnvDiscoversCachedModel(t *testing.T) {
	restorePiperEnv(t)
	cache := t.TempDir()
	model := filepath.Join(cache, "en_US-lessac-medium.onnx")
	if err := os.WriteFile(model, bytesOfSize(2048), 0o600); err != nil {
		t.Fatal(err)
	}
	writePiperSidecar(t, model)
	os.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)
	os.Setenv("GORMES_TTS_PIPER_BIN", "piper-test")

	synth := NewPiperSynthesizerFromEnv()
	if synth == nil {
		t.Fatal("NewPiperSynthesizerFromEnv returned nil, want cached model runtime")
	}
	if synth.ModelPath != model || synth.Binary != "piper-test" {
		t.Fatalf("synth = %+v, want cached model %q and configured binary", synth, model)
	}
}

func TestNewPiperSynthesizerFromEnvPrefersConfiguredVoiceInCache(t *testing.T) {
	restorePiperEnv(t)
	cache := t.TempDir()
	low := filepath.Join(cache, "en_US-lessac-low.onnx")
	high := filepath.Join(cache, "en_US-amy-medium.onnx")
	if err := os.WriteFile(low, bytesOfSize(2048), 0o600); err != nil {
		t.Fatal(err)
	}
	writePiperSidecar(t, low)
	if err := os.WriteFile(high, bytesOfSize(2048), 0o600); err != nil {
		t.Fatal(err)
	}
	writePiperSidecar(t, high)
	os.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)
	os.Setenv("GORMES_TTS_PIPER_VOICE", "amy")

	synth := NewPiperSynthesizerFromEnv()
	if synth == nil || synth.ModelPath != high {
		t.Fatalf("synth = %+v, want preferred cached voice %q", synth, high)
	}
}

func TestCachedPiperModelsListsNestedONNXFiles(t *testing.T) {
	restorePiperEnv(t)
	cache := t.TempDir()
	model := filepath.Join(cache, "voices", "en_US-lessac-medium.onnx")
	if err := os.MkdirAll(filepath.Dir(model), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(model, bytesOfSize(2048), 0o600); err != nil {
		t.Fatal(err)
	}
	writePiperSidecar(t, model)
	if err := os.WriteFile(filepath.Join(cache, "ignored.txt"), []byte("ignored"), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)

	models := CachedPiperModels()
	if len(models) != 1 || models[0] != model {
		t.Fatalf("CachedPiperModels() = %#v, want only %q", models, model)
	}
}

func TestCachedPiperModelsSkipsMissingKnownVoiceSidecar(t *testing.T) {
	restorePiperEnv(t)
	cache := t.TempDir()
	model := filepath.Join(cache, "en_US-lessac-medium.onnx")
	if err := os.WriteFile(model, bytesOfSize(2048), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)

	statuses := CachedPiperModelStatuses()
	if len(statuses) != 1 || statuses[0].Usable || !strings.Contains(statuses[0].Reason, "missing sidecar") {
		t.Fatalf("statuses = %#v, want unusable missing sidecar", statuses)
	}
}

func TestCachedPiperModelsSkipsEmptyONNXFiles(t *testing.T) {
	restorePiperEnv(t)
	cache := t.TempDir()
	empty := filepath.Join(cache, "empty.onnx")
	usable := filepath.Join(cache, "usable.onnx")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(usable, bytesOfSize(2048), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)

	models := CachedPiperModels()
	if len(models) != 1 || models[0] != usable {
		t.Fatalf("CachedPiperModels() = %#v, want only usable model %q", models, usable)
	}
}

func TestResolvePiperModelSourceResolvesNamedVoice(t *testing.T) {
	restorePiperEnv(t)
	os.Setenv("GORMES_TTS_PIPER_REGISTRY_BASE_URL", "https://models.example/piper")

	got := ResolvePiperModelSource("lessac-medium")
	want := "https://models.example/piper/en/en_US/lessac/medium/en_US-lessac-medium.onnx"
	if got != want {
		t.Fatalf("ResolvePiperModelSource() = %q, want %q", got, want)
	}
}

func TestInstallPiperModelDownloadsNamedVoiceAndSidecarIntoCache(t *testing.T) {
	restorePiperEnv(t)
	cache := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".onnx.json") {
			_, _ = w.Write([]byte(`{"audio":{"sample_rate":22050}}`))
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/en/en_US/lessac/medium/en_US-lessac-medium.onnx") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write(bytesOfSize(2048))
	}))
	defer server.Close()
	os.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)
	os.Setenv("GORMES_TTS_PIPER_REGISTRY_BASE_URL", server.URL)

	installed, err := InstallPiperModel(context.Background(), "lessac-medium")
	if err != nil {
		t.Fatalf("InstallPiperModel: %v", err)
	}
	if installed != filepath.Join(cache, "en_US-lessac-medium.onnx") {
		t.Fatalf("installed = %q", installed)
	}
	data, err := os.ReadFile(installed)
	if err != nil || len(data) != 2048 {
		t.Fatalf("installed data len = %d, err=%v", len(data), err)
	}
	if _, err := os.Stat(installed + ".json"); err != nil {
		t.Fatalf("installed sidecar missing: %v", err)
	}
}

func TestInstallPiperModelRejectsMissingNamedVoiceSidecar(t *testing.T) {
	restorePiperEnv(t)
	cache := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".onnx.json") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(bytesOfSize(2048))
	}))
	defer server.Close()
	os.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)
	os.Setenv("GORMES_TTS_PIPER_REGISTRY_BASE_URL", server.URL)

	installed, err := InstallPiperModel(context.Background(), "lessac-medium")
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("InstallPiperModel err = %v, want sidecar download error", err)
	}
	if _, statErr := os.Stat(installed); !os.IsNotExist(statErr) {
		t.Fatalf("model should be removed after sidecar failure, statErr=%v", statErr)
	}
}

func TestInstallPiperModelRejectsTruncatedNamedVoiceDownload(t *testing.T) {
	restorePiperEnv(t)
	cache := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("too small"))
	}))
	defer server.Close()
	os.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)
	os.Setenv("GORMES_TTS_PIPER_REGISTRY_BASE_URL", server.URL)

	installed, err := InstallPiperModel(context.Background(), "lessac-medium")
	if err == nil || !strings.Contains(err.Error(), "too small") {
		t.Fatalf("InstallPiperModel err = %v, want size validation error", err)
	}
	if IsPiperModelUsable(installed) {
		t.Fatalf("truncated model %q should not be usable", installed)
	}
}

func TestInstallPiperModelCopiesLocalONNXIntoCache(t *testing.T) {
	restorePiperEnv(t)
	cache := t.TempDir()
	source := filepath.Join(t.TempDir(), "voice.onnx")
	if err := os.WriteFile(source, []byte("model bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)

	installed, err := InstallPiperModel(context.Background(), source)
	if err != nil {
		t.Fatalf("InstallPiperModel: %v", err)
	}
	if installed != filepath.Join(cache, "voice.onnx") {
		t.Fatalf("installed = %q", installed)
	}
	data, err := os.ReadFile(installed)
	if err != nil || string(data) != "model bytes" {
		t.Fatalf("installed data = %q, err=%v", data, err)
	}
}

func TestRemoveUnusablePiperModelsDeletesOnlyBrokenFiles(t *testing.T) {
	restorePiperEnv(t)
	cache := t.TempDir()
	usable := filepath.Join(cache, "usable.onnx")
	broken := filepath.Join(cache, "broken.onnx")
	if err := os.WriteFile(usable, bytesOfSize(2048), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(broken, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	os.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)

	removed, err := RemoveUnusablePiperModels()
	if err != nil {
		t.Fatalf("RemoveUnusablePiperModels: %v", err)
	}
	if len(removed) != 1 || removed[0] != broken {
		t.Fatalf("removed = %#v, want only %q", removed, broken)
	}
	if _, err := os.Stat(usable); err != nil {
		t.Fatalf("usable model removed: %v", err)
	}
	if _, err := os.Stat(broken); !os.IsNotExist(err) {
		t.Fatalf("broken model still exists or stat err=%v", err)
	}
}

func TestInstallPiperModelRejectsNonONNX(t *testing.T) {
	restorePiperEnv(t)
	os.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", t.TempDir())

	if _, err := InstallPiperModel(context.Background(), filepath.Join(t.TempDir(), "voice.txt")); err == nil {
		t.Fatal("InstallPiperModel accepted non-ONNX source")
	}
}

func TestPiperSynthesizerRejectsMissingModel(t *testing.T) {
	_, err := (&PiperSynthesizer{Binary: "piper"}).Synthesize(context.Background(), Request{Text: "hello", OutputPath: filepath.Join(t.TempDir(), "speech.wav")})
	if !IsErrorCode(err, ErrorCodeProviderUnavailable) {
		t.Fatalf("err = %v, want provider unavailable", err)
	}
}

func restorePiperEnv(t *testing.T) {
	t.Helper()
	keys := []string{"GORMES_TTS_PIPER_MODEL", "GORMES_TTS_PIPER_MODEL_CACHE", "GORMES_TTS_PIPER_CACHE_DIR", "GORMES_TTS_PIPER_VOICE", "GORMES_TTS_PIPER_BIN", "GORMES_TTS_PIPER_REGISTRY_BASE_URL"}
	original := make(map[string]string, len(keys))
	present := make(map[string]bool, len(keys))
	for _, key := range keys {
		original[key], present[key] = os.LookupEnv(key)
		os.Unsetenv(key)
	}
	t.Cleanup(func() {
		for _, key := range keys {
			if present[key] {
				os.Setenv(key, original[key])
			} else {
				os.Unsetenv(key)
			}
		}
	})
}

type fakePiperRunner struct {
	last        PiperCommandRequest
	writeOutput bool
}

func writePiperSidecar(t *testing.T, model string) {
	t.Helper()
	if err := os.WriteFile(model+".json", []byte(`{"audio":{"sample_rate":22050}}`), 0o600); err != nil {
		t.Fatal(err)
	}
}

func bytesOfSize(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte('a' + (i % 26))
	}
	return data
}

func (f *fakePiperRunner) RunPiper(_ context.Context, req PiperCommandRequest) error {
	f.last = req
	if f.writeOutput {
		return os.WriteFile(req.OutputPath, []byte("RIFFfakeWAVE"), 0o600)
	}
	return nil
}
