package cli

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestProfileStorageContractHomogeneousRoots(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	contract, err := NewProfileStorageContract(base)
	if err != nil {
		t.Fatalf("NewProfileStorageContract() error = %v, want nil", err)
	}

	for _, name := range []string{"main", "coder"} {
		t.Run(name, func(t *testing.T) {
			got, err := contract.ProfileRoot(name)
			if err != nil {
				t.Fatalf("ProfileRoot(%q) error = %v, want nil", name, err)
			}
			if want := filepath.Join(base, "profiles", name); got != want {
				t.Fatalf("ProfileRoot(%q) = %q, want homogeneous root %q", name, got, want)
			}
		})
	}

	if got, want := contract.BaseHome(), base; got != want {
		t.Fatalf("BaseHome() = %q, want global base %q", got, want)
	}
	if got, want := contract.ProfilesRoot(), filepath.Join(base, "profiles"); got != want {
		t.Fatalf("ProfilesRoot() = %q, want %q", got, want)
	}
}

func TestProfileStorageContractProfileLocalPaths(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	contract, err := NewProfileStorageContract(base)
	if err != nil {
		t.Fatalf("NewProfileStorageContract() error = %v, want nil", err)
	}

	for _, tc := range []struct {
		name        string
		configPath  string
		memoryPath  string
		sessionPath string
		workspace   string
		cache       string
		runtime     string
	}{
		{
			name:        "main",
			configPath:  filepath.Join(base, "profiles", "main", "config.toml"),
			memoryPath:  filepath.Join(base, "profiles", "main", "memory.db"),
			sessionPath: filepath.Join(base, "profiles", "main", "sessions.db"),
			workspace:   filepath.Join(base, "profiles", "main", "workspace"),
			cache:       filepath.Join(base, "profiles", "main", "cache"),
			runtime:     filepath.Join(base, "profiles", "main", "runtime"),
		},
		{
			name:        "coder",
			configPath:  filepath.Join(base, "profiles", "coder", "config.toml"),
			memoryPath:  filepath.Join(base, "profiles", "coder", "memory.db"),
			sessionPath: filepath.Join(base, "profiles", "coder", "sessions.db"),
			workspace:   filepath.Join(base, "profiles", "coder", "workspace"),
			cache:       filepath.Join(base, "profiles", "coder", "cache"),
			runtime:     filepath.Join(base, "profiles", "coder", "runtime"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := contract.ProfileConfigPath(tc.name); err != nil || got != tc.configPath {
				t.Fatalf("ProfileConfigPath(%q) = %q, %v; want %q, nil", tc.name, got, err, tc.configPath)
			}
			if got, err := contract.ProfileMemoryDBPath(tc.name); err != nil || got != tc.memoryPath {
				t.Fatalf("ProfileMemoryDBPath(%q) = %q, %v; want %q, nil", tc.name, got, err, tc.memoryPath)
			}
			if got, err := contract.ProfileSessionDBPath(tc.name); err != nil || got != tc.sessionPath {
				t.Fatalf("ProfileSessionDBPath(%q) = %q, %v; want %q, nil", tc.name, got, err, tc.sessionPath)
			}
			if got, err := contract.ProfileWorkspaceDir(tc.name); err != nil || got != tc.workspace {
				t.Fatalf("ProfileWorkspaceDir(%q) = %q, %v; want %q, nil", tc.name, got, err, tc.workspace)
			}
			if got, err := contract.ProfileCacheDir(tc.name); err != nil || got != tc.cache {
				t.Fatalf("ProfileCacheDir(%q) = %q, %v; want %q, nil", tc.name, got, err, tc.cache)
			}
			if got, err := contract.ProfileRuntimeDir(tc.name); err != nil || got != tc.runtime {
				t.Fatalf("ProfileRuntimeDir(%q) = %q, %v; want %q, nil", tc.name, got, err, tc.runtime)
			}
		})
	}
}

func TestProfileStorageContractRejectsInvalidInputs(t *testing.T) {
	if _, err := NewProfileStorageContract(" "); !errors.Is(err, ErrProfileBaseHomeRequired) {
		t.Fatalf("NewProfileStorageContract(blank) error = %v, want ErrProfileBaseHomeRequired", err)
	}
	if got, err := (ProfileStorageContract{}).ProfileRoot("main"); !errors.Is(err, ErrProfileBaseHomeRequired) || got != "" {
		t.Fatalf("zero-value ProfileRoot(main) = %q, %v; want empty path and ErrProfileBaseHomeRequired", got, err)
	}

	contract, err := NewProfileStorageContract(filepath.Join(t.TempDir(), ".gormes"))
	if err != nil {
		t.Fatalf("NewProfileStorageContract() error = %v, want nil", err)
	}
	got, err := contract.ProfileRoot("Coder")
	if !errors.Is(err, ErrProfileNameInvalidChars) {
		t.Fatalf("ProfileRoot(Coder) error = %v, want ErrProfileNameInvalidChars", err)
	}
	if got != "" {
		t.Fatalf("ProfileRoot(Coder) = %q, want empty string on validation failure", got)
	}
	got, err = contract.ProfileRoot("default")
	if !errors.Is(err, ErrProfileNameReserved) {
		t.Fatalf("ProfileRoot(default) error = %v, want ErrProfileNameReserved", err)
	}
	if got != "" {
		t.Fatalf("ProfileRoot(default) = %q, want empty string on validation failure", got)
	}
}

func TestResolveProfileRuntimeRootMainUsesProfilesRoot(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	got, err := ResolveProfileRuntimeRoot(base, "main")
	if err != nil {
		t.Fatalf("ResolveProfileRuntimeRoot(main) error = %v, want nil", err)
	}
	want := filepath.Join(base, "profiles", "main")
	if got != want {
		t.Fatalf("ResolveProfileRuntimeRoot(main) = %q, want %q", got, want)
	}
}

func TestResolveProfileRuntimeRootNamedUsesHomogeneousContract(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	got, err := ResolveProfileRuntimeRoot(base, "coder")
	if err != nil {
		t.Fatalf("ResolveProfileRuntimeRoot(coder) error = %v, want nil", err)
	}
	if want := filepath.Join(base, "profiles", "coder"); got != want {
		t.Fatalf("ResolveProfileRuntimeRoot(coder) = %q, want %q", got, want)
	}
}

func TestProfileStorageContractNoFilesystemAccess(t *testing.T) {
	const nonexistent = "/this/path/definitely/does/not/exist/anywhere/.gormes"
	contract, err := NewProfileStorageContract(nonexistent)
	if err != nil {
		t.Fatalf("NewProfileStorageContract(nonexistent) error = %v, want nil", err)
	}
	got, err := contract.ProfileRoot("main")
	if err != nil {
		t.Fatalf("ProfileRoot(main) under nonexistent base error = %v, want nil", err)
	}
	if want := filepath.Join(nonexistent, "profiles", "main"); got != want {
		t.Fatalf("ProfileRoot(main) under nonexistent base = %q, want %q", got, want)
	}
}
