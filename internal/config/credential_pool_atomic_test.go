package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCredentialPoolAtomicWrite_NoTornWrite(t *testing.T) {
	home := t.TempDir()
	authPath := filepath.Join(home, "auth.json")

	store := credentialPoolAuthStore{
		CredentialPool: map[string][]PooledCredential{
			"openai": {{
				ID:       "key-1",
				AuthType: CredentialAuthAPIKey,
				Label:    "test-key",
			}},
		},
	}

	if err := writeCredentialPoolAuthStore(home, store); err != nil {
		t.Fatalf("writeCredentialPoolAuthStore: %v", err)
	}

	data, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("read auth.json: %v", err)
	}

	var readBack credentialPoolAuthStore
	if err := json.Unmarshal(data, &readBack); err != nil {
		t.Fatalf("stored auth.json is not valid JSON: %v\ncontent: %s", err, strings.TrimSpace(string(data)))
	}

	if len(readBack.CredentialPool) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(readBack.CredentialPool))
	}
	entries := readBack.CredentialPool["openai"]
	if len(entries) != 1 || entries[0].ID != "key-1" {
		t.Fatalf("credential not preserved: %+v", entries)
	}
}

func TestCredentialPoolAtomicWrite_PreservesExistingMode(t *testing.T) {
	home := t.TempDir()
	authPath := filepath.Join(home, "auth.json")

	store := credentialPoolAuthStore{
		CredentialPool: map[string][]PooledCredential{
			"openai": {{ID: "key-1", AuthType: CredentialAuthAPIKey}},
		},
	}

	if err := writeCredentialPoolAuthStore(home, store); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := os.Chmod(authPath, 0o640); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	store.CredentialPool["openai"] = append(store.CredentialPool["openai"], PooledCredential{
		ID: "key-2", AuthType: CredentialAuthAPIKey,
	})
	if err := writeCredentialPoolAuthStore(home, store); err != nil {
		t.Fatalf("second write: %v", err)
	}

	info, err := os.Stat(authPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("mode = %#o, want 0640", got)
	}
}

func TestCredentialPoolAtomicWrite_TempfileCleanedOnError(t *testing.T) {
	home := t.TempDir()

	store := credentialPoolAuthStore{
		CredentialPool: map[string][]PooledCredential{
			"openai": {{ID: "key-1", AuthType: CredentialAuthAPIKey}},
		},
	}

	if err := writeCredentialPoolAuthStore(home, store); err != nil {
		t.Fatalf("write: %v", err)
	}

	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".auth-") {
			t.Fatalf("orphan tempfile found: %s", e.Name())
		}
	}
}

func TestCredentialPoolAtomicWrite_EmptyStore(t *testing.T) {
	home := t.TempDir()
	authPath := filepath.Join(home, "auth.json")

	store := credentialPoolAuthStore{
		CredentialPool: make(map[string][]PooledCredential),
	}

	if err := writeCredentialPoolAuthStore(home, store); err != nil {
		t.Fatalf("writeCredentialPoolAuthStore: %v", err)
	}

	data, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("read auth.json: %v", err)
	}

	var readBack credentialPoolAuthStore
	if err := json.Unmarshal(data, &readBack); err != nil {
		t.Fatalf("empty store is not valid JSON: %v", err)
	}
}
