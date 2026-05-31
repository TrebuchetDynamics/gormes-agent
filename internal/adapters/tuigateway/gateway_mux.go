package tuigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/internal/httpjson"
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/internal/httpstream"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// KernelHandle is the dispatch contract the GatewayMux uses to deliver
// PlatformEvents into a kernel.Kernel (or an equivalent test fake).
// Keeping the surface narrow means the gateway transport never touches
// kernel internals — it only forwards events and reads render frames.
type KernelHandle interface {
	Submit(kernel.PlatformEvent) error
	Render() <-chan kernel.RenderFrame
}

// GatewayMux is the native Go HTTP front for the remote TUI: it streams
// kernel.RenderFrames as SSE on GET /events and accepts JSON-encoded
// platform events on the dedicated /submit, /cancel, /resize endpoints
// plus a unified /platform-event dispatcher for clients that prefer one
// envelope. The mux mirrors the upstream tui_gateway/server.py JSON-RPC
// methods (prompt.submit, session.interrupt, terminal.resize) but speaks
// plain HTTP+SSE so the consumer can be any Go process.
type GatewayMux struct {
	*http.ServeMux
	handle KernelHandle

	mu            sync.RWMutex
	lastResize    map[string]int
	nextWSSession atomic.Uint64
}

// NewGatewayMux returns a multiplexer wired to the supplied kernel
// handle. The returned mux embeds an *http.ServeMux so callers can
// compose additional routes (health, metrics) onto the same listener.
func NewGatewayMux(handle KernelHandle) *GatewayMux {
	g := &GatewayMux{
		ServeMux:   http.NewServeMux(),
		handle:     handle,
		lastResize: make(map[string]int),
	}
	g.HandleFunc("GET /events", g.handleEvents)
	g.HandleFunc("POST /submit", g.handleSubmit)
	g.HandleFunc("POST /cancel", g.handleCancel)
	g.HandleFunc("POST /resize", g.handleResize)
	g.HandleFunc("POST /platform-event", g.handlePlatformEvent)
	g.HandleFunc("GET /api/ws", g.handleWebSocket)
	return g
}

// LastResizeCols returns the most recent column count POSTed to /resize
// for the given session id. Tests use it to assert the resize handler
// recorded the value; production callers can also surface it for
// diagnostics. Returns 0 when no resize has been observed.
func (g *GatewayMux) LastResizeCols(sessionID string) int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastResize[sessionID]
}

func (g *GatewayMux) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, -32603, "streaming unsupported")
		return
	}
	httpstream.SetSSEHeaders(w)
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	frames := g.handle.Render()
	for {
		select {
		case <-r.Context().Done():
			return
		case f, ok := <-frames:
			if !ok {
				return
			}
			if err := httpstream.WriteEvent(w, "frame", f); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (g *GatewayMux) handleSubmit(w http.ResponseWriter, r *http.Request) {
	var evt SubmitEvent
	if err := decodeJSONBody(r, &evt); err != nil {
		writeJSONError(w, http.StatusBadRequest, -32700, "parse error")
		return
	}
	if err := g.handle.Submit(kernel.PlatformEvent{
		Kind:      kernel.PlatformEventSubmit,
		SessionID: evt.SessionID,
		Text:      evt.Text,
	}); err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, -32000, err.Error())
		return
	}
	writeJSONOK(w, map[string]any{"status": "queued"})
}

func (g *GatewayMux) handleCancel(w http.ResponseWriter, r *http.Request) {
	var evt CancelEvent
	if err := decodeJSONBody(r, &evt); err != nil {
		writeJSONError(w, http.StatusBadRequest, -32700, "parse error")
		return
	}
	if err := g.handle.Submit(kernel.PlatformEvent{
		Kind:      kernel.PlatformEventCancel,
		SessionID: evt.SessionID,
	}); err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, -32000, err.Error())
		return
	}
	writeJSONOK(w, map[string]any{"status": "ok"})
}

func (g *GatewayMux) handleResize(w http.ResponseWriter, r *http.Request) {
	var evt ResizeEvent
	if err := decodeJSONBody(r, &evt); err != nil {
		writeJSONError(w, http.StatusBadRequest, -32700, "parse error")
		return
	}
	g.mu.Lock()
	g.lastResize[evt.SessionID] = evt.Cols
	g.mu.Unlock()
	writeJSONOK(w, map[string]any{"cols": evt.Cols})
}

// handlePlatformEvent dispatches the unified envelope. It first decodes
// just the kind discriminator, then re-decodes into the concrete struct
// so each variant goes through the same handler as its dedicated
// endpoint. Unknown kinds get a 400 with a JSON-RPC error envelope.
func (g *GatewayMux) handlePlatformEvent(w http.ResponseWriter, r *http.Request) {
	body, err := readBodyOnce(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, -32700, "parse error")
		return
	}
	var head struct {
		Kind PlatformEventKind `json:"kind"`
	}
	if err := json.Unmarshal(body, &head); err != nil {
		writeJSONError(w, http.StatusBadRequest, -32700, "parse error")
		return
	}
	if !ValidPlatformEventKind(head.Kind) {
		writeJSONError(w, http.StatusBadRequest, -32602, fmt.Sprintf("unknown platform event kind: %q", head.Kind))
		return
	}
	switch head.Kind {
	case PlatformEventKindSubmit:
		var evt SubmitEvent
		if err := json.Unmarshal(body, &evt); err != nil {
			writeJSONError(w, http.StatusBadRequest, -32700, "parse error")
			return
		}
		if err := g.handle.Submit(kernel.PlatformEvent{
			Kind:      kernel.PlatformEventSubmit,
			SessionID: evt.SessionID,
			Text:      evt.Text,
		}); err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, -32000, err.Error())
			return
		}
		writeJSONOK(w, map[string]any{"status": "queued"})
	case PlatformEventKindCancel:
		var evt CancelEvent
		if err := json.Unmarshal(body, &evt); err != nil {
			writeJSONError(w, http.StatusBadRequest, -32700, "parse error")
			return
		}
		if err := g.handle.Submit(kernel.PlatformEvent{
			Kind:      kernel.PlatformEventCancel,
			SessionID: evt.SessionID,
		}); err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, -32000, err.Error())
			return
		}
		writeJSONOK(w, map[string]any{"status": "ok"})
	case PlatformEventKindResize:
		var evt ResizeEvent
		if err := json.Unmarshal(body, &evt); err != nil {
			writeJSONError(w, http.StatusBadRequest, -32700, "parse error")
			return
		}
		g.mu.Lock()
		g.lastResize[evt.SessionID] = evt.Cols
		g.mu.Unlock()
		writeJSONOK(w, map[string]any{"cols": evt.Cols})
	default:
		// Progress and image_metadata are server→client emit-only events;
		// the mux accepts them as valid kinds but does not dispatch them
		// onto the kernel. Acknowledging keeps the wire compatible with
		// future relay use cases without leaking through to the kernel.
		writeJSONOK(w, map[string]any{"status": "ack"})
	}
}

func decodeJSONBody(r *http.Request, dst any) error {
	body, err := readBodyOnce(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dst)
}

func readBodyOnce(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	const maxBody = 1 << 20 // 1 MiB ceiling — generous for a control-plane envelope.
	limited := http.MaxBytesReader(nil, r.Body, maxBody)
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 1024)
	for {
		n, err := limited.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			break
		}
	}
	if len(buf) == 0 {
		return nil, fmt.Errorf("empty body")
	}
	return buf, nil
}

func writeJSONOK(w http.ResponseWriter, payload any) {
	httpjson.Write(w, http.StatusOK, payload)
}

func writeJSONError(w http.ResponseWriter, status, code int, msg string) {
	httpjson.Write(w, status, map[string]any{
		"code":    code,
		"message": msg,
	})
}

var gatewayWSUpgrader = websocket.Upgrader{}

func (g *GatewayMux) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := gatewayWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var writeMu sync.Mutex
	write := func(v any) bool {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(v) == nil
	}

	write(websocketMessage{
		JSONRPC: "2.0",
		Method:  "event",
		Params:  mustRawJSON(map[string]any{"type": "gateway.ready", "payload": map[string]any{"transport": "websocket"}}),
	})

	go func() {
		frames := g.handle.Render()
		for {
			select {
			case <-ctx.Done():
				return
			case frame, ok := <-frames:
				if !ok {
					return
				}
				if !write(websocketMessage{
					JSONRPC: "2.0",
					Method:  "event",
					Params:  mustRawJSON(map[string]any{"type": "frame", "payload": frame}),
				}) {
					cancel()
					return
				}
			}
		}
	}()

	for {
		var req websocketRequest
		if err := conn.ReadJSON(&req); err != nil {
			return
		}
		if !g.dispatchWebSocketRequest(write, req) {
			return
		}
	}
}

func (g *GatewayMux) dispatchWebSocketRequest(write func(any) bool, req websocketRequest) bool {
	switch req.Method {
	case "session.create":
		sessionID := fmt.Sprintf("ws-%d", g.nextWSSession.Add(1))
		return write(websocketMessage{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  mustRawJSON(map[string]any{"session_id": sessionID}),
		})
	case "prompt.submit":
		var params struct {
			SessionID string `json:"session_id"`
			Text      string `json:"text"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return write(websocketMessage{JSONRPC: "2.0", ID: req.ID, Error: &websocketError{Code: -32700, Message: "parse error"}})
		}
		if err := g.handle.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, SessionID: params.SessionID, Text: params.Text}); err != nil {
			return write(websocketMessage{JSONRPC: "2.0", ID: req.ID, Error: &websocketError{Code: -32000, Message: err.Error()}})
		}
		return write(websocketMessage{JSONRPC: "2.0", ID: req.ID, Result: mustRawJSON(map[string]any{"status": "queued"})})
	case "session.interrupt":
		var params struct {
			SessionID string `json:"session_id"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return write(websocketMessage{JSONRPC: "2.0", ID: req.ID, Error: &websocketError{Code: -32700, Message: "parse error"}})
		}
		if err := g.handle.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventCancel, SessionID: params.SessionID}); err != nil {
			return write(websocketMessage{JSONRPC: "2.0", ID: req.ID, Error: &websocketError{Code: -32000, Message: err.Error()}})
		}
		return write(websocketMessage{JSONRPC: "2.0", ID: req.ID, Result: mustRawJSON(map[string]any{"status": "interrupted"})})
	case "terminal.resize":
		var params struct {
			SessionID string `json:"session_id"`
			Cols      int    `json:"cols"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return write(websocketMessage{JSONRPC: "2.0", ID: req.ID, Error: &websocketError{Code: -32700, Message: "parse error"}})
		}
		g.mu.Lock()
		g.lastResize[params.SessionID] = params.Cols
		g.mu.Unlock()
		return write(websocketMessage{JSONRPC: "2.0", ID: req.ID, Result: mustRawJSON(map[string]any{"cols": params.Cols})})
	default:
		return write(websocketMessage{JSONRPC: "2.0", ID: req.ID, Error: &websocketError{Code: -32601, Message: "method not found"}})
	}
}

func mustRawJSON(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}
