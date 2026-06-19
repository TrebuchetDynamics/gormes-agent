package navivox

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const (
	navivoxWebSocketProtocol            = "navivox.v1"
	navivoxWebSocketTokenProtocolPrefix = "gormes.navivox.token."
	navivoxEventBufferCap               = 256
	navivoxSessionMaxAge                = 24 * time.Hour
	navivoxSessionSweepInterval         = 5 * time.Minute
	navivoxWebSocketPingInterval        = 25 * time.Second
	navivoxWebSocketReadTimeout         = 90 * time.Second
)

type sessionState struct {
	ID            string    `json:"session_id"`
	LastRequestID string    `json:"last_request_id,omitempty"`
	ProfileServer string    `json:"profile_server,omitempty"`
	ProfileID     string    `json:"profile_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Subscribers   int       `json:"subscribers"`

	lastMessageID string
	lastText      string
	seq           int
}

type client struct {
	ch       *Channel
	conn     *websocket.Conn
	writeMu  sync.Mutex
	sessions map[string]struct{}
	requests map[string]string
	identity string
	events   chan ServerEvent
	done     chan struct{}
}

type ClientMessage struct {
	Type      string         `json:"type"`
	RequestID string         `json:"request_id"`
	SessionID string         `json:"session_id,omitempty"`
	Text      string         `json:"text,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type ServerEvent struct {
	Type       string          `json:"type"`
	RequestID  string          `json:"request_id,omitempty"`
	SessionID  string          `json:"session_id,omitempty"`
	Text       string          `json:"text,omitempty"`
	Code       string          `json:"code,omitempty"`
	Message    string          `json:"message,omitempty"`
	ToolName   string          `json:"tool_name,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Status     string          `json:"status,omitempty"`
	SafetyID   string          `json:"safety_id,omitempty"`
	ApprovalID string          `json:"approval_id,omitempty"`
	Severity   string          `json:"severity,omitempty"`
	Risk       string          `json:"risk,omitempty"`
	Seq        int             `json:"seq,omitempty"`
	Metadata   map[string]any  `json:"metadata,omitempty"`
	Contact    *ProfileContact `json:"contact,omitempty"`
}

type SafetyEvent struct {
	ID       string
	Severity string
	Message  string
	Risk     string
	Metadata map[string]any
}

type ApprovalEvent struct {
	ID         string
	ToolCallID string
	Prompt     string
	Risk       string
	Metadata   map[string]any
}

func (c *Channel) handleStream(inbox chan<- gateway.InboundEvent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c.authRateLimited(r) {
			writeNavivoxError(w, http.StatusTooManyRequests, "", "auth_rate_limited", "Authentication attempts are temporarily rate limited")
			return
		}
		if navivoxRequestHasURLCredential(r) {
			c.recordAuthFailure(r)
			writeNavivoxError(w, http.StatusUnauthorized, "", "url_credentials_rejected", "URL credentials are not accepted")
			return
		}
		identity, ok := c.authenticate(r)
		if !ok {
			c.recordAuthFailure(r)
			writeNavivoxError(w, http.StatusUnauthorized, "", "unauthorized", "Unauthorized")
			return
		}
		c.clearAuthFailures(r)
		if !navivoxWebSocketProtocolOffered(r) {
			writeNavivoxError(w, http.StatusBadRequest, "", "protocol_required", "Navivox WebSocket protocol is required")
			return
		}
		releasePairingStream, ok := c.reservePairingStream()
		if !ok {
			writeNavivoxError(w, http.StatusConflict, "", "pairing_token_consumed", "Pairing token already claimed")
			return
		}
		upgrader := websocket.Upgrader{
			Subprotocols: []string{navivoxWebSocketProtocol},
			CheckOrigin: func(req *http.Request) bool {
				return c.originAllowed(req.Header.Get("Origin"))
			},
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			releasePairingStream(false)
			return
		}
		releasePairingStream(true)
		cl := &client{
			ch:       c,
			conn:     conn,
			sessions: map[string]struct{}{},
			requests: map[string]string{},
			identity: identity,
			events:   make(chan ServerEvent, navivoxEventBufferCap),
			done:     make(chan struct{}, 1),
		}
		conn.SetReadLimit(navivoxMaxTurnRequestBytes + 4096)
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(navivoxWebSocketReadTimeout))
		})
		conn.SetPingHandler(func(data string) error {
			_ = conn.SetReadDeadline(time.Now().Add(navivoxWebSocketReadTimeout))
			cl.writeMu.Lock()
			defer cl.writeMu.Unlock()
			return cl.conn.WriteMessage(websocket.PongMessage, []byte(data))
		})
		_ = conn.SetReadDeadline(time.Now().Add(navivoxWebSocketReadTimeout))
		go cl.eventPump()
		go cl.pingLoop()
		c.addClient(cl)
		defer c.removeClient(cl)
		defer conn.Close()
		for {
			_, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(navivoxWebSocketReadTimeout))
			if len(payload) > navivoxMaxTurnRequestBytes {
				var envelope struct {
					RequestID string `json:"request_id"`
				}
				_ = json.Unmarshal(payload, &envelope)
				_ = cl.write(ServerEvent{
					Type:      "error",
					RequestID: safeNavivoxRequestID(envelope.RequestID),
					Code:      "request_too_large",
					Message:   "Request is too large",
				})
				continue
			}
			var msg ClientMessage
			if err := json.Unmarshal(payload, &msg); err != nil {
				_ = cl.write(ServerEvent{
					Type:    "error",
					Code:    "bad_request",
					Message: "Invalid JSON",
				})
				continue
			}
			if err := cl.handle(r.Context(), inbox, msg); err != nil {
				_ = cl.write(ServerEvent{
					Type:      "error",
					RequestID: safeNavivoxRequestID(msg.RequestID),
					Code:      codeForNavivoxError(err),
					Message:   safeNavivoxError(err),
				})
			}
		}
	}
}

func navivoxWebSocketProtocolOffered(r *http.Request) bool {
	for _, protocol := range websocket.Subprotocols(r) {
		if protocol == navivoxWebSocketProtocol {
			return true
		}
	}
	return false
}

func (c *Channel) reservePairingStream() (func(bool), bool) {
	if !c.singleUsePairingStream {
		return func(bool) {}, true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pairingStreamReserved || c.pairingStreamConsumed {
		return nil, false
	}
	c.pairingStreamReserved = true
	return func(success bool) {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.pairingStreamReserved = false
		if success {
			c.pairingStreamConsumed = true
		}
	}, true
}

func (cl *client) handle(ctx context.Context, inbox chan<- gateway.InboundEvent, msg ClientMessage) error {
	msg.Type = strings.TrimSpace(msg.Type)
	requestID, ok := normalizeNavivoxRequestID(msg.RequestID)
	msg.RequestID = requestID
	if msg.RequestID == "" {
		return navivoxError{code: "bad_request", message: "request_id is required"}
	}
	if !ok {
		return navivoxError{code: "bad_request", message: "request_id is too long"}
	}
	switch msg.Type {
	case "ping":
		return cl.write(ServerEvent{Type: "pong", RequestID: msg.RequestID})
	case "start_turn":
		sessionID, contact, err := cl.ch.enqueueTurn(ctx, inbox, turnInputFromClientMessage(msg), cl.identity)
		if err != nil {
			return err
		}
		cl.subscribe(sessionID, msg.RequestID)
		if err := cl.write(ServerEvent{Type: "session_started", RequestID: msg.RequestID, SessionID: sessionID}); err != nil {
			return err
		}
		if contact != nil {
			cl.ch.broadcastProfileContact(*contact)
		}
		return nil
	case "cancel_turn":
		return cl.enqueueTurnControl(ctx, inbox, msg, "cancelled")
	case "stop_turn":
		return cl.enqueueTurnControl(ctx, inbox, msg, "stopped")
	case "subscribe_session":
		sessionID, ok := normalizeNavivoxSessionID(msg.SessionID)
		if sessionID == "" {
			return navivoxError{code: "bad_request", message: "session_id is required"}
		}
		if !ok {
			return navivoxError{code: "bad_request", message: "session_id is too long"}
		}
		cl.subscribe(sessionID, msg.RequestID)
		return cl.write(ServerEvent{Type: "session_started", RequestID: msg.RequestID, SessionID: sessionID})
	default:
		return navivoxError{code: "bad_request", message: "unsupported message type"}
	}
}

func (cl *client) enqueueTurnControl(ctx context.Context, inbox chan<- gateway.InboundEvent, msg ClientMessage, status string) error {
	sessionID, ok := normalizeNavivoxSessionID(msg.SessionID)
	if sessionID == "" {
		return navivoxError{code: "bad_request", message: "session_id is required"}
	}
	if !ok {
		return navivoxError{code: "bad_request", message: "session_id is too long"}
	}
	ev := gateway.InboundEvent{
		Platform:  PlatformName,
		ChatID:    sessionID,
		ChatType:  "private",
		UserID:    "navivox",
		UserName:  cl.identity,
		MsgID:     msg.RequestID,
		MessageID: msg.RequestID,
		Kind:      gateway.EventCancel,
	}
	if err := enqueue(ctx, inbox, ev); err != nil {
		return err
	}
	return cl.write(ServerEvent{Type: "done", RequestID: msg.RequestID, SessionID: sessionID, Status: status})
}

func (c *Channel) sweepSessions() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	for id, state := range c.sessions {
		if now.Sub(state.UpdatedAt) > navivoxSessionMaxAge {
			delete(c.sessions, id)
		}
	}
}

func (c *Channel) ensureSessionLocked(id, requestID string) *sessionState {
	state, ok := c.sessions[id]
	now := c.now()
	if !ok {
		state = &sessionState{ID: id, CreatedAt: now}
		c.sessions[id] = state
	}
	if requestID != "" {
		state.LastRequestID = requestID
	}
	state.UpdatedAt = now
	return state
}

func (cl *client) subscribe(sessionID, requestID string) {
	cl.ch.mu.Lock()
	defer cl.ch.mu.Unlock()
	_, alreadySubscribed := cl.sessions[sessionID]
	cl.sessions[sessionID] = struct{}{}
	if requestID != "" {
		cl.requests[sessionID] = requestID
	}
	state := cl.ch.ensureSessionLocked(sessionID, requestID)
	if !alreadySubscribed {
		state.Subscribers++
	}
}

func (c *Channel) addClient(cl *client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clients[cl] = struct{}{}
}

func (c *Channel) removeClient(cl *client) {
	c.mu.Lock()
	delete(c.clients, cl)
	for sessionID := range cl.sessions {
		if state := c.sessions[sessionID]; state != nil && state.Subscribers > 0 {
			state.Subscribers--
		}
	}
	if c.singleUsePairingStream {
		c.pairingStreamConsumed = false
	}
	c.mu.Unlock()
	close(cl.events)
	<-cl.done
}

func (c *Channel) broadcast(sessionID string, ev ServerEvent) {
	c.mu.Lock()
	for cl := range c.clients {
		if _, ok := cl.sessions[sessionID]; !ok {
			continue
		}
		next := ev
		if next.RequestID == "" {
			next.RequestID = cl.requests[sessionID]
		}
		select {
		case cl.events <- next:
		default:
			// Buffer full — drop event rather than block all clients.
		}
	}
	if state := c.sessions[sessionID]; state != nil {
		state.UpdatedAt = c.now()
	}
	c.mu.Unlock()
}

func (cl *client) eventPump() {
	defer close(cl.done)
	for ev := range cl.events {
		if err := cl.write(ev); err != nil {
			return
		}
	}
}

func (cl *client) pingLoop() {
	ticker := time.NewTicker(navivoxWebSocketPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-cl.done:
			return
		case <-ticker.C:
			cl.writeMu.Lock()
			err := cl.conn.WriteMessage(websocket.PingMessage, nil)
			cl.writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

func (cl *client) write(ev ServerEvent) error {
	cl.writeMu.Lock()
	defer cl.writeMu.Unlock()
	return cl.conn.WriteJSON(ev)
}

func safeNavivoxToolName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "tool_progress"
	}
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-' || r == '.' || r == ':':
			b.WriteRune(r)
		}
		if b.Len() >= 64 {
			break
		}
	}
	if b.Len() == 0 {
		return "tool_progress"
	}
	return b.String()
}

func safeNavivoxToolSummary(raw string) string {
	raw = strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if raw == "" {
		return "Tool progress"
	}
	runes := []rune(raw)
	if len(runes) > 240 {
		return string(runes[:237]) + "..."
	}
	return raw
}

func safeNavivoxToolMetadata(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]any, len(raw))
	for key, value := range raw {
		safeKey := safeNavivoxToolName(key)
		if safeKey == "tool_progress" && strings.TrimSpace(key) != "tool_progress" {
			continue
		}
		if navivoxToolMetadataSensitiveKey(safeKey) {
			continue
		}
		switch typed := value.(type) {
		case string:
			out[safeKey] = safeNavivoxToolSummary(typed)
		case bool:
			out[safeKey] = typed
		case int:
			out[safeKey] = typed
		case int64:
			out[safeKey] = typed
		case float64:
			out[safeKey] = typed
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func navivoxToolMetadataSensitiveKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	collapsed := strings.NewReplacer("_", "", "-", "", ".", "", ":", " ").Replace(key)
	collapsed = strings.Join(strings.Fields(collapsed), "")
	for _, marker := range []string{"secret", "token", "password", "apikey", "credential", "rawaudio", "audiobytes"} {
		if strings.Contains(collapsed, marker) {
			return true
		}
	}
	return false
}
