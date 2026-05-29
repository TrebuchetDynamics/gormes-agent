package tuigateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// Default backoff envelope for SSE reconnects. The remote TUI reconnects
// fast on transient drops (the Python tui_gateway does the same in ws.py
// via best-effort retry) but caps the wait so flapping gateways do not
// keep the consumer awake.
const (
	defaultReconnectInitial = 100 * time.Millisecond
	defaultReconnectMax     = 5 * time.Second
)

// RemoteClient consumes SSE-streamed kernel.RenderFrames from a remote
// Gormes gateway and posts platform events back over plain HTTP. It is
// the "transport client" half of the remote TUI surface: callers wire its
// Frames() channel into the Bubble Tea model and its Submit/Cancel/Resize
// helpers into the model's Submitter/Canceller closures.
//
// The struct is safe to share across goroutines — the SSE consumer runs
// on a dedicated worker started by DialSSE; HTTP POST helpers (Submit,
// Cancel, Resize, PostPlatformEvent) are independently safe to call
// concurrently because each one constructs its own request.
type RemoteClient struct {
	httpClient *http.Client
	baseURL    string
	sessionID  string
	sidecarURL string

	frames chan kernel.RenderFrame
	errors chan error

	reconnectInitial time.Duration
	reconnectMax     time.Duration

	reconnectCount atomic.Uint64
	closed         atomic.Bool
	cancel         context.CancelFunc

	mu sync.Mutex // guards lifecycle ops

	ws        *websocket.Conn
	sidecarWS *websocket.Conn
	pending   map[string]chan websocketMessage
	pendingMu sync.Mutex
	writeMu   sync.Mutex
	sidecarMu sync.Mutex
	reqSeq    atomic.Uint64
}

// DialOption tunes a RemoteClient before DialSSE opens the stream.
// Options are evaluated in order; later options override earlier ones for
// the same field. The package keeps the option set deliberately small so
// upstream parity discussions stay focused on the wire contract.
type DialOption func(*RemoteClient)

// WithHTTPClient swaps the http.Client used for the SSE GET and the
// outbound POST helpers. Tests pass a client backed by an httptest
// server; production wiring uses http.DefaultClient.
func WithHTTPClient(c *http.Client) DialOption {
	return func(rc *RemoteClient) {
		if c != nil {
			rc.httpClient = c
		}
	}
}

// WithSessionID sets the session id RemoteClient embeds in every outbound
// platform event. The remote gateway uses it to route the dispatch back
// to the correct kernel turn. Empty leaves the field blank, matching the
// "no resident session" invariant.
func WithSessionID(sid string) DialOption {
	return func(rc *RemoteClient) { rc.sessionID = sid }
}

// WithSidecarURL configures a best-effort websocket mirror that receives raw
// event frames from a websocket attach session. A sidecar connection failure is
// reported on Errors() but does not tear down the main gateway connection.
func WithSidecarURL(raw string) DialOption {
	return func(rc *RemoteClient) { rc.sidecarURL = strings.TrimSpace(raw) }
}

// WithReconnectBackoff overrides the exponential reconnect envelope.
// initial is the first wait after a transport drop; the wait doubles up
// to max. Tests pin both to small values so a forced reconnect lands
// inside the test deadline.
func WithReconnectBackoff(initial, max time.Duration) DialOption {
	return func(rc *RemoteClient) {
		if initial > 0 {
			rc.reconnectInitial = initial
		}
		if max > 0 && max >= initial {
			rc.reconnectMax = max
		}
	}
}

// NewRemoteClient constructs a RemoteClient bound to baseURL with no
// in-flight SSE connection. Callers either call DialSSE to open the
// events stream (the typical path) or use the POST helpers directly when
// they only need to send a one-shot platform event without consuming
// frames. The returned client is otherwise quiescent.
func NewRemoteClient(baseURL string, opts ...DialOption) *RemoteClient {
	rc := &RemoteClient{
		httpClient:       http.DefaultClient,
		baseURL:          strings.TrimRight(baseURL, "/"),
		frames:           make(chan kernel.RenderFrame, 8),
		errors:           make(chan error, 4),
		reconnectInitial: defaultReconnectInitial,
		reconnectMax:     defaultReconnectMax,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(rc)
		}
	}
	return rc
}

// DialSSE opens the GET /events SSE stream and returns a RemoteClient
// pumping kernel.RenderFrames onto Frames(). The first connection must
// succeed: a 4xx/5xx or transport error surfaces immediately so callers
// can surface "remote streaming unavailable" without spinning. Later
// reconnects happen on a worker goroutine and never block the caller.
//
// The returned client owns a derived context cancelled by Close. The
// Frames channel closes when the run loop exits (either via Close or
// when ctx is cancelled).
func DialSSE(ctx context.Context, baseURL string, opts ...DialOption) (*RemoteClient, error) {
	rc := NewRemoteClient(baseURL, opts...)
	runCtx, cancel := context.WithCancel(ctx)
	rc.cancel = cancel

	resp, err := rc.openStream(runCtx)
	if err != nil {
		cancel()
		return nil, err
	}

	go rc.run(runCtx, resp)
	return rc, nil
}

// DialWebSocketAttach connects to an existing TUI gateway websocket using the
// Hermes JSON-RPC attach contract. It adapts event frames into the same native
// RenderFrame stream consumed by the SSE remote TUI path.
func DialWebSocketAttach(ctx context.Context, gatewayURL string, opts ...DialOption) (*RemoteClient, error) {
	rc := NewRemoteClient(gatewayURL, opts...)
	rc.pending = make(map[string]chan websocketMessage)
	runCtx, cancel := context.WithCancel(ctx)
	rc.cancel = cancel

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, gatewayURL, nil)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("tuigateway: websocket attach %s: %w", RedactRemoteURL(gatewayURL), err)
	}
	rc.ws = conn

	if rc.sidecarURL != "" {
		rc.connectSidecar(ctx)
	}

	go rc.runWebSocket(runCtx)

	result, err := rc.requestWebSocket(ctx, "session.create", map[string]any{"cols": 80})
	if err != nil {
		rc.Close()
		return nil, err
	}
	var created struct {
		SessionID string `json:"session_id"`
	}
	if len(result) > 0 {
		if err := json.Unmarshal(result, &created); err != nil {
			rc.Close()
			return nil, fmt.Errorf("tuigateway: decode session.create result: %w", err)
		}
	}
	rc.sessionID = strings.TrimSpace(created.SessionID)
	return rc, nil
}

// Frames returns the receive-side of the kernel render-frame stream. The
// channel is closed when the run loop exits.
func (rc *RemoteClient) Frames() <-chan kernel.RenderFrame { return rc.frames }

// Errors returns transport-error events the consumer can drain for
// observability. The channel is buffered; older errors are dropped if
// the buffer fills, since the Frames stream is the authoritative signal.
func (rc *RemoteClient) Errors() <-chan error { return rc.errors }

// Reconnects reports the number of successful reconnect attempts since
// DialSSE. Useful for tests asserting the reconnect path actually fired.
func (rc *RemoteClient) Reconnects() uint64 { return rc.reconnectCount.Load() }

// Close signals the run loop to stop and releases the embedded context.
// Frames closes shortly after. Idempotent: safe to call multiple times.
func (rc *RemoteClient) Close() {
	if !rc.closed.CompareAndSwap(false, true) {
		return
	}
	rc.mu.Lock()
	cancel := rc.cancel
	rc.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	rc.writeMu.Lock()
	if rc.ws != nil {
		_ = rc.ws.Close()
		rc.ws = nil
	}
	rc.writeMu.Unlock()
	rc.sidecarMu.Lock()
	if rc.sidecarWS != nil {
		_ = rc.sidecarWS.Close()
		rc.sidecarWS = nil
	}
	rc.sidecarMu.Unlock()
}

// PostPlatformEvent serialises evt as JSON and POSTs it to the gateway's
// generic /platform-event endpoint. The gateway dispatches on the kind
// discriminator, mirroring the upstream JSON-RPC method tag without the
// envelope. Returns the underlying transport error, if any, and any
// non-2xx response promoted to a typed error.
func (rc *RemoteClient) PostPlatformEvent(ctx context.Context, evt platformEvent) error {
	return rc.postJSON(ctx, "/platform-event", evt)
}

// Submit posts a SubmitEvent to /submit using the resident session id.
// Equivalent to PostPlatformEvent on a SubmitEvent, but routed through
// the dedicated endpoint so the GatewayMux can dispatch with simple
// path matchers instead of a kind switch on every request.
func (rc *RemoteClient) Submit(ctx context.Context, text string) error {
	if rc.isWebSocketAttach() {
		_, err := rc.requestWebSocket(ctx, "prompt.submit", map[string]any{
			"session_id": rc.sessionID,
			"text":       text,
		})
		return err
	}
	return rc.postJSON(ctx, "/submit", SubmitEvent{
		Kind:      PlatformEventKindSubmit,
		SessionID: rc.sessionID,
		Text:      text,
	})
}

// Cancel posts a CancelEvent to /cancel using the resident session id.
func (rc *RemoteClient) Cancel(ctx context.Context) error {
	if rc.isWebSocketAttach() {
		_, err := rc.requestWebSocket(ctx, "session.interrupt", map[string]any{
			"session_id": rc.sessionID,
		})
		return err
	}
	return rc.postJSON(ctx, "/cancel", CancelEvent{
		Kind:      PlatformEventKindCancel,
		SessionID: rc.sessionID,
	})
}

// Resize posts a ResizeEvent to /resize using the resident session id
// and the supplied terminal column count.
func (rc *RemoteClient) Resize(ctx context.Context, cols int) error {
	if rc.isWebSocketAttach() {
		_, err := rc.requestWebSocket(ctx, "terminal.resize", map[string]any{
			"session_id": rc.sessionID,
			"cols":       cols,
		})
		return err
	}
	return rc.postJSON(ctx, "/resize", ResizeEvent{
		Kind:      PlatformEventKindResize,
		SessionID: rc.sessionID,
		Cols:      cols,
	})
}

func (rc *RemoteClient) isWebSocketAttach() bool {
	rc.writeMu.Lock()
	defer rc.writeMu.Unlock()
	return rc.ws != nil
}

func (rc *RemoteClient) postJSON(ctx context.Context, path string, body any) error {
	endpoint, err := url.JoinPath(rc.baseURL, path)
	if err != nil {
		return fmt.Errorf("tuigateway: build %s URL: %w", path, err)
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("tuigateway: marshal %s body: %w", path, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("tuigateway: build %s request: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("tuigateway: POST %s: %w", path, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("tuigateway: POST %s: HTTP %d", path, resp.StatusCode)
	}
	return nil
}

func (rc *RemoteClient) openStream(ctx context.Context) (*http.Response, error) {
	endpoint, err := url.JoinPath(rc.baseURL, "/events")
	if err != nil {
		return nil, fmt.Errorf("tuigateway: build /events URL: %w", err)
	}
	if rc.sessionID != "" {
		endpoint += "?session_id=" + url.QueryEscape(rc.sessionID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("tuigateway: build /events request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tuigateway: dial /events: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("tuigateway: /events returned HTTP %d", resp.StatusCode)
	}
	return resp, nil
}

// run is the long-lived worker that drains an SSE response body and
// reconnects on transport errors until ctx is cancelled. Closes the
// frames channel on exit so consumers never block indefinitely.
func (rc *RemoteClient) run(ctx context.Context, initial *http.Response) {
	defer close(rc.frames)

	resp := initial
	backoff := rc.reconnectInitial
	for {
		if resp == nil {
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			r, err := rc.openStream(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				rc.publishError(err)
				backoff *= 2
				if backoff > rc.reconnectMax {
					backoff = rc.reconnectMax
				}
				continue
			}
			resp = r
			rc.reconnectCount.Add(1)
			backoff = rc.reconnectInitial
		}

		rc.consume(ctx, resp.Body)
		_ = resp.Body.Close()
		resp = nil

		if ctx.Err() != nil {
			return
		}
	}
}

// consume reads the SSE body until EOF or ctx cancellation, decoding
// frame events into kernel.RenderFrames. The local parser mirrors the
// minimal subset already used in internal/hermes/sse.go: lines beginning
// with "event: " set the discriminator, "data: " accumulates payload,
// blank lines flush.
func (rc *RemoteClient) consume(ctx context.Context, body io.Reader) {
	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)

	var event, data string
	flush := func() {
		if event == "frame" && data != "" {
			var f kernel.RenderFrame
			if err := json.Unmarshal([]byte(data), &f); err == nil {
				select {
				case rc.frames <- f:
				case <-ctx.Done():
				}
			}
		}
		event, data = "", ""
	}
	for sc.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		line := sc.Text()
		if line == "" {
			flush()
			continue
		}
		switch {
		case strings.HasPrefix(line, ":"):
			// SSE comment / keepalive — ignore.
		case strings.HasPrefix(line, "event: "):
			event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			if data != "" {
				data += "\n"
			}
			data += strings.TrimPrefix(line, "data: ")
		}
	}
	// EOF without a trailing blank line: flush whatever we have.
	if event != "" || data != "" {
		flush()
	}
}

func (rc *RemoteClient) publishError(err error) {
	select {
	case rc.errors <- err:
	default:
		// Drain one and try again so the caller sees the freshest error.
		select {
		case <-rc.errors:
		default:
		}
		select {
		case rc.errors <- err:
		default:
		}
	}
}

type websocketRequest struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      string          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type websocketMessage struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      string          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *websocketError `json:"error,omitempty"`
}

type websocketError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (rc *RemoteClient) connectSidecar(ctx context.Context) {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, rc.sidecarURL, nil)
	if err != nil {
		rc.publishError(fmt.Errorf("tuigateway: websocket sidecar attach %s: %w", RedactRemoteURL(rc.sidecarURL), err))
		return
	}
	rc.sidecarMu.Lock()
	rc.sidecarWS = conn
	rc.sidecarMu.Unlock()
}

func (rc *RemoteClient) runWebSocket(ctx context.Context) {
	defer close(rc.frames)
	defer rc.rejectWebSocketPending(errors.New("gateway websocket closed"))

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		rc.writeMu.Lock()
		conn := rc.ws
		rc.writeMu.Unlock()
		if conn == nil {
			return
		}
		_, payload, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() == nil && !rc.closed.Load() {
				rc.publishError(fmt.Errorf("tuigateway: websocket read: %w", err))
			}
			return
		}
		var msg websocketMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			rc.publishError(fmt.Errorf("tuigateway: websocket protocol: %w", err))
			continue
		}
		if msg.Method == "event" {
			rc.mirrorWebSocketEvent(payload)
			rc.handleWebSocketEvent(ctx, msg.Params)
			continue
		}
		if msg.ID != "" {
			rc.deliverWebSocketResponse(msg)
		}
	}
}

func (rc *RemoteClient) requestWebSocket(ctx context.Context, method string, params any) (json.RawMessage, error) {
	rc.writeMu.Lock()
	conn := rc.ws
	rc.writeMu.Unlock()
	if conn == nil {
		return nil, errors.New("tuigateway: websocket attach is not connected")
	}
	rawParams, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("tuigateway: marshal websocket %s params: %w", method, err)
	}
	id := fmt.Sprintf("r%d", rc.reqSeq.Add(1))
	ch := make(chan websocketMessage, 1)
	rc.pendingMu.Lock()
	rc.pending[id] = ch
	rc.pendingMu.Unlock()

	req := websocketRequest{JSONRPC: "2.0", ID: id, Method: method, Params: rawParams}
	rc.writeMu.Lock()
	err = conn.WriteJSON(req)
	rc.writeMu.Unlock()
	if err != nil {
		rc.pendingMu.Lock()
		delete(rc.pending, id)
		rc.pendingMu.Unlock()
		return nil, fmt.Errorf("tuigateway: websocket %s send: %w", method, err)
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			if resp.Error.Message != "" {
				return nil, fmt.Errorf("tuigateway: websocket %s: %s", method, resp.Error.Message)
			}
			return nil, fmt.Errorf("tuigateway: websocket %s failed", method)
		}
		return resp.Result, nil
	case <-ctx.Done():
		rc.pendingMu.Lock()
		delete(rc.pending, id)
		rc.pendingMu.Unlock()
		return nil, ctx.Err()
	}
}

func (rc *RemoteClient) deliverWebSocketResponse(msg websocketMessage) {
	rc.pendingMu.Lock()
	ch := rc.pending[msg.ID]
	delete(rc.pending, msg.ID)
	rc.pendingMu.Unlock()
	if ch != nil {
		ch <- msg
	}
}

func (rc *RemoteClient) rejectWebSocketPending(err error) {
	rc.pendingMu.Lock()
	pending := rc.pending
	rc.pending = make(map[string]chan websocketMessage)
	rc.pendingMu.Unlock()
	for _, ch := range pending {
		ch <- websocketMessage{Error: &websocketError{Message: err.Error()}}
	}
}

func (rc *RemoteClient) handleWebSocketEvent(ctx context.Context, raw json.RawMessage) {
	var params struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		rc.publishError(fmt.Errorf("tuigateway: websocket event params: %w", err))
		return
	}
	if params.Type != "frame" || len(params.Payload) == 0 {
		return
	}
	var frame kernel.RenderFrame
	if err := json.Unmarshal(params.Payload, &frame); err != nil {
		rc.publishError(fmt.Errorf("tuigateway: websocket frame payload: %w", err))
		return
	}
	select {
	case rc.frames <- frame:
	case <-ctx.Done():
	}
}

func (rc *RemoteClient) mirrorWebSocketEvent(payload []byte) {
	rc.sidecarMu.Lock()
	conn := rc.sidecarWS
	if conn == nil {
		rc.sidecarMu.Unlock()
		return
	}
	err := conn.WriteMessage(websocket.TextMessage, payload)
	if err != nil {
		_ = conn.Close()
		rc.sidecarWS = nil
	}
	rc.sidecarMu.Unlock()
	if err != nil {
		rc.publishError(fmt.Errorf("tuigateway: websocket sidecar mirror: %w", err))
	}
}

// RedactRemoteURL removes query-string bearer tokens and embedded user-info
// credentials from remote gateway URLs before they are shown in diagnostics.
func RedactRemoteURL(raw string) string {
	if raw == "" {
		return raw
	}
	if parsed, err := url.Parse(raw); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		userInfo := ""
		if parsed.User != nil {
			userInfo = "***@"
		}
		query := ""
		if parsed.RawQuery != "" {
			query = "?***"
		}
		return parsed.Scheme + "://" + userInfo + parsed.Host + parsed.EscapedPath() + query
	}
	noUserInfo := raw
	if schemeIdx := strings.Index(noUserInfo, "://"); schemeIdx >= 0 {
		authorityStart := schemeIdx + len("://")
		authorityEnd := len(noUserInfo)
		for _, sep := range []string{"/", "?", "#"} {
			if idx := strings.Index(noUserInfo[authorityStart:], sep); idx >= 0 && authorityStart+idx < authorityEnd {
				authorityEnd = authorityStart + idx
			}
		}
		if at := strings.LastIndex(noUserInfo[authorityStart:authorityEnd], "@"); at >= 0 {
			noUserInfo = noUserInfo[:authorityStart] + "***@" + noUserInfo[authorityStart+at+1:]
		}
	}
	if queryIdx := strings.Index(noUserInfo, "?"); queryIdx >= 0 {
		return noUserInfo[:queryIdx] + "?***"
	}
	return noUserInfo
}
