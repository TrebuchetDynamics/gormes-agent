package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
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

func TestTUIStartupMissingProviderRendersFirstRunGuidance(t *testing.T) {
	setupNativeTUITestEnv(t)

	cmd := newRootCommand()
	stdout, stderr, err := executeNativeTUICommand(cmd)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil first-run guidance\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Gormes setup needed",
		"not ready: provider endpoint is not configured",
		"Provider: provider endpoint is not configured",
		"Authentication: provider credential is not configured",
		"Next: gormes setup --quick --target terminal",
		"Non-interactive mode will not prompt.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{
		"Gormes provider setup needed",
		"provider setup failed:",
		"hermes endpoint unconfigured",
		"gormes --offline",
	} {
		if strings.Contains(stderr, forbidden) {
			t.Fatalf("stderr contains old provider setup text %q:\n%s", forbidden, stderr)
		}
	}
	if count := strings.Count(stdout, "Gormes setup needed"); count != 1 {
		t.Fatalf("first-run setup heading count = %d, want 1:\n%s", count, stdout)
	}
}

func TestTUIStartupFallsBackWhenSessionDBLocked(t *testing.T) {
	setupNativeTUITestEnv(t)

	locked, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		t.Fatalf("lock session DB: %v", err)
	}
	defer locked.Close()

	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline: %v", err)
	}
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	var rendered string
	err = runResolvedTUIWithRuntime(cmd, tuiInvocation{
		Config: config.Config{Hermes: config.HermesCfg{Model: "fixture-model"}},
	}, rootRuntime{
		tuiProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) tuiProgram {
			return scriptedTUIProgram{run: func() {
				current := model
				current, _ = current.Update(tea.WindowSizeMsg{Width: 120, Height: 32})
				rendered = current.View()
			}}
		},
	})
	if err != nil {
		t.Fatalf("runResolvedTUIWithRuntime: %v", err)
	}
	if strings.Contains(stderr.String(), "session persistence unavailable") ||
		strings.Contains(stderr.String(), "running TUI with in-memory session state") {
		t.Fatalf("locked sessions.db warning should render inside Bubble Tea, not stderr:\n%s", stderr.String())
	}
	for _, want := range []string{
		"session state: in-memory",
		"sessions.db locked",
		"gateway status/stop",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered TUI missing %q:\n%s", want, rendered)
		}
	}
}

func TestTUIStartupSelfHealsCorruptSessionDB(t *testing.T) {
	setupNativeTUITestEnv(t)

	if err := os.MkdirAll(filepath.Dir(config.SessionDBPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.SessionDBPath(), []byte("not bolt"), 0o600); err != nil {
		t.Fatalf("seed corrupt session DB: %v", err)
	}

	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline: %v", err)
	}
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	var programRuns int
	err := runResolvedTUIWithRuntime(cmd, tuiInvocation{
		Config: config.Config{Hermes: config.HermesCfg{Model: "fixture-model"}},
	}, rootRuntime{
		tuiProgramFactory: func(_ tea.Model, _ ...tea.ProgramOption) tuiProgram {
			return fakeTUIProgram{run: func() { programRuns++ }}
		},
	})
	if err != nil {
		t.Fatalf("runResolvedTUIWithRuntime must self-heal corrupt sessions.db: %v\nstderr=%s", err, stderr.String())
	}
	if programRuns != 1 {
		t.Fatalf("programRuns = %d, want 1", programRuns)
	}
	if !strings.Contains(stderr.String(), "session persistence self-healed") {
		t.Fatalf("stderr must mention session self-heal; got:\n%s", stderr.String())
	}
	backups, err := filepath.Glob(config.SessionDBPath() + ".corrupt-*")
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("corrupt sessions.db must be preserved as one quarantine backup, got %v", backups)
	}
}
