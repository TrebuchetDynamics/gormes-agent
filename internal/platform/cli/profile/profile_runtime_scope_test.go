package profile

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveProfileRuntimeScopeExplicitMain(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	scope, err := ResolveProfileRuntimeScope(ProfileRuntimeScopeOptions{
		BaseHome:        base,
		ExplicitProfile: "main",
		ListKnownProfiles: func() ([]string, error) {
			return []string{"main", "profile-name"}, nil
		},
	})
	if err != nil {
		t.Fatalf("ResolveProfileRuntimeScope(main) error = %v, want nil", err)
	}
	want := ProfileRuntimeScope{
		ProfileID:     "main",
		BaseHome:      base,
		RuntimeHome:   filepath.Join(base, "profiles", "main"),
		ConfigPath:    filepath.Join(base, "profiles", "main", "config.toml"),
		MemoryDBPath:  filepath.Join(base, "profiles", "main", "memory.db"),
		SessionDBPath: filepath.Join(base, "profiles", "main", "sessions.db"),
		WorkspaceDir:  filepath.Join(base, "profiles", "main", "workspace"),
		CacheDir:      filepath.Join(base, "profiles", "main", "cache"),
		RuntimeDir:    filepath.Join(base, "profiles", "main", "runtime"),
	}
	if !reflect.DeepEqual(scope, want) {
		t.Fatalf("scope = %#v, want %#v", scope, want)
	}
}

func TestResolveProfileRuntimeScopeExplicitNamedProfile(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	scope, err := ResolveProfileRuntimeScope(ProfileRuntimeScopeOptions{
		BaseHome:        base,
		ExplicitProfile: "profile-name",
		ListKnownProfiles: func() ([]string, error) {
			return []string{"main", "profile-name"}, nil
		},
	})
	if err != nil {
		t.Fatalf("ResolveProfileRuntimeScope(profile-name) error = %v, want nil", err)
	}
	if scope.ProfileID != "profile-name" || scope.RuntimeHome != filepath.Join(base, "profiles", "profile-name") {
		t.Fatalf("scope = %#v, want profile-name under profiles/profile-name", scope)
	}
}

func TestResolveProfileRuntimeScopeUsesStickyActiveThenMain(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	stickyScope, err := ResolveProfileRuntimeScope(ProfileRuntimeScopeOptions{
		BaseHome: base,
		ReadActiveProfile: func() (string, error) {
			return "profile-name", nil
		},
		ListKnownProfiles: func() ([]string, error) {
			return []string{"main", "profile-name"}, nil
		},
	})
	if err != nil {
		t.Fatalf("ResolveProfileRuntimeScope(sticky) error = %v, want nil", err)
	}
	if stickyScope.ProfileID != "profile-name" {
		t.Fatalf("sticky ProfileID = %q, want profile-name", stickyScope.ProfileID)
	}

	mainScope, err := ResolveProfileRuntimeScope(ProfileRuntimeScopeOptions{
		BaseHome: base,
		ReadActiveProfile: func() (string, error) {
			return "", ErrActiveProfileUnset
		},
		ListKnownProfiles: func() ([]string, error) {
			return []string{"main", "profile-name"}, nil
		},
	})
	if err != nil {
		t.Fatalf("ResolveProfileRuntimeScope(unset sticky) error = %v, want nil", err)
	}
	if mainScope.ProfileID != "main" {
		t.Fatalf("unset sticky ProfileID = %q, want main", mainScope.ProfileID)
	}
}

func TestResolveProfileRuntimeScopeRejectsDefaultAndMissingProfiles(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	_, err := ResolveProfileRuntimeScope(ProfileRuntimeScopeOptions{
		BaseHome:        base,
		ExplicitProfile: "default",
		ListKnownProfiles: func() ([]string, error) {
			return []string{"main"}, nil
		},
	})
	if !errors.Is(err, ErrProfileNameReserved) {
		t.Fatalf("ResolveProfileRuntimeScope(default) error = %v, want ErrProfileNameReserved", err)
	}

	_, err = ResolveProfileRuntimeScope(ProfileRuntimeScopeOptions{
		BaseHome:        base,
		ExplicitProfile: "ghost",
		ListKnownProfiles: func() ([]string, error) {
			return []string{"main", "profile-name"}, nil
		},
	})
	if !errors.Is(err, ErrProfileRuntimeScopeMissing) {
		t.Fatalf("ResolveProfileRuntimeScope(ghost) error = %v, want ErrProfileRuntimeScopeMissing", err)
	}
}

func TestResolveProfileRuntimeScopeIsPure(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	oldHome := os.Getenv("GORMES_HOME")
	t.Setenv("GORMES_HOME", oldHome)

	_, err := ResolveProfileRuntimeScope(ProfileRuntimeScopeOptions{
		BaseHome:        base,
		ExplicitProfile: "main",
		ListKnownProfiles: func() ([]string, error) {
			return []string{"main"}, nil
		},
	})
	if err != nil {
		t.Fatalf("ResolveProfileRuntimeScope(main) error = %v, want nil", err)
	}
	if got := os.Getenv("GORMES_HOME"); got != oldHome {
		t.Fatalf("GORMES_HOME = %q, want unchanged %q", got, oldHome)
	}
	if _, err := os.Stat(filepath.Join(base, "profiles", "main")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("resolver created profile directory or stat failed unexpectedly: %v", err)
	}
}
