package whisper

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tinyEnModelPath(tb testing.TB, ctx context.Context, onEnsureError func(testing.TB, string, error)) string {
	tb.Helper()
	if path := strings.TrimSpace(os.Getenv("GORMES_WASI_WHISPER_MODEL")); path != "" {
		if err := verifyModelFile(path, TinyEnModelArtifact); err != nil {
			tb.Fatalf("verify %s: %v", path, err)
		}
		return path
	}

	cacheDir := strings.TrimSpace(os.Getenv("GORMES_WASI_WHISPER_MODEL_CACHE"))
	if cacheDir == "" {
		userCache, err := os.UserCacheDir()
		if err != nil {
			tb.Fatalf("resolve user cache dir: %v", err)
		}
		cacheDir = filepath.Join(userCache, "gormes", "wasi-whisper")
	}
	path, err := EnsureModel(ctx, TinyEnModelArtifact, cacheDir, nil)
	if err != nil {
		onEnsureError(tb, cacheDir, err)
	}
	return path
}
