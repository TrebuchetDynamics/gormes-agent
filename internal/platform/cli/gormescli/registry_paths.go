package gormescli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func DefaultBrowserArtifactDir() string {
	return filepath.Join(filepath.Dir(config.ToolAuditLogPath()), "browser-artifacts")
}

func DefaultWebAuthStorePath() string {
	return filepath.Join(config.GormesHome(), "auth.json")
}

func DefaultMemoryToolDir() string {
	return filepath.Join(config.GormesHome(), "memory")
}

func DefaultAudioCacheDir() string {
	return filepath.Join(config.GormesHome(), "cache", "audio")
}

// profileAudioCacheDir returns the profile-scoped audio cache directory
// for the named profile. Used by gateway channels that know their profile ID.
func ProfileAudioCacheDir(profileID string) string {
	if profileID == "" {
		return DefaultAudioCacheDir()
	}
	contract := config.NewProfileStorageContract(config.GormesBaseHome())
	root, err := contract.ProfileCacheDir(profileID)
	if err != nil {
		return DefaultAudioCacheDir()
	}
	return filepath.Join(root, "audio")
}

func DefaultTranscriptionCacheDir() string {
	if override := strings.TrimSpace(os.Getenv("GORMES_STT_CACHE_DIR")); override != "" {
		return override
	}
	return filepath.Join(config.GormesHome(), "cache", "whisper")
}

// profileTranscriptionCacheDir returns the profile-scoped transcription
// cache directory for the named profile. Note that whisper model files are
// large (~75MB); per-profile isolation means each profile downloads its own.
func ProfileTranscriptionCacheDir(profileID string) string {
	if override := strings.TrimSpace(os.Getenv("GORMES_STT_CACHE_DIR")); override != "" {
		return override
	}
	if profileID == "" {
		return DefaultTranscriptionCacheDir()
	}
	contract := config.NewProfileStorageContract(config.GormesBaseHome())
	root, err := contract.ProfileCacheDir(profileID)
	if err != nil {
		return DefaultTranscriptionCacheDir()
	}
	return filepath.Join(root, "whisper")
}
