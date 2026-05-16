package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func defaultBrowserArtifactDir() string {
	return filepath.Join(filepath.Dir(config.ToolAuditLogPath()), "browser-artifacts")
}

func defaultWebAuthStorePath() string {
	return filepath.Join(config.GormesHome(), "auth.json")
}

func defaultMemoryToolDir() string {
	return filepath.Join(config.GormesHome(), "memory")
}

func defaultAudioCacheDir() string {
	return filepath.Join(config.GormesHome(), "cache", "audio")
}

func defaultTranscriptionCacheDir() string {
	if override := strings.TrimSpace(os.Getenv("GORMES_STT_CACHE_DIR")); override != "" {
		return override
	}
	return filepath.Join(config.GormesHome(), "cache", "whisper")
}
