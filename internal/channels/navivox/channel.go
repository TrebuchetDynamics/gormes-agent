package navivox

import (
	"context"
	"crypto/rand"
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

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/network/vpnhost"
)

const PlatformName = "navivox"

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
	profileRouting  config.NavivoxProfileRoutingReport
}

type ChannelOption func(*Channel)

func WithProfileRouting(report config.NavivoxProfileRoutingReport) ChannelOption {
	return func(c *Channel) {
		c.profileRouting = report
	}
}

func NewChannel(cfg config.NavivoxCfg, log *slog.Logger, opts ...ChannelOption) (*Channel, error) {
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
	ch := &Channel{
		cfg:             cfg,
		log:             log,
		now:             func() time.Time { return time.Now().UTC() },
		newID:           randomID,
		sessions:        map[string]*sessionState{},
		clients:         map[*client]struct{}{},
		profileContacts: map[string]ProfileContact{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(ch)
		}
	}
	return ch, nil
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
	mux.HandleFunc("/v1/navivox/profile-routing", c.withAuth(c.handleProfileRouting))
	mux.HandleFunc("/v1/navivox/memory/overview", c.withAuth(c.handleMemoryOverview))
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

func (c *Channel) SendSafetyWarning(ctx context.Context, chatID string, warning SafetyEvent) (string, error) {
	id := strings.TrimSpace(warning.ID)
	if id == "" {
		id = c.newID()
	}
	severity := strings.TrimSpace(warning.Severity)
	if severity == "" {
		severity = "warning"
	}
	c.broadcast(chatID, ServerEvent{
		Type:      "safety_warning",
		SessionID: chatID,
		SafetyID:  id,
		Severity:  severity,
		Message:   safeNavivoxToolSummary(warning.Message),
		Risk:      safeNavivoxToolSummary(warning.Risk),
		Metadata:  safeNavivoxToolMetadata(warning.Metadata),
	})
	return id, ctx.Err()
}

func (c *Channel) SendApprovalRequired(ctx context.Context, chatID string, approval ApprovalEvent) (string, error) {
	id := strings.TrimSpace(approval.ID)
	if id == "" {
		id = c.newID()
	}
	c.broadcast(chatID, ServerEvent{
		Type:       "approval_required",
		SessionID:  chatID,
		ApprovalID: id,
		ToolCallID: strings.TrimSpace(approval.ToolCallID),
		Message:    safeNavivoxToolSummary(approval.Prompt),
		Risk:       safeNavivoxToolSummary(approval.Risk),
		Metadata:   safeNavivoxToolMetadata(approval.Metadata),
	})
	return id, ctx.Err()
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
		"enabled":             c.cfg.Enabled,
		"bind_host":           c.cfg.BindHost,
		"port":                c.cfg.Port,
		"exposure_mode":       c.cfg.ExposureMode,
		"auth_mode":           c.cfg.AuthMode,
		"protocol_version":    navivoxWebSocketProtocol,
		"websocket_protocols": []string{navivoxWebSocketProtocol, navivoxLegacyWebSocketProtocol},
		"capabilities": []string{
			"profile_contacts",
			"profile_routing",
			"memory_overview",
			"stream_turns",
			"tool_progress",
			"safety_warnings",
			"approval_required",
			"turn_control",
			"setup_handoff",
		},
		"setup_handoff": map[string]any{
			"recommended_path":          "navivox",
			"title":                     "Continue setup in Navivox",
			"description":               "Pair your Android app and continue setup there.",
			"steps":                     []string{"Choose provider", "Choose model", "Confirm workspace", "Enable channels"},
			"mutation_policy":           "read_only_handoff",
			"entry_screen":              "setup.provider",
			"bridge_keepalive_required": true,
			"bridge_lifecycle":          "termux_pair_command",
			"sections": []map[string]string{
				{"id": "provider", "title": "Choose provider", "navivox_screen": "setup.provider", "fallback_cli_command": "gormes setup provider"},
				{"id": "model", "title": "Choose model", "navivox_screen": "setup.model", "fallback_cli_command": "gormes setup model"},
				{"id": "workspace", "title": "Confirm workspace", "navivox_screen": "setup.workspace", "fallback_cli_command": "gormes setup workspace"},
				{"id": "channels", "title": "Enable channels", "navivox_screen": "setup.channels", "fallback_cli_command": "gormes setup gateway"},
			},
			"cli_setup_command": "gormes setup",
		},
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

func (c *Channel) handleProfileRouting(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodGet {
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
		return
	}
	writeNavivoxJSON(w, http.StatusOK, c.profileRouting)
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
