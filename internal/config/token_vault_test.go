package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTokenVaultRegisterCredentialFile(t *testing.T) {
	hermesHome := t.TempDir()
	writeFixtureCredential(t, hermesHome, "credentials/acme.txt", "plain-existing-token")

	vault, err := NewTokenVault(TokenVaultOptions{HermesHome: hermesHome})
	if err != nil {
		t.Fatal(err)
	}
	mount, err := vault.RegisterCredentialFile("credentials/acme.txt")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(mount.HostPath, hermesHome+string(os.PathSeparator)) {
		t.Fatalf("HostPath = %q, want under temp Hermes home %q", mount.HostPath, hermesHome)
	}
	if mount.ContainerPath != "/root/.hermes/credentials/acme.txt" {
		t.Fatalf("ContainerPath = %q", mount.ContainerPath)
	}
	if mount.Source != TokenVaultSourceRegistered {
		t.Fatalf("Source = %q", mount.Source)
	}
}

func TestTokenVaultDefaultUsesGormesHomeNotHermesHome(t *testing.T) {
	root := t.TempDir()
	gormesHome := filepath.Join(root, "gormes")
	hermesHome := filepath.Join(root, "hermes")
	writeFixtureCredential(t, gormesHome, "credentials/native.txt", "plain-gormes-token")
	writeFixtureCredential(t, hermesHome, "credentials/native.txt", "plain-hermes-token")
	t.Setenv("GORMES_HOME", gormesHome)
	t.Setenv("HERMES_HOME", hermesHome)

	vault, err := NewTokenVault(TokenVaultOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mount, err := vault.RegisterCredentialFile("credentials/native.txt")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(mount.HostPath, gormesHome+string(os.PathSeparator)) {
		t.Fatalf("HostPath = %q, want under Gormes home %q", mount.HostPath, gormesHome)
	}
	if strings.HasPrefix(mount.HostPath, hermesHome+string(os.PathSeparator)) {
		t.Fatalf("HostPath = %q, want not under poisoned Hermes home %q", mount.HostPath, hermesHome)
	}
}

func TestTokenVaultRejectsUnsafePaths(t *testing.T) {
	hermesHome := t.TempDir()
	writeFixtureCredential(t, hermesHome, "safe.txt", "plain-existing-token")
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("plain-existing-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Symlink(outside, filepath.Join(hermesHome, "escape-link")); err != nil {
			t.Fatal(err)
		}
	}

	vault, err := NewTokenVault(TokenVaultOptions{HermesHome: hermesHome})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		path string
		code TokenVaultReason
	}{
		{name: "absolute", path: filepath.Join(hermesHome, "safe.txt"), code: TokenVaultReasonAbsolutePath},
		{name: "traversal", path: "../safe.txt", code: TokenVaultReasonTraversal},
	}
	if runtime.GOOS != "windows" {
		cases = append(cases, struct {
			name string
			path string
			code TokenVaultReason
		}{name: "symlink escape", path: "escape-link", code: TokenVaultReasonSymlinkEscape})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := vault.RegisterCredentialFile(tc.path)
			if err == nil {
				t.Fatalf("RegisterCredentialFile(%q) err = nil, want rejection", tc.path)
			}
			var vaultErr *TokenVaultError
			if !AsTokenVaultError(err, &vaultErr) {
				t.Fatalf("err = %T %v, want TokenVaultError", err, err)
			}
			if vaultErr.Reason != tc.code {
				t.Fatalf("Reason = %q, want %q", vaultErr.Reason, tc.code)
			}
			evidence := vaultErr.Evidence()
			if evidence.HostPath != "" {
				t.Fatalf("evidence HostPath = %q, want redacted empty host path", evidence.HostPath)
			}
			if strings.Contains(evidence.Message, hermesHome) || strings.Contains(evidence.Message, outside) {
				t.Fatalf("evidence leaked host path: %q", evidence.Message)
			}
		})
	}
}

func TestTokenVaultSessionIsolation(t *testing.T) {
	homeA := t.TempDir()
	homeB := t.TempDir()
	writeFixtureCredential(t, homeA, "a.txt", "plain-existing-token")
	writeFixtureCredential(t, homeB, "b.txt", "plain-existing-token")

	vaultA, err := NewTokenVault(TokenVaultOptions{HermesHome: homeA})
	if err != nil {
		t.Fatal(err)
	}
	vaultB, err := NewTokenVault(TokenVaultOptions{HermesHome: homeB})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := vaultA.RegisterCredentialFile("a.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := vaultB.RegisterCredentialFile("b.txt"); err != nil {
		t.Fatal(err)
	}

	mountsA := vaultA.Mounts()
	mountsB := vaultB.Mounts()
	if len(mountsA) != 1 || mountsA[0].ContainerPath != "/root/.hermes/a.txt" {
		t.Fatalf("vaultA mounts = %#v", mountsA)
	}
	if len(mountsB) != 1 || mountsB[0].ContainerPath != "/root/.hermes/b.txt" {
		t.Fatalf("vaultB mounts = %#v", mountsB)
	}
}

func TestTokenVaultConfigCredentialFiles(t *testing.T) {
	hermesHome := t.TempDir()
	writeFixtureCredential(t, hermesHome, "tokens/acme.txt", "plain-existing-token")
	writeFixtureCredential(t, hermesHome, "tokens/shared.txt", "plain-existing-token")
	configPath := filepath.Join(hermesHome, "config.yaml")
	if err := os.WriteFile(configPath, []byte(`terminal:
  credential_files:
    - tokens/acme.txt
    - missing.txt
    - ../escape.txt
    - tokens/shared.txt
`), 0o600); err != nil {
		t.Fatal(err)
	}

	vault, err := NewTokenVault(TokenVaultOptions{HermesHome: hermesHome})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := vault.RegisterCredentialFile("tokens/shared.txt"); err != nil {
		t.Fatal(err)
	}
	result, err := vault.LoadConfigCredentialFiles(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mounts) != 2 {
		t.Fatalf("Mounts len = %d, mounts=%#v rejected=%#v", len(result.Mounts), result.Mounts, result.Rejected)
	}
	if got := mountByContainer(result.Mounts, "/root/.hermes/tokens/acme.txt"); got == nil || got.Source != TokenVaultSourceConfig {
		t.Fatalf("config mount missing or wrong source: %#v", result.Mounts)
	}
	if got := mountByContainer(result.Mounts, "/root/.hermes/tokens/shared.txt"); got == nil || got.Source != TokenVaultSourceRegistered {
		t.Fatalf("registered mount should win dedupe by container path: %#v", result.Mounts)
	}
	if len(result.Rejected) != 2 {
		t.Fatalf("Rejected len = %d, want missing + traversal: %#v", len(result.Rejected), result.Rejected)
	}
	for _, evidence := range result.Rejected {
		if evidence.HostPath != "" {
			t.Fatalf("rejected evidence leaked HostPath: %#v", evidence)
		}
		if strings.Contains(evidence.Message, hermesHome) {
			t.Fatalf("rejected evidence leaked Hermes home: %q", evidence.Message)
		}
	}
}

func TestTokenVaultClear(t *testing.T) {
	hermesHome := t.TempDir()
	writeFixtureCredential(t, hermesHome, "registered.txt", "plain-existing-token")
	writeFixtureCredential(t, hermesHome, "configured.txt", "plain-existing-token")
	configPath := filepath.Join(hermesHome, "config.yaml")
	if err := os.WriteFile(configPath, []byte(`terminal:
  credential_files:
    - configured.txt
`), 0o600); err != nil {
		t.Fatal(err)
	}

	vault, err := NewTokenVault(TokenVaultOptions{HermesHome: hermesHome})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := vault.RegisterCredentialFile("registered.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := vault.LoadConfigCredentialFiles(configPath); err != nil {
		t.Fatal(err)
	}
	vault.Clear()
	mounts := vault.Mounts()
	if len(mounts) != 1 || mounts[0].ContainerPath != "/root/.hermes/configured.txt" || mounts[0].Source != TokenVaultSourceConfig {
		t.Fatalf("Mounts after Clear = %#v, want only deterministic config-derived mount", mounts)
	}
}

func writeFixtureCredential(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mountByContainer(mounts []CredentialFileMount, containerPath string) *CredentialFileMount {
	for i := range mounts {
		if mounts[i].ContainerPath == containerPath {
			return &mounts[i]
		}
	}
	return nil
}
