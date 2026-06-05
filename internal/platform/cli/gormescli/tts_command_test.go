package gormescli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTTSPiperListShowsCachedModels(t *testing.T) {
	restoreTTSPiperCommandEnv(t)
	cache := t.TempDir()
	model := filepath.Join(cache, "en_US-lessac-medium.onnx")
	if err := os.WriteFile(model, []byte(strings.Repeat("m", 2048)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(model+".json", []byte(`{"audio":{"sample_rate":22050}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)

	stdout, stderr, err := executeRootCommandForTest(NewTTSCommand(), "piper", "list")
	if err != nil {
		t.Fatalf("tts piper list: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Piper model cache: " + cache, model, "Selected default: " + model} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestTTSPiperListShowsUnusableCachedModels(t *testing.T) {
	restoreTTSPiperCommandEnv(t)
	cache := t.TempDir()
	usable := filepath.Join(cache, "usable.onnx")
	broken := filepath.Join(cache, "broken.onnx")
	if err := os.WriteFile(usable, []byte(strings.Repeat("m", 2048)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(broken, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)

	stdout, stderr, err := executeRootCommandForTest(NewTTSCommand(), "piper", "list")
	if err != nil {
		t.Fatalf("tts piper list: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Usable Piper models:", usable, "Unusable Piper models:", broken, "reason=Piper model broken.onnx is too small"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "Selected default: "+broken) {
		t.Fatalf("unusable model selected as default:\n%s", stdout)
	}
}

func TestTTSPiperVoicesShowsNamedInstallShortcuts(t *testing.T) {
	stdout, stderr, err := executeRootCommandForTest(NewTTSCommand(), "piper", "voices")
	if err != nil {
		t.Fatalf("tts piper voices: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"lessac-medium", "language=en_US", "quality=medium", "en_US-lessac-medium.onnx"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestTTSPiperCleanUnusableRemovesOnlyBrokenModels(t *testing.T) {
	restoreTTSPiperCommandEnv(t)
	cache := t.TempDir()
	usable := filepath.Join(cache, "usable.onnx")
	broken := filepath.Join(cache, "broken.onnx")
	if err := os.WriteFile(usable, []byte(strings.Repeat("m", 2048)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(broken, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)

	stdout, stderr, err := executeRootCommandForTest(NewTTSCommand(), "piper", "clean", "--unusable")
	if err != nil {
		t.Fatalf("tts piper clean --unusable: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Removed unusable Piper model: "+broken) {
		t.Fatalf("stdout missing removed model:\n%s", stdout)
	}
	if _, err := os.Stat(usable); err != nil {
		t.Fatalf("usable model removed: %v", err)
	}
	if _, err := os.Stat(broken); !os.IsNotExist(err) {
		t.Fatalf("broken model still exists or stat err=%v", err)
	}
}

func TestTTSPiperCleanRequiresUnusableFlag(t *testing.T) {
	restoreTTSPiperCommandEnv(t)
	stdout, stderr, err := executeRootCommandForTest(NewTTSCommand(), "piper", "clean")
	if err == nil || !strings.Contains(err.Error(), "without --unusable") {
		t.Fatalf("tts piper clean err = %v, want --unusable guard\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
}

func TestTTSPiperRepairFetchesMissingKnownVoiceSidecar(t *testing.T) {
	restoreTTSPiperCommandEnv(t)
	cache := t.TempDir()
	model := filepath.Join(cache, "en_US-lessac-medium.onnx")
	if err := os.WriteFile(model, []byte(strings.Repeat("m", 2048)), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json") {
			t.Fatalf("unexpected repair path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"audio":{"sample_rate":22050}}`))
	}))
	defer server.Close()
	t.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)
	t.Setenv("GORMES_TTS_PIPER_REGISTRY_BASE_URL", server.URL)

	stdout, stderr, err := executeRootCommandForTest(NewTTSCommand(), "piper", "repair")
	if err != nil {
		t.Fatalf("tts piper repair: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	want := model + ".json"
	if !strings.Contains(stdout, "Repaired Piper sidecar: "+want) {
		t.Fatalf("stdout missing repaired sidecar %q:\n%s", want, stdout)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("sidecar missing after repair: %v", err)
	}
}

func TestTTSPiperInstallNamedVoiceDownloadsIntoCache(t *testing.T) {
	restoreTTSPiperCommandEnv(t)
	cache := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/en/en_US/lessac/medium/en_US-lessac-medium.onnx"):
			_, _ = w.Write([]byte(strings.Repeat("m", 2048)))
		case strings.HasSuffix(r.URL.Path, "/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json"):
			_, _ = w.Write([]byte(`{"audio": {}}`))
		default:
			t.Fatalf("unexpected download path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)
	t.Setenv("GORMES_TTS_PIPER_REGISTRY_BASE_URL", server.URL)

	stdout, stderr, err := executeRootCommandForTest(NewTTSCommand(), "piper", "install", "lessac-medium")
	if err != nil {
		t.Fatalf("tts piper install lessac-medium: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	installed := filepath.Join(cache, "en_US-lessac-medium.onnx")
	if !strings.Contains(stdout, "Installed Piper model: "+installed) {
		t.Fatalf("stdout = %s", stdout)
	}
	data, err := os.ReadFile(installed)
	if err != nil || len(data) != 2048 {
		t.Fatalf("installed data len = %d, err=%v", len(data), err)
	}
	if _, err := os.Stat(installed + ".json"); err != nil {
		t.Fatalf("installed sidecar missing: %v", err)
	}
}

func TestTTSPiperInstallRejectsTruncatedNamedVoiceDownload(t *testing.T) {
	restoreTTSPiperCommandEnv(t)
	cache := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("too small"))
	}))
	defer server.Close()
	t.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)
	t.Setenv("GORMES_TTS_PIPER_REGISTRY_BASE_URL", server.URL)

	stdout, stderr, err := executeRootCommandForTest(NewTTSCommand(), "piper", "install", "lessac-medium")
	if err == nil || !strings.Contains(err.Error(), "too small") {
		t.Fatalf("tts piper install err = %v, want size validation error\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
}

func TestTTSPiperInstallCopiesModelIntoCache(t *testing.T) {
	restoreTTSPiperCommandEnv(t)
	cache := t.TempDir()
	source := filepath.Join(t.TempDir(), "voice.onnx")
	if err := os.WriteFile(source, []byte("model bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)

	stdout, stderr, err := executeRootCommandForTest(NewTTSCommand(), "piper", "install", source)
	if err != nil {
		t.Fatalf("tts piper install: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	installed := filepath.Join(cache, "voice.onnx")
	if !strings.Contains(stdout, "Installed Piper model: "+installed) {
		t.Fatalf("stdout = %s", stdout)
	}
	data, err := os.ReadFile(installed)
	if err != nil || string(data) != "model bytes" {
		t.Fatalf("installed data = %q, err=%v", data, err)
	}
}

func restoreTTSPiperCommandEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"GORMES_TTS_PIPER_MODEL", "GORMES_TTS_PIPER_MODEL_CACHE", "GORMES_TTS_PIPER_CACHE_DIR", "GORMES_TTS_PIPER_VOICE", "GORMES_TTS_PIPER_BIN", "GORMES_TTS_PIPER_REGISTRY_BASE_URL"} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}
