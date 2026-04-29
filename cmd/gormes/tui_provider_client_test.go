package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestTUIUsesCodexCredentialPoolWhenEndpointEmpty(t *testing.T) {
	setupNativeTUITestEnv(t)
	hermesHome := t.TempDir()
	t.Setenv("HERMES_HOME", hermesHome)

	var sawHealth bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("request path = %q, want /health", r.URL.Path)
		}
		sawHealth = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := config.NewCodexOAuthStateStore(config.CodexOAuthStateStoreOptions{HermesHome: hermesHome})
	if _, err := store.SaveTokens(config.CodexOAuthTokens{
		AccountID:    "acct-tui",
		Label:        "TUI Account",
		AccessToken:  "tui-pool-access-token",
		RefreshToken: "tui-pool-refresh-token",
		BaseURL:      server.URL,
	}); err != nil {
		t.Fatalf("SaveTokens: %v", err)
	}

	err := runResolvedTUIWithRuntime(newRootCommand(), tuiInvocation{
		Inference: config.InferenceResolution{
			Model:    "gpt-5.5",
			Provider: "openai-codex",
		},
		Config: config.Config{Hermes: config.HermesCfg{
			Model:    "gpt-5.5",
			Provider: "openai-codex",
		}},
	}, rootRuntime{
		tuiProgramFactory: func(_ tea.Model, _ ...tea.ProgramOption) tuiProgram {
			return fakeTUIProgram{}
		},
	})
	if err != nil {
		t.Fatalf("runResolvedTUIWithRuntime: %v", err)
	}
	if !sawHealth {
		t.Fatal("TUI startup did not call credential-pool backed Codex endpoint")
	}
}
