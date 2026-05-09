package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// TestRemoteTUI_FlagThreadsThroughRunResolvedTUI proves the root command
// reads --remote into the resolved invocation so downstream wiring can
// branch on it. The fixture intercepts runResolvedTUI and asserts the
// remote URL is present without calling the real TUI runtime.
func TestRemoteTUI_FlagThreadsThroughRunResolvedTUI(t *testing.T) {
	setupNativeTUITestEnv(t)

	var seen tuiInvocation
	cmd := newRootCommandWithRuntime(rootRuntime{
		runResolvedTUI: func(_ *cobra.Command, invocation tuiInvocation) error {
			seen = invocation
			return nil
		},
	})
	stdout, stderr, err := executeNativeTUICommand(cmd, "--offline", "--remote", "http://gateway.example/")
	if err != nil {
		t.Fatalf("Execute() err=%v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if seen.RemoteURL != "http://gateway.example/" {
		t.Fatalf("invocation.RemoteURL = %q; want http://gateway.example/", seen.RemoteURL)
	}
}

// TestRemoteTUI_StartupBypassesAPIServerHealthAndPackageManagers proves
// that --remote <url> skips local provider health probes and never spawns
// node/npm/python — the remote SSE consumer is pure Go HTTP. Local
// Bubble Tea continues to run and the program factory is invoked once.
func TestRemoteTUI_StartupBypassesAPIServerHealthAndPackageManagers(t *testing.T) {
	setupNativeTUITestEnv(t)

	var healthHits atomic.Int32
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			healthHits.Add(1)
		}
		if r.URL.Path == "/events" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher := w.(http.Flusher)
			f := kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1, Model: "remote-fixture"}
			data, _ := json.Marshal(f)
			fmt.Fprintf(w, "event: frame\ndata: %s\n\n", data)
			flusher.Flush()
			<-r.Context().Done()
			return
		}
		http.NotFound(w, r)
	}))
	defer gateway.Close()

	workDir := t.TempDir()
	t.Chdir(workDir)
	commandLog := filepath.Join(workDir, "unexpected-package-command.log")
	installFailingPackageCommands(t, commandLog)

	var programRuns atomic.Int32
	cmd := newRootCommandWithRuntime(rootRuntime{
		tuiProgramFactory: func(tea.Model, ...tea.ProgramOption) tuiProgram {
			return fakeTUIProgram{run: func() { programRuns.Add(1) }}
		},
	})
	stdout, stderr, err := executeNativeTUICommand(cmd, "--remote", gateway.URL)
	if err != nil {
		t.Fatalf("Execute() err=%v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if programRuns.Load() != 1 {
		t.Fatalf("programRuns = %d; want 1", programRuns.Load())
	}
	if healthHits.Load() != 0 {
		t.Errorf("healthHits = %d; want 0 (remote mode bypasses local provider health)", healthHits.Load())
	}
	if data, err := os.ReadFile(commandLog); err == nil {
		t.Fatalf("--remote startup invoked package command unexpectedly:\n%s", data)
	}
	if strings.Contains(stderr, "api_server not reachable") {
		t.Fatalf("stderr surfaced obsolete API-server health error in remote mode:\n%s", stderr)
	}
}

func TestRemoteTUI_EnvGatewayURLSelectsWebSocketAttach(t *testing.T) {
	setupNativeTUITestEnv(t)

	var sessionCreates atomic.Int32
	upgrader := websocket.Upgrader{}
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/events" {
			http.Error(w, "SSE endpoint must not be used for websocket attach", http.StatusTeapot)
			return
		}
		if r.URL.Path != "/api/ws" {
			http.NotFound(w, r)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		for {
			var req struct {
				ID     string `json:"id"`
				Method string `json:"method"`
			}
			if err := conn.ReadJSON(&req); err != nil {
				return
			}
			if req.Method == "session.create" {
				sessionCreates.Add(1)
				_ = conn.WriteJSON(map[string]any{
					"jsonrpc": "2.0",
					"id":      req.ID,
					"result":  map[string]any{"session_id": "sid-env"},
				})
				continue
			}
			_ = conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{"status": "ok"}})
		}
	}))
	defer gateway.Close()
	t.Setenv("HERMES_TUI_GATEWAY_URL", "ws"+strings.TrimPrefix(gateway.URL, "http")+"/api/ws?token=secret")

	workDir := t.TempDir()
	t.Chdir(workDir)
	commandLog := filepath.Join(workDir, "unexpected-package-command.log")
	installFailingPackageCommands(t, commandLog)

	var programRuns atomic.Int32
	cmd := newRootCommandWithRuntime(rootRuntime{
		tuiProgramFactory: func(tea.Model, ...tea.ProgramOption) tuiProgram {
			return fakeTUIProgram{run: func() { programRuns.Add(1) }}
		},
	})
	stdout, stderr, err := executeNativeTUICommand(cmd)
	if err != nil {
		t.Fatalf("Execute() err=%v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if programRuns.Load() != 1 {
		t.Fatalf("programRuns = %d; want 1", programRuns.Load())
	}
	if sessionCreates.Load() != 1 {
		t.Fatalf("sessionCreates = %d; want 1 websocket session.create", sessionCreates.Load())
	}
	if data, err := os.ReadFile(commandLog); err == nil {
		t.Fatalf("websocket attach startup invoked package command unexpectedly:\n%s", data)
	}
}

// TestRemoteTUI_DoctorTUIStatusReportsRemoteDegradedMode confirms the
// degraded-mode evidence: when the operator is running purely in local
// Bubble Tea mode (no --remote), the doctor TUI status flags remote
// streaming as unavailable while still reporting the local runtime as
// healthy. This matches the row's degraded_mode contract: TUI status
// reports remote streaming unavailable while local Bubble Tea continues
// to work.
func TestRemoteTUI_DoctorTUIStatusReportsRemoteDegradedMode(t *testing.T) {
	got := doctorTUIStatus().Format()
	lower := strings.ToLower(got)
	for _, want := range []string{"native tui", "go-native bubble tea", "remote", "websocket"} {
		if !strings.Contains(lower, want) {
			t.Errorf("doctor TUI status missing %q:\n%s", want, got)
		}
	}
}
