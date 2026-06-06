package testkit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/modelcache"
)

func TinyEnModelPath(t testing.TB, ctx context.Context) string {
	t.Helper()
	if path := strings.TrimSpace(os.Getenv("GORMES_WASI_WHISPER_MODEL")); path != "" {
		if err := modelcache.Verify(path, modelcache.TinyEnModelArtifact); err != nil {
			t.Fatalf("verify %s: %v", path, err)
		}
		return path
	}

	cacheDir := strings.TrimSpace(os.Getenv("GORMES_WASI_WHISPER_MODEL_CACHE"))
	if cacheDir == "" {
		userCache, err := os.UserCacheDir()
		if err != nil {
			t.Fatalf("resolve user cache dir: %v", err)
		}
		cacheDir = filepath.Join(userCache, "gormes", "wasi-whisper")
	}
	path, err := modelcache.Ensure(ctx, modelcache.TinyEnModelArtifact, cacheDir, nil)
	if err != nil {
		t.Skipf("WASI Whisper tiny.en model unavailable in %s: %v", cacheDir, err)
	}
	return path
}

func ReadWhisperWASM(t testing.TB) []byte {
	t.Helper()
	wasm, err := os.ReadFile(WhisperTestdataPath("whisper.wasm"))
	if err != nil {
		t.Fatalf("read whisper.wasm: %v", err)
	}
	return wasm
}

func WhisperTestdataPath(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}
