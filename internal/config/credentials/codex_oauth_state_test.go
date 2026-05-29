package credentials

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCodexOAuthStateStore_PersistsGormesOwnedCredential(t *testing.T) {
	home := t.TempDir()
	store := NewCodexOAuthStateStore(CodexOAuthStateStoreOptions{
		HermesHome: home,
		Now:        func() time.Time { return time.Unix(1_775_000_000, 0).UTC() },
	})

	status, err := store.SaveTokens(CodexOAuthTokens{
		AccountID:    "acct-work",
		Label:        "Work ChatGPT",
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		BaseURL:      "https://chatgpt.com/backend-api/codex",
		Source:       "device-code",
	})
	if err != nil {
		t.Fatalf("SaveTokens returned error: %v", err)
	}
	if status.Code != CodexOAuthStatusAuthorized || !status.Authenticated {
		t.Fatalf("status = %#v, want authorized authenticated", status)
	}
	if status.AccessToken != "" || status.RefreshToken != "" {
		t.Fatalf("status leaked secrets: %#v", status)
	}

	pool, evidence, err := LoadCredentialPool(CredentialPoolOptions{HermesHome: home, Provider: CodexOAuthProvider})
	if err != nil {
		t.Fatalf("LoadCredentialPool returned error: %v", err)
	}
	if evidence.Code != CredentialPoolEvidenceLoaded {
		t.Fatalf("credential evidence = %#v", evidence)
	}
	entries := pool.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries = %#v, want one", entries)
	}
	entry := entries[0]
	if entry.ID != "acct-work" || entry.Label != "Work ChatGPT" {
		t.Fatalf("entry id/label = %q/%q", entry.ID, entry.Label)
	}
	if entry.AuthType != CredentialAuthOAuth || entry.Source != "device-code" {
		t.Fatalf("entry auth/source = %q/%q", entry.AuthType, entry.Source)
	}
	if entry.AccessToken != "access-token" || entry.RefreshToken != "refresh-token" {
		t.Fatalf("tokens were not persisted in the Gormes auth store")
	}
	if entry.InferenceBaseURL != "https://chatgpt.com/backend-api/codex" {
		t.Fatalf("InferenceBaseURL = %q", entry.InferenceBaseURL)
	}

	raw, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if err != nil {
		t.Fatalf("read auth.json: %v", err)
	}
	if strings.Contains(string(raw), ".codex") {
		t.Fatalf("auth store unexpectedly references Codex CLI state: %s", raw)
	}
}

func TestCodexOAuthStateStore_SaveTokensPreservesOtherAccounts(t *testing.T) {
	home := t.TempDir()
	store := NewCodexOAuthStateStore(CodexOAuthStateStoreOptions{HermesHome: home})

	if _, err := store.SaveTokens(CodexOAuthTokens{
		AccountID:    "acct-personal",
		Label:        "Personal",
		AccessToken:  "personal-access",
		RefreshToken: "personal-refresh",
	}); err != nil {
		t.Fatalf("SaveTokens personal returned error: %v", err)
	}
	if _, err := store.SaveTokens(CodexOAuthTokens{
		AccountID:    "acct-work",
		Label:        "Work",
		AccessToken:  "work-access",
		RefreshToken: "work-refresh",
	}); err != nil {
		t.Fatalf("SaveTokens work returned error: %v", err)
	}

	pool, _, err := LoadCredentialPool(CredentialPoolOptions{HermesHome: home, Provider: CodexOAuthProvider})
	if err != nil {
		t.Fatalf("LoadCredentialPool returned error: %v", err)
	}
	entries := pool.Entries()
	if len(entries) != 2 {
		t.Fatalf("entries = %#v, want two preserved Codex accounts", entries)
	}
	if entries[0].ID != "acct-personal" || entries[1].ID != "acct-work" {
		t.Fatalf("entry IDs = %q/%q, want both accounts preserved", entries[0].ID, entries[1].ID)
	}
}

func TestCodexOAuthStateStore_ExplicitCodexCLIImportRejectsExpiredTokens(t *testing.T) {
	home := t.TempDir()
	now := time.Unix(1_775_000_000, 0).UTC()
	store := NewCodexOAuthStateStore(CodexOAuthStateStoreOptions{
		HermesHome: home,
		Now:        func() time.Time { return now },
	})
	codexAuth := filepath.Join(t.TempDir(), "auth.json")
	writeCodexJSON0600(t, codexAuth, map[string]any{
		"tokens": map[string]any{
			"access_token":  unsignedJWT(t, now.Add(-time.Minute)),
			"refresh_token": "stale-refresh",
		},
	})

	status, err := store.ImportCodexCLITokens(CodexCLIImportRequest{
		AuthPath: codexAuth,
		Explicit: true,
	})
	if err != nil {
		t.Fatalf("ImportCodexCLITokens returned error: %v", err)
	}
	if status.Code != CodexOAuthStatusImportRejected || status.Evidence != CodexOAuthEvidenceImportExpired {
		t.Fatalf("status = %#v, want expired import rejection", status)
	}
	pool, _, err := LoadCredentialPool(CredentialPoolOptions{HermesHome: home, Provider: CodexOAuthProvider})
	if err != nil {
		t.Fatalf("LoadCredentialPool returned error: %v", err)
	}
	if entries := pool.Entries(); len(entries) != 0 {
		t.Fatalf("expired Codex CLI tokens were imported: %#v", entries)
	}

	writeCodexJSON0600(t, codexAuth, map[string]any{
		"tokens": map[string]any{
			"access_token":  unsignedJWT(t, now.Add(time.Hour)),
			"refresh_token": "fresh-refresh",
		},
	})
	status, err = store.ImportCodexCLITokens(CodexCLIImportRequest{
		AuthPath: codexAuth,
		Explicit: true,
	})
	if err != nil {
		t.Fatalf("ImportCodexCLITokens fresh returned error: %v", err)
	}
	if status.Code != CodexOAuthStatusAuthorized || status.Source != CodexOAuthSourceCodexCLIImport {
		t.Fatalf("fresh import status = %#v", status)
	}
}

func TestCodexOAuthStateStore_DoesNotReadCodexCLIWithoutExplicitImport(t *testing.T) {
	store := NewCodexOAuthStateStore(CodexOAuthStateStoreOptions{HermesHome: t.TempDir()})
	status, err := store.ImportCodexCLITokens(CodexCLIImportRequest{})
	if err != nil {
		t.Fatalf("ImportCodexCLITokens returned error: %v", err)
	}
	if status.Code != CodexOAuthStatusImportNotRequested {
		t.Fatalf("status = %#v, want import_not_requested", status)
	}
}

func writeCodexJSON0600(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func unsignedJWT(t *testing.T, exp time.Time) string {
	t.Helper()
	encode := base64.RawURLEncoding.EncodeToString
	header := encode([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := encode([]byte(`{"exp":` + strconv.FormatInt(exp.Unix(), 10) + `}`))
	return header + "." + payload + "."
}
