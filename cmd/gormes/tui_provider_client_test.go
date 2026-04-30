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
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)

	var providerRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerRequests++
		http.Error(w, "provider should not be contacted during TUI startup", http.StatusForbidden)
	}))
	defer server.Close()

	store := config.NewCodexOAuthStateStore(config.CodexOAuthStateStoreOptions{HermesHome: gormesHome})
	if _, err := store.SaveTokens(config.CodexOAuthTokens{
		AccountID:    "acct-tui",
		Label:        "TUI Account",
		AccessToken:  "tui-pool-access-token",
		RefreshToken: "tui-pool-refresh-token",
		BaseURL:      server.URL,
	}); err != nil {
		t.Fatalf("SaveTokens: %v", err)
	}

	var programRuns int
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
			return fakeTUIProgram{run: func() { programRuns++ }}
		},
	})
	if err != nil {
		t.Fatalf("runResolvedTUIWithRuntime: %v", err)
	}
	if programRuns != 1 {
		t.Fatalf("programRuns = %d, want 1", programRuns)
	}
	if providerRequests != 0 {
		t.Fatalf("providerRequests = %d, want 0 before the first submitted turn", providerRequests)
	}
}

func TestTUIStartupDoesNotProbeProviderHealth(t *testing.T) {
	setupNativeTUITestEnv(t)

	var providerRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerRequests++
		http.Error(w, "provider should not be contacted during TUI startup", http.StatusForbidden)
	}))
	defer server.Close()

	var programRuns int
	err := runResolvedTUIWithRuntime(newRootCommand(), tuiInvocation{
		Inference: config.InferenceResolution{
			Model:    "gpt-5.5",
			Provider: "openai",
		},
		Config: config.Config{Hermes: config.HermesCfg{
			Endpoint: server.URL,
			APIKey:   "test-api-key",
			Model:    "gpt-5.5",
			Provider: "openai",
		}},
	}, rootRuntime{
		tuiProgramFactory: func(_ tea.Model, _ ...tea.ProgramOption) tuiProgram {
			return fakeTUIProgram{run: func() { programRuns++ }}
		},
	})
	if err != nil {
		t.Fatalf("runResolvedTUIWithRuntime: %v", err)
	}
	if programRuns != 1 {
		t.Fatalf("programRuns = %d, want 1", programRuns)
	}
	if providerRequests != 0 {
		t.Fatalf("providerRequests = %d, want 0 before the first submitted turn", providerRequests)
	}
}
