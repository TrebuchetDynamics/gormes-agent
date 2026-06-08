package credentials

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCredentialPoolBorrowedSourcesSanitizedAtDiskBoundary(t *testing.T) {
	home := t.TempDir()
	entries := []PooledCredential{{
		ID:                 "bws-openrouter",
		Label:              "OpenRouter from Bitwarden",
		AuthType:           CredentialAuthAPIKey,
		Source:             "bitwarden",
		OwnerProfile:       "main",
		AccessToken:        "sk-borrowed-access-token",
		RefreshToken:       "borrowed-refresh-token",
		AgentKey:           "borrowed-agent-key",
		AgentKeyID:         "agent-key-id-safe",
		AgentKeyExpiresAt:  "2030-01-02T03:04:05Z",
		LastStatus:         CredentialStatusOK,
		LastErrorReason:    "rate_limit",
		LastErrorMessage:   "safe operator diagnostic",
		LastErrorResetAt:   12345,
		RequestCount:       7,
		MaxConcurrentLease: 1,
	}}
	if err := SaveCredentialPoolEntries(CredentialPoolOptions{HermesHome: home, Provider: "openrouter"}, entries); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}

	raw := readRawAuthJSON(t, home)
	for _, leak := range []string{"sk-borrowed-access-token", "borrowed-refresh-token", "borrowed-agent-key"} {
		if strings.Contains(raw, leak) {
			t.Fatalf("borrowed credential leaked %q in auth.json: %s", leak, raw)
		}
	}
	stored := readCredentialPoolEntry(t, home, "openrouter", 0)
	if stored["source"] != "bitwarden" || stored["label"] != "OpenRouter from Bitwarden" || stored["last_status"] != CredentialStatusOK || stored["request_count"].(float64) != 7 {
		t.Fatalf("safe borrowed metadata not preserved: %#v", stored)
	}
	if _, ok := stored["access_token"]; ok {
		t.Fatalf("borrowed access_token persisted: %#v", stored)
	}
	if _, ok := stored["refresh_token"]; ok {
		t.Fatalf("borrowed refresh_token persisted: %#v", stored)
	}
	if _, ok := stored["agent_key"]; ok {
		t.Fatalf("borrowed agent_key persisted: %#v", stored)
	}
	fingerprint, _ := stored["secret_fingerprint"].(string)
	if !strings.HasPrefix(fingerprint, "sha256:") || len(fingerprint) != len("sha256:0123456789abcdef") {
		t.Fatalf("secret_fingerprint = %q, want sha256:16hex", fingerprint)
	}
}

func TestCredentialPoolBorrowedBoundaryPreservesOwnedAndManualCredentials(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		source   string
	}{
		{name: "manual", provider: "openrouter", source: "manual"},
		{name: "manual scoped", provider: "openrouter", source: "manual:rotated"},
		{name: "nous device code", provider: NousOAuthProvider, source: NousOAuthDeviceCodeSource},
		{name: "codex device code", provider: "openai-codex", source: "device-code"},
		{name: "codex cli import", provider: "openai-codex", source: "codex-cli-import"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			if err := SaveCredentialPoolEntries(CredentialPoolOptions{HermesHome: home, Provider: tc.provider}, []PooledCredential{{
				ID:           "owned",
				Label:        "Owned",
				AuthType:     CredentialAuthOAuth,
				Source:       tc.source,
				AccessToken:  "owned-access-token",
				RefreshToken: "owned-refresh-token",
				AgentKey:     "owned-agent-key",
			}}); err != nil {
				t.Fatalf("SaveCredentialPoolEntries: %v", err)
			}
			stored := readCredentialPoolEntry(t, home, tc.provider, 0)
			if stored["access_token"] != "owned-access-token" || stored["refresh_token"] != "owned-refresh-token" || stored["agent_key"] != "owned-agent-key" {
				t.Fatalf("owned/manual credential fields were not preserved: %#v", stored)
			}
			if _, ok := stored["secret_fingerprint"]; ok {
				t.Fatalf("owned/manual entry unexpectedly fingerprinted: %#v", stored)
			}
		})
	}
}

func TestCredentialPoolUnknownNonEmptySourceIsBorrowedByDefault(t *testing.T) {
	home := t.TempDir()
	if err := SaveCredentialPoolEntries(CredentialPoolOptions{HermesHome: home, Provider: "openrouter"}, []PooledCredential{{
		ID:          "future",
		Label:       "Future source",
		AuthType:    CredentialAuthAPIKey,
		Source:      "future-secret-manager",
		AccessToken: "future-secret-token",
	}}); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}
	stored := readCredentialPoolEntry(t, home, "openrouter", 0)
	if _, ok := stored["access_token"]; ok {
		t.Fatalf("unknown borrowed source persisted access_token: %#v", stored)
	}
	if got, _ := stored["secret_fingerprint"].(string); !strings.HasPrefix(got, "sha256:") {
		t.Fatalf("unknown borrowed source fingerprint = %q", got)
	}
}

func TestCredentialPoolWriteBoundarySanitizesBorrowedEntries(t *testing.T) {
	home := t.TempDir()
	store := credentialPoolAuthStore{CredentialPool: map[string][]PooledCredential{
		"openrouter": {{ID: "raw", Label: "Raw", AuthType: CredentialAuthAPIKey, Source: "env:OPENROUTER_API_KEY", AccessToken: "raw-env-token"}},
	}}
	if err := writeCredentialPoolAuthStore(home, store); err != nil {
		t.Fatalf("writeCredentialPoolAuthStore: %v", err)
	}
	raw := readRawAuthJSON(t, home)
	if strings.Contains(raw, "raw-env-token") {
		t.Fatalf("write boundary leaked borrowed token: %s", raw)
	}
	stored := readCredentialPoolEntry(t, home, "openrouter", 0)
	if stored["source"] != "env:OPENROUTER_API_KEY" {
		t.Fatalf("source metadata not preserved: %#v", stored)
	}
	if got, _ := stored["secret_fingerprint"].(string); !strings.HasPrefix(got, "sha256:") {
		t.Fatalf("write boundary fingerprint = %q", got)
	}
}

func readRawAuthJSON(t *testing.T, home string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if err != nil {
		t.Fatalf("read auth.json: %v", err)
	}
	return string(body)
}

func readCredentialPoolEntry(t *testing.T, home, provider string, index int) map[string]any {
	t.Helper()
	var doc struct {
		CredentialPool map[string][]map[string]any `json:"credential_pool"`
	}
	if err := json.Unmarshal([]byte(readRawAuthJSON(t, home)), &doc); err != nil {
		t.Fatalf("unmarshal auth.json: %v", err)
	}
	entries := doc.CredentialPool[provider]
	if len(entries) <= index {
		t.Fatalf("credential_pool[%s] has %d entries, want index %d", provider, len(entries), index)
	}
	return entries[index]
}
