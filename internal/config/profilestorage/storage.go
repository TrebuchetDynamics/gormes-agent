package profilestorage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/paths"
)

const DefaultProfileID = "main"

// Scope identifies whether the active runtime storage root is the legacy base
// home or a profile-local runnable root.
type Scope string

const (
	ScopeBaseHome    Scope = "base_home"
	ScopeProfileRoot Scope = "profile_root"
)

// Contract is the resolved path contract for profile-owned runtime data. It is
// pure path resolution: callers remain responsible for directory creation,
// migration, locking, and persistence.
type Contract struct {
	BaseHome      string `json:"base_home"`
	Root          string `json:"root"`
	Scope         Scope  `json:"scope"`
	ProfileID     string `json:"profile_id,omitempty"`
	MemoryDBPath  string `json:"memory_db_path"`
	SessionDBPath string `json:"session_db_path"`
	WorkspaceDir  string `json:"workspace_dir"`
	CacheDir      string `json:"cache_dir"`
	RuntimeDir    string `json:"runtime_dir"`
}

// Current resolves the storage contract for the current process GORMES_HOME
// while preserving the legacy profiles/main materialization probe used by
// MemoryDBPath and SessionDBPath.
func Current() Contract {
	return New(paths.GormesHome())
}

// New resolves storage paths for gormesHome. If gormesHome is already a
// profile root, the scope is that profile. If it is a base home with a
// materialized profiles/main directory, main is the scope. All other base homes
// keep legacy root database paths.
func New(gormesHome string) Contract {
	home := filepath.Clean(strings.TrimSpace(gormesHome))
	if home == "." || home == "" {
		home = paths.GormesHome()
	}
	base := paths.GormesBaseHomeFor(home)
	if base != home && filepath.Base(filepath.Dir(home)) == "profiles" {
		profileID := filepath.Base(home)
		return profileStorageContract(base, home, ScopeProfileRoot, profileID)
	}
	mainRoot := filepath.Join(home, "profiles", DefaultProfileID)
	if isDirectory(mainRoot) {
		return profileStorageContract(home, mainRoot, ScopeProfileRoot, DefaultProfileID)
	}
	return profileStorageContract(home, home, ScopeBaseHome, "")
}

// ProfileRoot returns the homogeneous profile root for profileID under the
// contract's base home. It does not stat or create the directory.
func (c Contract) ProfileRoot(profileID string) (string, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return "", fmt.Errorf("config: profile id is required")
	}
	if strings.ContainsAny(profileID, `/\\`) || profileID == "." || profileID == ".." {
		return "", fmt.Errorf("config: invalid profile id %q", profileID)
	}
	base := strings.TrimSpace(c.BaseHome)
	if base == "" {
		base = paths.GormesBaseHome()
	}
	return filepath.Join(base, "profiles", profileID), nil
}

// ProfileCacheDir returns the profile-local cache directory for profileID.
func (c Contract) ProfileCacheDir(profileID string) (string, error) {
	root, err := c.ProfileRoot(profileID)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "cache"), nil
}

func profileStorageContract(baseHome, root string, scope Scope, profileID string) Contract {
	return Contract{
		BaseHome:      filepath.Clean(baseHome),
		Root:          filepath.Clean(root),
		Scope:         scope,
		ProfileID:     profileID,
		MemoryDBPath:  filepath.Join(root, "memory.db"),
		SessionDBPath: filepath.Join(root, "sessions.db"),
		WorkspaceDir:  filepath.Join(root, "workspace"),
		CacheDir:      filepath.Join(root, "cache"),
		RuntimeDir:    filepath.Join(root, "runtime"),
	}
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
