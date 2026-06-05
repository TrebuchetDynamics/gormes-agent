package gormescli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestRemoteTUIRootFlagThreadsToRunE(t *testing.T) {
	var seen string
	cmd := NewRootCommand(RootOptions{
		RunE: func(cmd *cobra.Command, _ []string) error {
			seen, _ = cmd.Flags().GetString("remote")
			return nil
		},
	}, stubRootFactories())

	stdout, stderr, err := executeRootCommandForTest(cmd, "--offline", "--remote", "http://gateway.example/")
	if err != nil {
		t.Fatalf("Execute() err=%v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if seen != "http://gateway.example/" {
		t.Fatalf("remote flag = %q; want http://gateway.example/", seen)
	}
}

func TestRunRemoteTUIStartupConsumesSSEWithoutLocalPackageCommands(t *testing.T) {
	var eventsHit atomic.Int32
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/events" {
			http.NotFound(w, r)
			return
		}
		eventsHit.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)
		frame := kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1, Model: "remote-fixture"}
		data, _ := json.Marshal(frame)
		fmt.Fprintf(w, "event: frame\ndata: %s\n\n", data)
		flusher.Flush()
		<-r.Context().Done()
	}))
	defer gateway.Close()

	var runs atomic.Int32
	var stderr bytes.Buffer
	err := RunRemoteTUI(context.Background(), &stderr, RemoteTUIOptions{
		RemoteURL: gateway.URL,
		ModelOptions: func(context.Context) tui.Options {
			return tui.Options{VoiceRecordKey: "ctrl+v"}
		},
		ProgramFactory: func(tea.Model, ...tea.ProgramOption) TUIProgram {
			return remoteTUITestProgram{run: func() { runs.Add(1) }}
		},
	})
	if err != nil {
		t.Fatalf("RunRemoteTUI: %v\nstderr=%s", err, stderr.String())
	}
	if runs.Load() != 1 {
		t.Fatalf("program runs = %d; want 1", runs.Load())
	}
	if eventsHit.Load() != 1 {
		t.Fatalf("eventsHit = %d; want one SSE connection", eventsHit.Load())
	}
	if strings.Contains(stderr.String(), "api_server not reachable") {
		t.Fatalf("stderr surfaced obsolete API-server health error in remote mode:\n%s", stderr.String())
	}
}

func TestRunRemoteTUIWebSocketAttachCreatesSession(t *testing.T) {
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

	var runs atomic.Int32
	remoteURL := "ws" + strings.TrimPrefix(gateway.URL, "http") + "/api/ws?token=secret"
	err := RunRemoteTUI(context.Background(), nil, RemoteTUIOptions{
		RemoteURL: remoteURL,
		ProgramFactory: func(tea.Model, ...tea.ProgramOption) TUIProgram {
			return remoteTUITestProgram{run: func() { runs.Add(1) }}
		},
	})
	if err != nil {
		t.Fatalf("RunRemoteTUI websocket: %v", err)
	}
	if runs.Load() != 1 {
		t.Fatalf("program runs = %d; want 1", runs.Load())
	}
	if sessionCreates.Load() != 1 {
		t.Fatalf("sessionCreates = %d; want 1 websocket session.create", sessionCreates.Load())
	}
}

func TestRunRemoteTUIUnavailableRedactsURL(t *testing.T) {
	var stderr bytes.Buffer
	boom := errors.New("dial refused")
	err := RunRemoteTUI(context.Background(), &stderr, RemoteTUIOptions{
		RemoteURL: "ws://gateway.example/api/ws?token=secret-token",
		Dial: func(context.Context, string, string) (RemoteTUIClient, error) {
			return nil, boom
		},
		ProgramFactory: func(tea.Model, ...tea.ProgramOption) TUIProgram {
			t.Fatal("program factory must not run when dial fails")
			return nil
		},
	})
	if !errors.Is(err, boom) {
		t.Fatalf("RunRemoteTUI err = %v; want %v", err, boom)
	}
	out := stderr.String()
	if !strings.Contains(out, "remote streaming unavailable") || !strings.Contains(out, "without --remote") {
		t.Fatalf("stderr missing remote degraded guidance:\n%s", out)
	}
	if strings.Contains(out, "secret-token") {
		t.Fatalf("stderr leaked remote token:\n%s", out)
	}
}

type remoteTUITestProgram struct {
	run func()
}

func (p remoteTUITestProgram) Run() (tea.Model, error) {
	if p.run != nil {
		p.run()
	}
	return nil, nil
}

func (p remoteTUITestProgram) Quit() {}
