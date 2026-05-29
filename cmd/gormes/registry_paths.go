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

// profileAudioCacheDir returns the profile-scoped audio cache directory
// for the named profile. Used by gateway channels that know their profile ID.
func profileAudioCacheDir(profileID string) string {
	if profileID == "" {
		return defaultAudioCacheDir()
	}
	contract := config.NewProfileStorageContract(config.GormesBaseHome())
	root, err := contract.ProfileCacheDir(profileID)
	if err != nil {
		return defaultAudioCacheDir()
	}
	return filepath.Join(root, "audio")
}

func defaultTranscriptionCacheDir() string {
	if override := strings.TrimSpace(os.Getenv("GORMES_STT_CACHE_DIR")); override != "" {
		return override
	}
	return filepath.Join(config.GormesHome(), "cache", "whisper")
}

// profileTranscriptionCacheDir returns the profile-scoped transcription
// cache directory for the named profile. Note that whisper model files are
// large (~75MB); per-profile isolation means each profile downloads its own.
func profileTranscriptionCacheDir(profileID string) string {
	if override := strings.TrimSpace(os.Getenv("GORMES_STT_CACHE_DIR")); override != "" {
		return override
	}
	if profileID == "" {
		return defaultTranscriptionCacheDir()
	}
	contract := config.NewProfileStorageContract(config.GormesBaseHome())
	root, err := contract.ProfileCacheDir(profileID)
	if err != nil {
		return defaultTranscriptionCacheDir()
	}
	return filepath.Join(root, "whisper")
}
