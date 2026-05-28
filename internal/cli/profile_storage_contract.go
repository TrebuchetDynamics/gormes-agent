package cli

import (
	"errors"
	"path/filepath"
	"strings"
)

// ErrProfileBaseHomeRequired is returned when the homogeneous profile storage
// contract is constructed without the global Gormes base home (normally
// ~/.gormes). The contract is pure path math; callers remain responsible for
// resolving environment defaults and creating or migrating directories.
var ErrProfileBaseHomeRequired = errors.New("gormes base home is required")

// ProfileStorageContract describes the target multi-profile storage layout for
// Gormes: global metadata lives under BaseHome and every runnable profile,
// including "main", lives under BaseHome/profiles/<name>.
//
// This helper intentionally does not read environment variables, stat the
// filesystem, or create directories. That keeps migration and setup flows
// explicit while giving callers one central contract for future profile-ready
// config/provider/runtime paths.
type ProfileStorageContract struct {
	baseHome string
}

// NewProfileStorageContract returns a pure path resolver for the homogeneous
// profile layout rooted at baseHome (for example ~/.gormes).
func NewProfileStorageContract(baseHome string) (ProfileStorageContract, error) {
	baseHome = strings.TrimSpace(baseHome)
	if baseHome == "" {
		return ProfileStorageContract{}, ErrProfileBaseHomeRequired
	}
	return ProfileStorageContract{baseHome: filepath.Clean(baseHome)}, nil
}

// BaseHome returns the global Gormes base directory that owns metadata such as
// active_profile and the profiles directory itself.
func (c ProfileStorageContract) BaseHome() string {
	return c.baseHome
}

// ProfilesRoot returns the directory that contains all runnable profile roots.
func (c ProfileStorageContract) ProfilesRoot() string {
	return filepath.Join(c.baseHome, "profiles")
}

// ProfileRoot resolves name to BaseHome/profiles/<name> for any valid profile
// name.
func (c ProfileStorageContract) ProfileRoot(name string) (string, error) {
	if strings.TrimSpace(c.baseHome) == "" {
		return "", ErrProfileBaseHomeRequired
	}
	name = strings.TrimSpace(name)
	if err := ValidateProfileName(name); err != nil {
		return "", err
	}
	return filepath.Join(c.ProfilesRoot(), name), nil
}

// ProfileConfigPath returns the profile-local config.toml path for name.
func (c ProfileStorageContract) ProfileConfigPath(name string) (string, error) {
	root, err := c.ProfileRoot(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "config.toml"), nil
}

// ProfileMemoryDBPath returns the profile-local memory database path for name.
func (c ProfileStorageContract) ProfileMemoryDBPath(name string) (string, error) {
	root, err := c.ProfileRoot(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "memory.db"), nil
}

// ProfileSessionDBPath returns the profile-local session database path for name.
func (c ProfileStorageContract) ProfileSessionDBPath(name string) (string, error) {
	root, err := c.ProfileRoot(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "sessions.db"), nil
}

// ProfileWorkspaceDir returns the profile-local workspace directory for name.
// This is the default first entry in a profile's workspace list. Additional
// workspace paths are configured explicitly in profile config.
func (c ProfileStorageContract) ProfileWorkspaceDir(name string) (string, error) {
	root, err := c.ProfileRoot(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "workspace"), nil
}

// ProfileCacheDir returns the profile-local cache directory for name.
// Subdirectories (audio, whisper) live under this root.
func (c ProfileStorageContract) ProfileCacheDir(name string) (string, error) {
	root, err := c.ProfileRoot(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "cache"), nil
}

// ProfileRuntimeDir returns the profile-local runtime directory for name.
// Gateway lifecycle state (gateway_state.json, gateway.pid, gateway-locks, gateway.log)
// lives under this root.
func (c ProfileStorageContract) ProfileRuntimeDir(name string) (string, error) {
	root, err := c.ProfileRoot(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "runtime"), nil
}

// ResolveProfileRuntimeRoot returns the runnable profile root for baseHome and
// name without creating or migrating directories. Every valid profile,
// including the built-in main profile, resolves under BaseHome/profiles/<name>
// so runtime state is never mixed into the global metadata home.
func ResolveProfileRuntimeRoot(baseHome, name string) (string, error) {
	contract, err := NewProfileStorageContract(baseHome)
	if err != nil {
		return "", err
	}
	return contract.ProfileRoot(strings.TrimSpace(name))
}
