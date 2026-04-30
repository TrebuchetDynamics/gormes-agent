package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAnthropicAuthStatePrefersKeychainOverJSON(t *testing.T) {
	credentialsPath := filepath.Join(t.TempDir(), ".claude", ".credentials.json")
	writeAnthropicClaudeCredentials(t, credentialsPath, map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken":  "json-access",
			"refreshToken": "json-refresh",
			"expiresAt":    int64(1_900_000_000_000),
		},
	})
	store := NewAnthropicAuthStateStore(AnthropicAuthStateStoreOptions{
		CredentialsPath: credentialsPath,
		Keychain: func(context.Context) (AnthropicClaudeCredentials, error) {
			return AnthropicClaudeCredentials{
				AccessToken:  "keychain-access",
				RefreshToken: "keychain-refresh",
				ExpiresAtMS:  1_900_000_000_000,
			}, nil
		},
		Now: func() time.Time { return time.Unix(1_775_000_000, 0).UTC() },
	})

	status, err := store.CheckAuth(context.Background())
	if err != nil {
		t.Fatalf("CheckAuth returned error: %v", err)
	}
	if status.Code != AnthropicAuthStatusAuthorized || !status.Authenticated {
		t.Fatalf("status = %#v, want authorized keychain credentials", status)
	}
	if status.Source != AnthropicOAuthSourceMacOSKeychain {
		t.Fatalf("Source = %q, want keychain", status.Source)
	}
	if status.Evidence != AnthropicOAuthEvidenceKeychainSelected {
		t.Fatalf("Evidence = %q, want keychain selected", status.Evidence)
	}
	if status.AccessToken != "" || status.RefreshToken != "" {
		t.Fatalf("status leaked secrets: %#v", status)
	}
	rendered, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	if strings.Contains(string(rendered), "keychain-access") || strings.Contains(string(rendered), "keychain-refresh") {
		t.Fatalf("redacted status JSON leaked credentials: %s", rendered)
	}

	credentials := status.Credentials
	if credentials.AccessToken != "keychain-access" || credentials.RefreshToken != "keychain-refresh" {
		t.Fatalf("credentials = %#v, want fake keychain entry", credentials)
	}
	if credentials.Source != AnthropicOAuthSourceMacOSKeychain {
		t.Fatalf("credentials source = %q, want keychain", credentials.Source)
	}
}

func TestAnthropicAuthStatePreservesCorruptJSONBackup(t *testing.T) {
	now := time.Date(2026, 4, 29, 12, 34, 56, 0, time.UTC)
	credentialsPath := filepath.Join(t.TempDir(), ".claude", ".credentials.json")
	if err := os.MkdirAll(filepath.Dir(credentialsPath), 0o700); err != nil {
		t.Fatal(err)
	}
	corrupt := []byte(`{"claudeAiOauth":`)
	if err := os.WriteFile(credentialsPath, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewAnthropicAuthStateStore(AnthropicAuthStateStoreOptions{
		CredentialsPath: credentialsPath,
		Keychain: func(context.Context) (AnthropicClaudeCredentials, error) {
			return AnthropicClaudeCredentials{}, nil
		},
		Now: func() time.Time { return now },
	})

	status, err := store.CheckAuth(context.Background())
	if err != nil {
		t.Fatalf("CheckAuth returned error: %v", err)
	}
	if status.Code != AnthropicAuthStatusCorrupt {
		t.Fatalf("Code = %q, want corrupt", status.Code)
	}
	if status.Evidence != AnthropicOAuthEvidenceCorruptBackup {
		t.Fatalf("Evidence = %q, want corrupt backup", status.Evidence)
	}
	if status.BackupPath == "" {
		t.Fatalf("BackupPath is empty: %#v", status)
	}
	if !strings.Contains(filepath.Base(status.BackupPath), "20260429T123456Z") {
		t.Fatalf("BackupPath = %q, want deterministic timestamp", status.BackupPath)
	}
	original, err := os.ReadFile(credentialsPath)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	if string(original) != string(corrupt) {
		t.Fatalf("original credentials changed: %q", original)
	}
	backup, err := os.ReadFile(status.BackupPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(backup) != string(corrupt) {
		t.Fatalf("backup = %q, want original corrupt bytes", backup)
	}
	if status.AccessToken != "" || status.RefreshToken != "" {
		t.Fatalf("status leaked secrets: %#v", status)
	}
}

func TestAnthropicAuthStateStaleOAuthRequiresRelogin(t *testing.T) {
	now := time.Unix(1_775_000_000, 0).UTC()
	credentialsPath := filepath.Join(t.TempDir(), ".claude", ".credentials.json")
	writeAnthropicClaudeCredentials(t, credentialsPath, map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken": "sk-ant-oat-stale",
			"expiresAt":   now.Add(-time.Minute).UnixMilli(),
		},
	})
	store := NewAnthropicAuthStateStore(AnthropicAuthStateStoreOptions{
		CredentialsPath: credentialsPath,
		Keychain: func(context.Context) (AnthropicClaudeCredentials, error) {
			return AnthropicClaudeCredentials{}, nil
		},
		Now: func() time.Time { return now },
	})

	status, err := store.CheckAuth(context.Background())
	if err != nil {
		t.Fatalf("CheckAuth returned error: %v", err)
	}
	if status.Code != AnthropicAuthStatusReloginRequired || !status.ReloginRequired {
		t.Fatalf("status = %#v, want relogin-required stale OAuth", status)
	}
	if status.Evidence != AnthropicOAuthEvidenceStaleOAuth {
		t.Fatalf("Evidence = %q, want stale OAuth", status.Evidence)
	}
	if status.AccessToken != "" || status.RefreshToken != "" {
		t.Fatalf("status leaked secrets: %#v", status)
	}
}

func writeAnthropicClaudeCredentials(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
