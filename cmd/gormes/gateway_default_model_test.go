package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func TestGatewayBoot_OpenAICodexProvider_DefaultsModel(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("CODEX_HOME", filepath.Join(root, "codex"))
	t.Setenv("GORMES_ENDPOINT", "")
	t.Setenv("GORMES_MODEL", "")
	t.Setenv("GORMES_API_KEY", "")
	if err := os.MkdirAll(config.GormesHome(), 0o755); err != nil {
		t.Fatalf("create GORMES_HOME: %v", err)
	}
	if err := os.MkdirAll(os.Getenv("CODEX_HOME"), 0o755); err != nil {
		t.Fatalf("create CODEX_HOME: %v", err)
	}

	var sawModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		sawModel = payload.Model
		if sawModel == "hermes-agent" {
			t.Fatalf("provider request model = hermes-agent, want provider default")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"completed","output_text":"ok"}`))
	}))
	defer server.Close()

	prev := authCodexOAuthLogin
	authCodexOAuthLogin = func(context.Context, codexOAuthLoginRequest) (config.CodexOAuthTokens, error) {
		return config.CodexOAuthTokens{
			AccountID:    "acct",
			Label:        "Codex",
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			BaseURL:      server.URL,
		}, nil
	}
	t.Cleanup(func() { authCodexOAuthLogin = prev })

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "auth", "add", "openai-codex", "--type", "oauth", "--inference-url", server.URL)
	if err != nil {
		t.Fatalf("auth add: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load(nil): %v", err)
	}
	if cfg.Hermes.Model != "gpt-5.5" {
		t.Fatalf("cfg.Hermes.Model = %q, want provider default", cfg.Hermes.Model)
	}
	client, err := gormescli.NewProviderHTTPClient(cfg, cfg.Hermes.Provider)
	if err != nil {
		t.Fatalf("newProviderHTTPClient: %v", err)
	}
	stream, err := client.OpenStream(context.Background(), llm.ChatRequest{
		Model:    cfg.Hermes.Model,
		Messages: []llm.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer stream.Close()
	if _, err := stream.Recv(context.Background()); err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if sawModel != "gpt-5.5" {
		t.Fatalf("provider request model = %q, want gpt-5.5", sawModel)
	}
}
