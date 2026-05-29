package artifact

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

func TestEnsureDownloadsAndVerifiesArtifact(t *testing.T) {
	body := []byte("downloaded speech artifact bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	artifact := testArtifact(body, server.URL)
	got, err := Ensure(context.Background(), artifact, cacheDir, server.Client())
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	want := filepath.Join(cacheDir, artifact.Filename)
	if got != want {
		t.Fatalf("Ensure path = %q, want %q", got, want)
	}
	read, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("read final artifact: %v", err)
	}
	if string(read) != string(body) {
		t.Fatalf("final bytes = %q, want %q", read, body)
	}
	assertNoPartialArtifact(t, cacheDir)
}

func TestEnsureDownloadsWithDefaultClientWhenHTTPClientIsTypedNil(t *testing.T) {
	body := []byte("downloaded with default client")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()

	var client *http.Client
	cacheDir := t.TempDir()
	artifact := testArtifact(body, server.URL)
	got, err := Ensure(context.Background(), artifact, cacheDir, client)
	if err != nil {
		t.Fatalf("Ensure with typed nil client: %v", err)
	}
	if got != filepath.Join(cacheDir, artifact.Filename) {
		t.Fatalf("Ensure path = %q", got)
	}
}

func TestEnsureUsesVerifiedCacheWithoutNetwork(t *testing.T) {
	body := []byte("cached speech artifact bytes")
	cacheDir := t.TempDir()
	artifact := testArtifact(body, "https://example.invalid/voice.onnx")
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

func TestEnsureRejectsChecksumMismatchAndRemovesPartial(t *testing.T) {
	expected := []byte("expected speech artifact bytes")
	wrong := append([]byte(nil), expected...)
	wrong[0] = 'X'
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(wrong)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	artifact := testArtifact(expected, server.URL)
	_, err := Ensure(context.Background(), artifact, cacheDir, server.Client())
	if !cacheErrorCodeIs(err, ChecksumMismatch) {
		t.Fatalf("Ensure error = %v, want %s", err, ChecksumMismatch)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, artifact.Filename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("final artifact exists after failure: %v", err)
	}
	assertNoPartialArtifact(t, cacheDir)
}

func TestEnsureRejectsUnsafeArtifactFilename(t *testing.T) {
	cacheDir := t.TempDir()
	for _, filename := range []string{"../voice.onnx", `voices\\voice.onnx`, ".", ".."} {
		artifact := testArtifact([]byte("speech artifact bytes"), "https://example.invalid/voice.onnx")
		artifact.Filename = filename
		_, err := Ensure(context.Background(), artifact, cacheDir, nil)
		if !cacheErrorCodeIs(err, InvalidArtifact) {
			t.Fatalf("Ensure(%q) error = %v, want %s", filename, err, InvalidArtifact)
		}
	}
}

func testArtifact(body []byte, url string) Artifact {
	return Artifact{
		Filename:  "voice.onnx",
		URL:       url,
		SizeBytes: int64(len(body)),
		SHA256:    sha256Hex(body),
	}
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func cacheErrorCodeIs(err error, code string) bool {
	var cacheErr *CacheError
	if !errors.As(err, &cacheErr) {
		return false
	}
	return cacheErr.Code == code
}

func assertNoPartialArtifact(t *testing.T, cacheDir string) {
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
