package navivox

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/network/vpnhost"
)

const PlatformName = "navivox"

const (
	navivoxWebSocketProtocol            = "gormes.navivox.v1"
	navivoxWebSocketTokenProtocolPrefix = "gormes.navivox.token."
	navivoxEventBufferCap               = 256
	navivoxSessionMaxAge                = 24 * time.Hour
	navivoxSessionSweepInterval         = 5 * time.Minute
)

// vpnHostLister is the seam tests override to inject deterministic VPN
// host enumeration into NewChannel; production callers use the real CLIs.
var vpnHostLister = vpnhost.List

type Channel struct {
	cfg config.NavivoxCfg
	log *slog.Logger

	now   func() time.Time
	newID func() string

	mu       sync.Mutex
	sessions map[string]*sessionState
	clients  map[*client]struct{}

	profileContacts map[string]ProfileContact
	loadContacts    func(context.Context) ([]ProfileContact, error)
}

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
	Seq        int             `json:"seq,omitempty"`
	Metadata   map[string]any  `json:"metadata,omitempty"`
	Contact    *ProfileContact `json:"contact,omitempty"`
}

type turnRequest struct {
	RequestID string         `json:"request_id"`
	SessionID string         `json:"session_id,omitempty"`
	Text      string         `json:"text"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

func NewChannel(cfg config.NavivoxCfg, log *slog.Logger) (*Channel, error) {
	if log == nil {
		log = slog.Default()
	}
	if err := config.ValidateNavivoxForRuntime(&cfg); err != nil {
		return nil, err
	}
	if cfg.Enabled && config.NavivoxExposureRequiresVPN(cfg.ExposureMode) {
		hosts, _ := vpnHostLister(context.Background())
		ips := make([]string, 0, len(hosts)*2)
		for _, h := range hosts {
			if h.IPv4 != "" {
				ips = append(ips, h.IPv4)
			}
			if h.IPv6 != "" {
				ips = append(ips, h.IPv6)
			}
		}
		if err := config.ValidateNavivoxBindAgainstVPN(&cfg, ips); err != nil {
			return nil, err
		}
	}
	return &Channel{
		cfg:             cfg,
		log:             log,
		now:             func() time.Time { return time.Now().UTC() },
		newID:           randomID,
		sessions:        map[string]*sessionState{},
		clients:         map[*client]struct{}{},
		profileContacts: map[string]ProfileContact{},
	}, nil
}

func (c *Channel) Name() string { return PlatformName }

func (c *Channel) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	server := &http.Server{
		Addr:    net.JoinHostPort(c.cfg.BindHost, fmt.Sprintf("%d", c.cfg.Port)),
		Handler: c.Handler(inbox),
	}
	errCh := make(chan error, 1)
	go func() {
		c.log.Info("navivox gateway channel listening",
			"bind_host", c.cfg.BindHost,
			"port", c.cfg.Port,
			"exposure_mode", c.cfg.ExposureMode,
			"auth_mode", c.cfg.AuthMode)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return ctx.Err()
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return ctx.Err()
		}
		return err
	}
}

func (c *Channel) Handler(inbox chan<- gateway.InboundEvent) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", c.handleHealthz)
	mux.HandleFunc("/v1/navivox/status", c.withAuth(c.handleStatus))
	mux.HandleFunc("/v1/navivox/profile-contacts", c.withAuth(c.handleProfileContacts))
	mux.HandleFunc("/v1/navivox/sessions", c.withAuth(c.handleSessions))
	mux.HandleFunc("/v1/navivox/sessions/", c.withAuth(c.handleSessionByID))
	mux.HandleFunc("/v1/navivox/turn", c.withAuth(c.handleTurn(inbox)))
	mux.HandleFunc("/v1/navivox/stream", c.handleStream(inbox))
	return c.cors(mux)
}

func (c *Channel) Send(ctx context.Context, chatID, text string) (string, error) {
	msgID := c.newID()
	c.broadcast(chatID, ServerEvent{
		Type:      "assistant_message",
		SessionID: chatID,
		Text:      text,
	})
	c.updateProfileContactForSession(chatID, text, "assistant", ProfileContactTurnIdle)
	c.broadcast(chatID, ServerEvent{Type: "done", SessionID: chatID})
	return msgID, ctx.Err()
}

func (c *Channel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	msgID := c.newID()
	c.mu.Lock()
	c.ensureSessionLocked(chatID, "").lastMessageID = msgID
	c.mu.Unlock()
	return msgID, ctx.Err()
}

func (c *Channel) SendToolProgress(ctx context.Context, chatID string, progress gateway.ToolProgressEvent) (string, error) {
	toolCallID := strings.TrimSpace(progress.ID)
	if toolCallID == "" {
		toolCallID = c.newID()
	}
	status := strings.TrimSpace(string(progress.Status))
	if status == "" {
		status = string(gateway.ToolProgressStarted)
	}
	eventType := "tool_call_started"
	switch gateway.ToolProgressStatus(status) {
	case gateway.ToolProgressFinished, gateway.ToolProgressFailed:
		eventType = "tool_call_finished"
	}
	c.broadcast(chatID, ServerEvent{
		Type:       eventType,
		SessionID:  chatID,
		ToolName:   safeNavivoxToolName(progress.ToolName),
		ToolCallID: toolCallID,
		Status:     status,
		Message:    safeNavivoxToolSummary(progress.Summary),
		Metadata:   safeNavivoxToolMetadata(progress.Metadata),
	})
	return toolCallID, ctx.Err()
}

func (c *Channel) EditMessage(ctx context.Context, chatID, msgID, text string) error {
	return c.edit(chatID, msgID, text, false)
}

func (c *Channel) EditMessageFinal(ctx context.Context, chatID, msgID, text string, finalize bool) error {
	return c.edit(chatID, msgID, text, finalize)
}

func (c *Channel) edit(chatID, msgID, text string, finalize bool) error {
	c.mu.Lock()
	session := c.ensureSessionLocked(chatID, "")
	isPrefix := session.lastMessageID == msgID && strings.HasPrefix(text, session.lastText)
	delta := text
	seq := 0
	reset := false
	if isPrefix {
		delta = strings.TrimPrefix(text, session.lastText)
		seq = session.seq
		session.seq++
	} else {
		// Prefix match failed (LLM rewrote mid-stream). Send full text with reset flag.
		seq = session.seq
		session.seq++
		reset = true
		delta = text
	}
	session.lastMessageID = msgID
	session.lastText = text
	session.UpdatedAt = c.now()
	c.mu.Unlock()

	if !finalize {
		if delta != "" {
			ev := ServerEvent{Type: "assistant_delta", SessionID: chatID, Text: delta, Seq: seq}
			if reset {
				ev.Metadata = map[string]any{"reset": true}
			}
			c.broadcast(chatID, ev)
		}
		return nil
	}
	c.updateProfileContactForSession(chatID, text, "assistant", ProfileContactTurnIdle)
	c.broadcast(chatID, ServerEvent{Type: "assistant_message", SessionID: chatID, Text: text})
	c.broadcast(chatID, ServerEvent{Type: "done", SessionID: chatID})
	return nil
}

func (c *Channel) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
		return
	}
	writeNavivoxJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"platform": PlatformName,
	})
}

func (c *Channel) handleStatus(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodGet {
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
		return
	}
	c.mu.Lock()
	sessionCount := len(c.sessions)
	clientCount := len(c.clients)
	c.mu.Unlock()
	writeNavivoxJSON(w, http.StatusOK, map[string]any{
		"enabled":        c.cfg.Enabled,
		"bind_host":      c.cfg.BindHost,
		"port":           c.cfg.Port,
		"exposure_mode":  c.cfg.ExposureMode,
		"auth_mode":      c.cfg.AuthMode,
		"sessions":       sessionCount,
		"ws_connections": clientCount,
	})
}

func (c *Channel) handleProfileContacts(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodGet {
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
		return
	}
	contacts, err := c.profileContactSnapshot(r.Context())
	if err != nil {
		writeNavivoxError(w, http.StatusServiceUnavailable, "", "profile_contacts_unavailable", "Profile contacts are unavailable")
		return
	}
	writeNavivoxJSON(w, http.StatusOK, profileContactSnapshot{Contacts: contacts})
}

func (c *Channel) handleSessions(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodGet {
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
		return
	}
	c.mu.Lock()
	sessions := make([]sessionState, 0, len(c.sessions))
	for _, state := range c.sessions {
		sessions = append(sessions, *state)
	}
	c.mu.Unlock()
	writeNavivoxJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (c *Channel) handleSessionByID(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodGet {
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/navivox/sessions/")
	id = strings.TrimSpace(id)
	if id == "" {
		writeNavivoxError(w, http.StatusNotFound, "", "not_found", "Session not found")
		return
	}
	c.mu.Lock()
	state, ok := c.sessions[id]
	var out sessionState
	if ok {
		out = *state
	}
	c.mu.Unlock()
	if !ok {
		writeNavivoxError(w, http.StatusNotFound, "", "not_found", "Session not found")
		return
	}
	writeNavivoxJSON(w, http.StatusOK, map[string]any{"session": out})
}

func (c *Channel) handleTurn(inbox chan<- gateway.InboundEvent) func(http.ResponseWriter, *http.Request, string) {
	return func(w http.ResponseWriter, r *http.Request, identity string) {
		if r.Method != http.MethodPost {
			writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
			return
		}
		var req turnRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
			writeNavivoxError(w, http.StatusBadRequest, "", "bad_request", "Invalid JSON")
			return
		}
		sessionID, contact, err := c.enqueueTurn(r.Context(), inbox, ClientMessage{
			Type:      "start_turn",
			RequestID: req.RequestID,
			SessionID: req.SessionID,
			Text:      req.Text,
			Metadata:  req.Metadata,
		}, identity)
		if err != nil {
			writeNavivoxError(w, statusForNavivoxError(err), req.RequestID, codeForNavivoxError(err), safeNavivoxError(err))
			return
		}
		writeNavivoxJSON(w, http.StatusAccepted, map[string]any{
			"request_id": req.RequestID,
			"session_id": sessionID,
			"status":     "queued",
		})
		if contact != nil {
			c.broadcastProfileContact(*contact)
		}
	}
}

func (c *Channel) handleStream(inbox chan<- gateway.InboundEvent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, ok := c.authenticate(r)
		if !ok {
			writeNavivoxError(w, http.StatusUnauthorized, "", "unauthorized", "Unauthorized")
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
			return
		}
		cl := &client{
			ch:       c,
			conn:     conn,
			sessions: map[string]struct{}{},
			requests: map[string]string{},
			identity: identity,
			events:   make(chan ServerEvent, navivoxEventBufferCap),
			done:     make(chan struct{}, 1),
		}
		go cl.eventPump()
		c.addClient(cl)
		defer c.removeClient(cl)
		defer conn.Close()
		for {
			_, payload, err := conn.ReadMessage()
			if err != nil {
				return
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
					RequestID: msg.RequestID,
					Code:      codeForNavivoxError(err),
					Message:   safeNavivoxError(err),
				})
			}
		}
	}
}

func (cl *client) handle(ctx context.Context, inbox chan<- gateway.InboundEvent, msg ClientMessage) error {
	msg.Type = strings.TrimSpace(msg.Type)
	msg.RequestID = strings.TrimSpace(msg.RequestID)
	if msg.RequestID == "" {
		return navivoxError{code: "bad_request", message: "request_id is required"}
	}
	switch msg.Type {
	case "ping":
		return cl.write(ServerEvent{Type: "pong", RequestID: msg.RequestID})
	case "start_turn":
		sessionID, contact, err := cl.ch.enqueueTurn(ctx, inbox, msg, cl.identity)
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
		sessionID := strings.TrimSpace(msg.SessionID)
		if sessionID == "" {
			return navivoxError{code: "bad_request", message: "session_id is required"}
		}
		cl.subscribe(sessionID, msg.RequestID)
		return cl.write(ServerEvent{Type: "session_started", RequestID: msg.RequestID, SessionID: sessionID})
	default:
		return navivoxError{code: "bad_request", message: "unsupported message type"}
	}
}

func (cl *client) enqueueTurnControl(ctx context.Context, inbox chan<- gateway.InboundEvent, msg ClientMessage, status string) error {
	sessionID := strings.TrimSpace(msg.SessionID)
	if sessionID == "" {
		return navivoxError{code: "bad_request", message: "session_id is required"}
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

func (c *Channel) enqueueTurn(ctx context.Context, inbox chan<- gateway.InboundEvent, msg ClientMessage, identity string) (string, *ProfileContact, error) {
	requestID := strings.TrimSpace(msg.RequestID)
	if requestID == "" {
		return "", nil, navivoxError{code: "bad_request", message: "request_id is required"}
	}
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return "", nil, navivoxError{code: "bad_request", message: "text is required"}
	}
	sessionID := strings.TrimSpace(msg.SessionID)
	if sessionID == "" {
		sessionID = "navivox-" + c.newID()
	}
	serverID, profileID := profileScopeFromMetadata(msg.Metadata)
	c.mu.Lock()
	session := c.ensureSessionLocked(sessionID, requestID)
	session.ProfileServer = serverID
	session.ProfileID = profileID
	c.mu.Unlock()
	ev := gateway.InboundEvent{
		Platform:  PlatformName,
		ChatID:    sessionID,
		ChatType:  "private",
		UserID:    "navivox",
		UserName:  identity,
		MsgID:     requestID,
		MessageID: requestID,
		Kind:      gateway.EventSubmit,
		Text:      text,
	}
	if err := enqueue(ctx, inbox, ev); err != nil {
		return "", nil, err
	}
	c.mu.Lock()
	contact := c.profileContactRuntimeUpdateLocked(serverID, profileID, text, "user", ProfileContactTurnActive)
	c.mu.Unlock()
	c.log.Info("navivox turn queued", "client_identity", identity, "request_id", requestID, "session_id", sessionID, "action", "start_turn", "status", "queued")
	return sessionID, &contact, nil
}

func enqueue(ctx context.Context, inbox chan<- gateway.InboundEvent, ev gateway.InboundEvent) error {
	select {
	case inbox <- ev:
		return nil
	case <-ctx.Done():
		return navivoxError{code: "timeout", message: "request canceled"}
	default:
		return navivoxError{code: "runtime_error", message: "gateway inbox is full"}
	}
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

func (cl *client) write(ev ServerEvent) error {
	cl.writeMu.Lock()
	defer cl.writeMu.Unlock()
	return cl.conn.WriteJSON(ev)
}

type authenticatedHandler func(http.ResponseWriter, *http.Request, string)

func (c *Channel) withAuth(next authenticatedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, ok := c.authenticate(r)
		if !ok {
			writeNavivoxError(w, http.StatusUnauthorized, "", "unauthorized", "Unauthorized")
			return
		}
		next(w, r, identity)
	}
}

func (c *Channel) authenticate(r *http.Request) (string, bool) {
	mode := strings.ToLower(strings.TrimSpace(c.cfg.AuthMode))
	switch mode {
	case config.NavivoxAuthTailscaleIdentity:
		identity := firstHeader(r, "Tailscale-User-Login", "X-Tailscale-User-Login", "Tailscale-Device-Name", "X-Tailscale-Device-Name")
		if identity == "" {
			return "", false
		}
		if len(c.cfg.AllowedTailnetIdentities) == 0 {
			return identity, true
		}
		for _, allowed := range c.cfg.AllowedTailnetIdentities {
			if strings.EqualFold(identity, allowed) {
				return identity, true
			}
		}
		return "", false
	case config.NavivoxAuthPairingToken, config.NavivoxAuthStaticToken:
		token := bearerToken(r)
		if token == "" {
			token = strings.TrimSpace(r.Header.Get("X-Gormes-Navivox-Token"))
		}
		if token == "" {
			token = webSocketProtocolToken(r)
		}
		if token == "" || c.cfg.Token == "" {
			return "", false
		}
		if hmac.Equal([]byte(token), []byte(c.cfg.Token)) {
			return "token", true
		}
		return "", false
	default:
		return "", false
	}
}

func bearerToken(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
}

func webSocketProtocolToken(r *http.Request) string {
	for _, protocol := range websocket.Subprotocols(r) {
		if !strings.HasPrefix(protocol, navivoxWebSocketTokenProtocolPrefix) {
			continue
		}
		encoded := strings.TrimPrefix(protocol, navivoxWebSocketTokenProtocolPrefix)
		decoded, err := base64.RawURLEncoding.DecodeString(encoded)
		if err != nil {
			return ""
		}
		return string(decoded)
	}
	return ""
}

func firstHeader(r *http.Request, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(r.Header.Get(name)); value != "" {
			return value
		}
	}
	return ""
}

func (c *Channel) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && c.originAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Gormes-Navivox-Token")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (c *Channel) originAllowed(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	for _, allowed := range c.cfg.AllowOrigins {
		if allowed == "*" || strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

type navivoxError struct {
	code    string
	message string
}

func (e navivoxError) Error() string { return e.message }

func codeForNavivoxError(err error) string {
	var ne navivoxError
	if errors.As(err, &ne) && ne.code != "" {
		return ne.code
	}
	return "runtime_error"
}

func statusForNavivoxError(err error) int {
	switch codeForNavivoxError(err) {
	case "bad_request":
		return http.StatusBadRequest
	case "unauthorized":
		return http.StatusUnauthorized
	case "not_found":
		return http.StatusNotFound
	case "timeout":
		return http.StatusGatewayTimeout
	default:
		return http.StatusServiceUnavailable
	}
}

func safeNavivoxError(err error) string {
	var ne navivoxError
	if errors.As(err, &ne) && ne.message != "" {
		return ne.message
	}
	return "Runtime error"
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

func writeNavivoxJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeNavivoxError(w http.ResponseWriter, status int, requestID, code, message string) {
	writeNavivoxJSON(w, status, ServerEvent{
		Type:      "error",
		RequestID: requestID,
		Code:      code,
		Message:   message,
	})
}

func randomID() string {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(raw[:])
}
