package modelcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveModelArtifactSupportsHermesLocalModelTiers(t *testing.T) {
	cases := []struct {
		model    string
		wantName string
		wantFile string
	}{
		{"", "base", "ggml-base.bin"},
		{"auto", "base", "ggml-base.bin"},
		{"tiny.en", "tiny.en", "ggml-tiny.en.bin"},
		{"ggml-tiny.bin", "tiny", "ggml-tiny.bin"},
		{"base.en", "base.en", "ggml-base.en.bin"},
		{"base", "base", "ggml-base.bin"},
		{"small.en", "small.en", "ggml-small.en.bin"},
		{"small", "small", "ggml-small.bin"},
		{"unknown", "base", "ggml-base.bin"},
	}
	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			gotName, artifact := ResolveModelArtifact(tc.model, "")
			if gotName != tc.wantName || artifact.Filename != tc.wantFile {
				t.Fatalf("ResolveModelArtifact(%q) = (%q, %q), want (%q, %q)", tc.model, gotName, artifact.Filename, tc.wantName, tc.wantFile)
			}
			if artifact.URL == "" || artifact.SizeBytes <= 0 || artifact.SHA256 == "" {
				t.Fatalf("artifact metadata incomplete: %+v", artifact)
			}
		})
	}
}

func TestModelArtifactTinyEnPinned(t *testing.T) {
	artifact := TinyEnModelArtifact
	if artifact.Filename != "ggml-tiny.en.bin" {
		t.Fatalf("Filename = %q", artifact.Filename)
	}
	if artifact.URL != "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.en.bin" {
		t.Fatalf("URL = %q", artifact.URL)
	}
	if artifact.SizeBytes != 77704715 {
		t.Fatalf("SizeBytes = %d", artifact.SizeBytes)
	}
	if artifact.SHA256 != "921e4cf8686fdd993dcd081a5da5b6c365bfde1162e72b08d75ac75289920b1f" {
		t.Fatalf("SHA256 = %q", artifact.SHA256)
	}
}

func TestEnsureUsesVerifiedExistingCache(t *testing.T) {
	cacheDir := t.TempDir()
	body := []byte("cached model bytes")
	artifact := testModelArtifact(body, "https://example.invalid/model.bin")
	finalPath := filepath.Join(cacheDir, artifact.Filename)
	if err := os.WriteFile(finalPath, body, 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("Ensure made a network request for a verified cache hit")
		return nil, nil
	})}

	got, err := Ensure(context.Background(), artifact, cacheDir, client)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if got != finalPath {
		t.Fatalf("Ensure path = %q, want %q", got, finalPath)
	}
}

func TestEnsureDownloadsAndVerifies(t *testing.T) {
	body := []byte("downloaded model bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	artifact := testModelArtifact(body, server.URL)
	got, err := Ensure(context.Background(), artifact, cacheDir, server.Client())
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if got != filepath.Join(cacheDir, artifact.Filename) {
		t.Fatalf("Ensure path = %q", got)
	}
	read, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("read final model: %v", err)
	}
	if string(read) != string(body) {
		t.Fatalf("final model bytes = %q, want %q", read, body)
	}
	assertNoModelCachePartial(t, cacheDir, artifact)
}

func TestEnsureRejectsChecksumMismatchAndRemovesPartial(t *testing.T) {
	expected := []byte("expected model bytes")
	wrong := append([]byte(nil), expected...)
	wrong[0] = 'X'
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(wrong)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	artifact := testModelArtifact(expected, server.URL)
	_, err := Ensure(context.Background(), artifact, cacheDir, server.Client())
	if !modelCacheErrorCodeIs(err, ModelCacheChecksumMismatch) {
		t.Fatalf("Ensure error = %v, want %s", err, ModelCacheChecksumMismatch)
	}
	assertNoFinalModel(t, cacheDir, artifact)
	assertNoModelCachePartial(t, cacheDir, artifact)
}

func TestEnsureRejectsShortBodyAndRemovesPartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("short"))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	artifact := testModelArtifact([]byte("expected longer model bytes"), server.URL)
	_, err := Ensure(context.Background(), artifact, cacheDir, server.Client())
	if !modelCacheErrorCodeIs(err, ModelCacheSizeMismatch) {
		t.Fatalf("Ensure error = %v, want %s", err, ModelCacheSizeMismatch)
	}
	assertNoFinalModel(t, cacheDir, artifact)
	assertNoModelCachePartial(t, cacheDir, artifact)
}

func TestEnsureRejectsHTTPStatusAndLeavesNoFinalFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "try later", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	artifact := testModelArtifact([]byte("expected model bytes"), server.URL)
	_, err := Ensure(context.Background(), artifact, cacheDir, server.Client())
	if !modelCacheErrorCodeIs(err, ModelCacheBadStatus) {
		t.Fatalf("Ensure error = %v, want %s", err, ModelCacheBadStatus)
	}
	assertNoFinalModel(t, cacheDir, artifact)
	assertNoModelCachePartial(t, cacheDir, artifact)
}

func TestEnsureRequiresCacheDir(t *testing.T) {
	artifact := testModelArtifact([]byte("expected model bytes"), "https://example.invalid/model.bin")
	_, err := Ensure(context.Background(), artifact, "", nil)
	if !modelCacheErrorCodeIs(err, ModelCacheInvalidCacheDir) {
		t.Fatalf("Ensure error = %v, want %s", err, ModelCacheInvalidCacheDir)
	}
}

func TestEnsureRejectsUnsafeArtifactFilename(t *testing.T) {
	cacheDir := t.TempDir()
	for _, filename := range []string{"../model.bin", `models\model.bin`, ".", ".."} {
		artifact := testModelArtifact([]byte("expected model bytes"), "https://example.invalid/model.bin")
		artifact.Filename = filename
		_, err := Ensure(context.Background(), artifact, cacheDir, nil)
		if !modelCacheErrorCodeIs(err, ModelCacheInvalidArtifact) {
			t.Fatalf("Ensure(%q) error = %v, want %s", filename, err, ModelCacheInvalidArtifact)
		}
	}
}

func testModelArtifact(body []byte, url string) ModelArtifact {
	return ModelArtifact{
		Filename:  "model.bin",
		URL:       url,
		SizeBytes: int64(len(body)),
		SHA256:    sha256Hex(body),
	}
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func modelCacheErrorCodeIs(err error, code string) bool {
	var cacheErr *ModelCacheError
	if !errors.As(err, &cacheErr) {
		return false
	}
	return cacheErr.Code == code
}

func assertNoFinalModel(t *testing.T, cacheDir string, artifact ModelArtifact) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(cacheDir, artifact.Filename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("final model exists after failure: %v", err)
	}
}

func assertNoModelCachePartial(t *testing.T, cacheDir string, artifact ModelArtifact) {
	t.Helper()
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("read cache dir: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".partial") {
			t.Fatalf("partial file left behind: %s", entry.Name())
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
