package gormescli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestRunSetupProviderLiveTestCodexUsesAuthReadinessNotGenericHealth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "")

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		http.Error(w, "generic health endpoint should not be called for Codex", http.StatusNotFound)
	}))
	defer server.Close()

	writeSetupFirstLiveTestConfig(t, server.URL)
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: config.CodexOAuthProvider}, []config.PooledCredential{{
		ID:               "codex-import",
		Label:            "Imported Codex CLI",
		AuthType:         config.CredentialAuthOAuth,
		Source:           config.CodexOAuthSourceCodexCLIImport,
		AccessToken:      "codex-access-secret",
		RefreshToken:     "codex-refresh-secret",
		InferenceBaseURL: server.URL,
		LastStatus:       config.CredentialStatusOK,
	}}); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}

	if err := RunSetupProviderLiveTest(context.Background()); err != nil {
		t.Fatalf("RunSetupProviderLiveTest: %v", err)
	}
	if called {
		t.Fatalf("RunSetupProviderLiveTest called generic /health endpoint for Codex")
	}
}

func TestRunSetupProviderLiveTestCodexMissingAuthFailsBeforeGenericHealth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "")

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	writeSetupFirstLiveTestConfig(t, server.URL)

	err := RunSetupProviderLiveTest(context.Background())
	if err == nil {
		t.Fatal("RunSetupProviderLiveTest error = nil, want missing auth error")
	}
	if !strings.Contains(err.Error(), "openai-codex credential unavailable") {
		t.Fatalf("RunSetupProviderLiveTest error = %v, want openai-codex credential unavailable", err)
	}
	if called {
		t.Fatalf("RunSetupProviderLiveTest called generic /health endpoint despite missing Codex auth")
	}
}

func writeSetupFirstLiveTestConfig(t *testing.T, endpoint string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	contents := "[hermes]\nprovider = '" + config.CodexOAuthProvider + "'\nendpoint = '" + endpoint + "'\nmodel = 'gpt-5.5'\n"
	if err := os.WriteFile(config.ConfigPath(), []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
