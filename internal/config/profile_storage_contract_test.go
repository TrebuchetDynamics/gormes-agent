package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfileStorageContractLegacyRootPreservesCurrentDBPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	contract := CurrentProfileStorageContract()
	if contract.Scope != ProfileStorageScopeBaseHome || contract.Root != home || contract.BaseHome != home {
		t.Fatalf("contract = %+v, want legacy base-home scope at GORMES_HOME", contract)
	}
	if contract.MemoryDBPath != filepath.Join(home, "memory.db") || contract.SessionDBPath != filepath.Join(home, "sessions.db") {
		t.Fatalf("db paths = %q/%q, want legacy root paths", contract.MemoryDBPath, contract.SessionDBPath)
	}
	if contract.MemoryDBPath != MemoryDBPath() || contract.SessionDBPath != SessionDBPath() {
		t.Fatalf("contract db paths must match public helpers: contract=%q/%q helpers=%q/%q", contract.MemoryDBPath, contract.SessionDBPath, MemoryDBPath(), SessionDBPath())
	}
}

func TestProfileStorageContractMaterializedMainPreservesCurrentDBPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	mainRoot := filepath.Join(home, "profiles", "main")
	if err := os.MkdirAll(mainRoot, 0o700); err != nil {
		t.Fatal(err)
	}

	contract := CurrentProfileStorageContract()
	if contract.Scope != ProfileStorageScopeProfileRoot || contract.ProfileID != DefaultProfileID || contract.Root != mainRoot || contract.BaseHome != home {
		t.Fatalf("contract = %+v, want materialized main profile scope", contract)
	}
	if contract.MemoryDBPath != filepath.Join(mainRoot, "memory.db") || contract.SessionDBPath != filepath.Join(mainRoot, "sessions.db") {
		t.Fatalf("db paths = %q/%q, want profiles/main paths", contract.MemoryDBPath, contract.SessionDBPath)
	}
	if contract.MemoryDBPath != MemoryDBPath() || contract.SessionDBPath != SessionDBPath() {
		t.Fatalf("contract db paths must match public helpers: contract=%q/%q helpers=%q/%q", contract.MemoryDBPath, contract.SessionDBPath, MemoryDBPath(), SessionDBPath())
	}
}

func TestProfileStorageContractActiveNamedProfilePreservesCurrentDBPaths(t *testing.T) {
	base := t.TempDir()
	profileRoot := filepath.Join(base, "profiles", "coder")
	t.Setenv("GORMES_HOME", profileRoot)

	contract := CurrentProfileStorageContract()
	if contract.Scope != ProfileStorageScopeProfileRoot || contract.ProfileID != "coder" || contract.Root != profileRoot || contract.BaseHome != base {
		t.Fatalf("contract = %+v, want active named profile scope", contract)
	}
	if contract.MemoryDBPath != filepath.Join(profileRoot, "memory.db") || contract.SessionDBPath != filepath.Join(profileRoot, "sessions.db") {
		t.Fatalf("db paths = %q/%q, want active profile root paths", contract.MemoryDBPath, contract.SessionDBPath)
	}
	if contract.MemoryDBPath != MemoryDBPath() || contract.SessionDBPath != SessionDBPath() {
		t.Fatalf("contract db paths must match public helpers: contract=%q/%q helpers=%q/%q", contract.MemoryDBPath, contract.SessionDBPath, MemoryDBPath(), SessionDBPath())
	}
}

func TestProfileStorageContractProfileCacheDirUsesBaseHomeWithoutCreatingDirs(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	contract := NewProfileStorageContract(base)

	got, err := contract.ProfileCacheDir("coder")
	if err != nil {
		t.Fatalf("ProfileCacheDir: %v", err)
	}
	if want := filepath.Join(base, "profiles", "coder", "cache"); got != want {
		t.Fatalf("ProfileCacheDir(coder) = %q, want %q", got, want)
	}
}
