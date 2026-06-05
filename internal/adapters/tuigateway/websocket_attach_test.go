package tuigateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestDialWebSocketAttach_CreatesSessionAndConsumesFrames(t *testing.T) {
	t.Parallel()

	var eventsHits int
	var methodsMu sync.Mutex
	var methods []string
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/events" {
			eventsHits++
			http.Error(w, "sse must not be used", http.StatusTeapot)
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
		var req websocketRequest
		if err := conn.ReadJSON(&req); err != nil {
			t.Errorf("read session.create: %v", err)
			return
		}
		methodsMu.Lock()
		methods = append(methods, req.Method)
		methodsMu.Unlock()
		if req.Method != "session.create" {
			t.Errorf("first method = %q; want session.create", req.Method)
		}
		if err := conn.WriteJSON(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  map[string]any{"session_id": "sid-ws"},
		}); err != nil {
			t.Errorf("write session.create response: %v", err)
			return
		}
		frame := kernel.RenderFrame{Phase: kernel.PhaseStreaming, Seq: 7, DraftText: "from websocket"}
		if err := conn.WriteJSON(map[string]any{
			"jsonrpc": "2.0",
			"method":  "event",
			"params":  map[string]any{"type": "frame", "payload": frame},
		}); err != nil {
			t.Errorf("write event frame: %v", err)
			return
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := DialWebSocketAttach(ctx, websocketURL(server.URL, "/api/ws"))
	if err != nil {
		t.Fatalf("DialWebSocketAttach: %v", err)
	}
	defer client.Close()

	select {
	case frame := <-client.Frames():
		if frame.Seq != 7 || frame.DraftText != "from websocket" || frame.Phase != kernel.PhaseStreaming {
			t.Fatalf("frame = %+v; want websocket RenderFrame", frame)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for websocket frame")
	}

	methodsMu.Lock()
	defer methodsMu.Unlock()
	if len(methods) != 1 || methods[0] != "session.create" {
		t.Fatalf("methods = %v; want only session.create", methods)
	}
	if eventsHits != 0 {
		t.Fatalf("/events hits = %d; want 0", eventsHits)
	}
}

func TestWebSocketAttach_SubmitCancelResize(t *testing.T) {
	t.Parallel()

	var methodsMu sync.Mutex
	var methods []string
	var params []map[string]any
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		for {
			var req websocketRequest
			if err := conn.ReadJSON(&req); err != nil {
				return
			}
			methodsMu.Lock()
			methods = append(methods, req.Method)
			var rawParams map[string]any
			_ = json.Unmarshal(req.Params, &rawParams)
			params = append(params, rawParams)
			methodsMu.Unlock()
			result := map[string]any{"status": "ok"}
			if req.Method == "session.create" {
				result = map[string]any{"session_id": "sid-ws"}
			}
			if err := conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := DialWebSocketAttach(ctx, websocketURL(server.URL, "/api/ws"))
	if err != nil {
		t.Fatalf("DialWebSocketAttach: %v", err)
	}
	defer client.Close()

	if err := client.Submit(ctx, "hello"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if err := client.Cancel(ctx); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if err := client.Resize(ctx, 132); err != nil {
		t.Fatalf("Resize: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for {
		methodsMu.Lock()
		done := len(methods) >= 4
		methodsMu.Unlock()
		if done {
			break
		}
		select {
		case <-deadline:
			methodsMu.Lock()
			t.Fatalf("methods = %v; params = %#v", methods, params)
			methodsMu.Unlock()
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	methodsMu.Lock()
	defer methodsMu.Unlock()
	want := []string{"session.create", "prompt.submit", "session.interrupt", "terminal.resize"}
	if strings.Join(methods[:4], ",") != strings.Join(want, ",") {
		t.Fatalf("methods = %v; want %v", methods[:4], want)
	}
	if params[1]["session_id"] != "sid-ws" || params[1]["text"] != "hello" {
		t.Fatalf("prompt.submit params = %#v; want sid-ws + hello", params[1])
	}
	if params[2]["session_id"] != "sid-ws" {
		t.Fatalf("session.interrupt params = %#v; want sid-ws", params[2])
	}
	if params[3]["session_id"] != "sid-ws" || int(params[3]["cols"].(float64)) != 132 {
		t.Fatalf("terminal.resize params = %#v; want sid-ws cols 132", params[3])
	}
}

func TestWebSocketAttach_MirrorsEventFramesToSidecar(t *testing.T) {
	t.Parallel()

	sidecarReady := make(chan struct{})
	sidecarSaw := make(chan string, 1)
	upgrader := websocket.Upgrader{}
	sidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("sidecar upgrade: %v", err)
			return
		}
		defer conn.Close()
		close(sidecarReady)
		_, payload, err := conn.ReadMessage()
		if err == nil {
			sidecarSaw <- string(payload)
		}
	}))
	defer sidecar.Close()

	mainServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("main upgrade: %v", err)
			return
		}
		defer conn.Close()
		var req websocketRequest
		if err := conn.ReadJSON(&req); err != nil {
			t.Errorf("read session.create: %v", err)
			return
		}
		if err := conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{"session_id": "sid-ws"}}); err != nil {
			return
		}
		select {
		case <-sidecarReady:
		case <-time.After(2 * time.Second):
			t.Error("sidecar was not connected before event")
			return
		}
		frame := kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 9}
		_ = conn.WriteJSON(map[string]any{
			"jsonrpc": "2.0",
			"method":  "event",
			"params":  map[string]any{"type": "frame", "payload": frame},
		})
		<-r.Context().Done()
	}))
	defer mainServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := DialWebSocketAttach(ctx, websocketURL(mainServer.URL, ""), WithSidecarURL(websocketURL(sidecar.URL, "")))
	if err != nil {
		t.Fatalf("DialWebSocketAttach: %v", err)
	}
	defer client.Close()

	select {
	case raw := <-sidecarSaw:
		if !strings.Contains(raw, `"method":"event"`) || !strings.Contains(raw, `"Seq":9`) {
			t.Fatalf("sidecar raw frame = %s; want mirrored event with seq 9", raw)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for sidecar mirror")
	}
}

func TestRedactRemoteURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "query",
			in:   "ws://gateway.test/api/ws?token=hunter2&channel=secret",
			want: "ws://gateway.test/api/ws?***",
		},
		{
			name: "userinfo",
			in:   "wss://alice:hunter2@gateway.test/api/ws",
			want: "wss://***@gateway.test/api/ws",
		},
		{
			name: "malformed query and userinfo",
			in:   "ws://alice:hunter2@gateway.test:99999/api/ws?token=secret",
			want: "ws://***@gateway.test:99999/api/ws?***",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RedactRemoteURL(tt.in); got != tt.want {
				t.Fatalf("RedactRemoteURL(%q) = %q; want %q", tt.in, got, tt.want)
			}
		})
	}
}

func websocketURL(base, path string) string {
	u := "ws" + strings.TrimPrefix(base, "http")
	if path == "" {
		return u
	}
	return strings.TrimRight(u, "/") + path
}
