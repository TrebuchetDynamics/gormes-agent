package profile

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ErrProfileRuntimeScopeMissing is returned when the requested runtime profile
// is valid syntactically but absent from the known profile inventory supplied to
// the resolver.
var ErrProfileRuntimeScopeMissing = errors.New("profile runtime scope: profile is missing")

// ErrSelectorHelperUnavailable is returned when profile runtime resolution is
// invoked without a required injected helper seam.
var ErrSelectorHelperUnavailable = errors.New("selector_helper_unavailable")

// ProfileRuntimeScope is the resolved per-profile boundary for a Gormes
// operation. It contains only identity and path data; applying environment
// variables or creating directories is the caller's responsibility.
type ProfileRuntimeScope struct {
	ProfileID     string
	BaseHome      string
	RuntimeHome   string
	ConfigPath    string
	MemoryDBPath  string
	SessionDBPath string
	WorkspaceDir  string
	CacheDir      string
	RuntimeDir    string
}

// ProfileRuntimeScopeOptions carries the external state needed to resolve a
// profile runtime scope without reading global process environment or touching
// the filesystem.
type ReadActiveProfileNameFunc func() (string, error)

type ProfileRuntimeScopeOptions struct {
	BaseHome          string
	ExplicitProfile   string
	ReadActiveProfile ReadActiveProfileNameFunc
	ListKnownProfiles func() ([]string, error)
}

// ResolveProfileRuntimeScope derives the active profile identity and all
// profile-owned paths from explicit input, sticky active profile state, and the
// homogeneous ProfileStorageContract. It is deliberately pure: no environment
// mutation, filesystem stats, or directory creation happen here.
func ResolveProfileRuntimeScope(opts ProfileRuntimeScopeOptions) (ProfileRuntimeScope, error) {
	baseHome := strings.TrimSpace(opts.BaseHome)
	contract, err := NewProfileStorageContract(baseHome)
	if err != nil {
		return ProfileRuntimeScope{}, err
	}

	profileID := strings.TrimSpace(opts.ExplicitProfile)
	if profileID == "" && opts.ReadActiveProfile != nil {
		active, err := opts.ReadActiveProfile()
		if err != nil && !errors.Is(err, ErrActiveProfileUnset) {
			return ProfileRuntimeScope{}, fmt.Errorf("profile runtime scope: read active profile: %w", err)
		}
		profileID = strings.TrimSpace(active)
	}
	if profileID == "" {
		profileID = "main"
	}
	if err := ValidateProfileName(profileID); err != nil {
		return ProfileRuntimeScope{}, fmt.Errorf("profile runtime scope %q: %w", profileID, err)
	}
	if opts.ListKnownProfiles == nil {
		return ProfileRuntimeScope{}, fmt.Errorf("profile runtime scope %q: %w", profileID, ErrSelectorHelperUnavailable)
	}
	known, err := opts.ListKnownProfiles()
	if err != nil {
		return ProfileRuntimeScope{}, fmt.Errorf("profile runtime scope: list profiles: %w", err)
	}
	if !profileRuntimeScopeKnown(profileID, known) {
		return ProfileRuntimeScope{}, fmt.Errorf("profile runtime scope %q: %w", profileID, ErrProfileRuntimeScopeMissing)
	}

	root, err := contract.ProfileRoot(profileID)
	if err != nil {
		return ProfileRuntimeScope{}, err
	}
	return ProfileRuntimeScope{
		ProfileID:     profileID,
		BaseHome:      contract.BaseHome(),
		RuntimeHome:   root,
		ConfigPath:    filepath.Join(root, "config.toml"),
		MemoryDBPath:  filepath.Join(root, "memory.db"),
		SessionDBPath: filepath.Join(root, "sessions.db"),
		WorkspaceDir:  filepath.Join(root, "workspace"),
		CacheDir:      filepath.Join(root, "cache"),
		RuntimeDir:    filepath.Join(root, "runtime"),
	}, nil
}

func profileRuntimeScopeKnown(profileID string, known []string) bool {
	for _, name := range known {
		if strings.TrimSpace(name) == profileID {
			return true
		}
	}
	return false
}
