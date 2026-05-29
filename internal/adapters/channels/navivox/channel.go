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
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/profileseed"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/network/vpnhost"
	sessionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/session"
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

	singleUsePairingStream bool
	pairingStreamReserved  bool
	pairingStreamConsumed  bool

	profileContacts    map[string]ProfileContact
	loadContacts       func(context.Context) ([]ProfileContact, error)
	profileRouting     config.NavivoxProfileRoutingReport
	configAdmin        configAdminBackend
	voiceProfiles      voiceProfileBackend
	runRecords         map[string]*sessionpkg.NavivoxRunRecord
	latestRunBySession map[string]string
}

type ChannelOption func(*Channel)

func WithProfileRouting(report config.NavivoxProfileRoutingReport) ChannelOption {
	return func(c *Channel) {
		c.profileRouting = report
	}
}

func WithSingleUsePairingStream() ChannelOption {
	return func(c *Channel) {
		c.singleUsePairingStream = true
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
		cfg:                cfg,
		log:                log,
		now:                func() time.Time { return time.Now().UTC() },
		newID:              randomID,
		sessions:           map[string]*sessionState{},
		clients:            map[*client]struct{}{},
		profileContacts:    map[string]ProfileContact{},
		configAdmin:        defaultConfigAdminBackend(),
		voiceProfiles:      defaultVoiceProfileBackend(),
		runRecords:         map[string]*sessionpkg.NavivoxRunRecord{},
		latestRunBySession: map[string]string{},
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
	mux.HandleFunc("/v1/navivox/capabilities", c.withAuth(c.handleCapabilities))
	mux.HandleFunc("/v1/navivox/profile-contacts", c.withAuth(c.handleProfileContacts))
	mux.HandleFunc("/v1/navivox/profile-routing", c.withAuth(c.handleProfileRouting))
	mux.HandleFunc("/v1/navivox/profile-seed", c.withAuth(c.handleProfileSeed))
	mux.HandleFunc("/v1/navivox/config-admin", c.withAuth(c.handleConfigAdmin))
	mux.HandleFunc("/v1/navivox/config-admin/", c.withAuth(c.handleConfigAdmin))
	mux.HandleFunc("/v1/navivox/voice-profiles", c.withAuth(c.handleVoiceProfiles))
	mux.HandleFunc("/v1/navivox/voice-profiles/", c.withAuth(c.handleVoiceProfiles))
	mux.HandleFunc("/v1/navivox/run-records/", c.withAuth(c.handleRunRecord))
	mux.HandleFunc("/v1/navivox/memory/overview", c.withAuth(c.handleMemoryOverview))
	mux.HandleFunc("/v1/navivox/sessions", c.withAuth(c.handleSessions))
	mux.HandleFunc("/v1/navivox/sessions/", c.withAuth(c.handleSessionByID))
	mux.HandleFunc("/v1/navivox/turn", c.withAuth(c.handleTurn(inbox)))
	mux.HandleFunc("/v1/navivox/stream", c.handleStream(inbox))
	return c.cors(mux)
}

func (c *Channel) Send(ctx context.Context, chatID, text string) (string, error) {
	msgID := c.newID()
	c.recordAssistantAndComplete(chatID, text)
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
	case gateway.ToolProgressUpdated:
		eventType = "tool_call_updated"
	case gateway.ToolProgressFinished, gateway.ToolProgressFailed:
		eventType = "tool_call_finished"
	}
	metadata := safeNavivoxToolMetadata(progress.Metadata)
	c.recordToolProgress(chatID, toolCallID, progress.ToolName, status, progress.Summary, metadata)
	c.broadcast(chatID, ServerEvent{
		Type:       eventType,
		SessionID:  chatID,
		ToolName:   safeNavivoxToolName(progress.ToolName),
		ToolCallID: toolCallID,
		Status:     status,
		Message:    safeNavivoxToolSummary(progress.Summary),
		Metadata:   metadata,
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
	c.recordAssistantAndComplete(chatID, text)
	c.updateProfileContactForSession(chatID, text, "assistant", ProfileContactTurnIdle)
	c.broadcast(chatID, ServerEvent{Type: "assistant_message", SessionID: chatID, Text: text})
	c.broadcast(chatID, ServerEvent{Type: "done", SessionID: chatID})
	return nil
}

func (c *Channel) recordRunStartLocked(sessionID, requestID, text string, metadata map[string]any) {
	if c.runRecords == nil {
		c.runRecords = map[string]*sessionpkg.NavivoxRunRecord{}
	}
	if c.latestRunBySession == nil {
		c.latestRunBySession = map[string]string{}
	}
	record := sessionpkg.NewNavivoxRunRecord(requestID, sessionID, text, metadata, c.now())
	c.runRecords[record.RunID] = &record
	c.latestRunBySession[sessionID] = record.RunID
}

func (c *Channel) recordToolProgress(sessionID, toolCallID, toolName, status, summary string, metadata map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	record := c.latestRunRecordLocked(sessionID)
	if record == nil {
		return
	}
	record.AppendToolEvent(toolCallID, safeNavivoxToolName(toolName), status, safeNavivoxToolSummary(summary), metadata, c.now())
}

func (c *Channel) recordAssistantAndComplete(sessionID, text string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	record := c.latestRunRecordLocked(sessionID)
	if record == nil {
		return
	}
	now := c.now()
	record.AppendAssistant(text, now)
	record.Complete(now)
}

func (c *Channel) latestRunRecordLocked(sessionID string) *sessionpkg.NavivoxRunRecord {
	if c.latestRunBySession == nil || c.runRecords == nil {
		return nil
	}
	runID := c.latestRunBySession[sessionID]
	if runID == "" {
		return nil
	}
	return c.runRecords[runID]
}

func (c *Channel) lookupRunRecord(id string) (sessionpkg.NavivoxRunRecord, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runRecords == nil {
		return sessionpkg.NavivoxRunRecord{}, false
	}
	if record := c.runRecords[id]; record != nil {
		return record.Clone(), true
	}
	if runID := c.latestRunBySession[id]; runID != "" {
		if record := c.runRecords[runID]; record != nil {
			return record.Clone(), true
		}
	}
	return sessionpkg.NavivoxRunRecord{}, false
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
		"profile_routing":     c.profileRouting,
		"protocol_version":    navivoxWebSocketProtocol,
		"websocket_protocols": []string{navivoxWebSocketProtocol},
		"capabilities_url":    "/v1/navivox/capabilities",
		"capabilities":        navivoxCapabilityNames(),
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

type profileSeedRequest struct {
	Seed       string   `json:"seed"`
	Apply      bool     `json:"apply"`
	Workspaces []string `json:"workspaces"`
	// WorkspaceRoots is accepted as a clearer client alias. It is still treated
	// as explicit operator confirmation; inferred seed suggestions are never
	// written to config.
	WorkspaceRoots []string `json:"workspace_roots"`
}

type profileSeedResponse struct {
	Action         string            `json:"action"`
	Status         string            `json:"status"`
	Applied        bool              `json:"applied,omitempty"`
	ProfileID      string            `json:"profile_id,omitempty"`
	Root           string            `json:"root,omitempty"`
	WorkspaceCount int               `json:"workspace_count,omitempty"`
	Draft          profileseed.Draft `json:"draft"`
	Contact        *ProfileContact   `json:"contact,omitempty"`
}

func (c *Channel) handleProfileSeed(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodPost {
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
		return
	}
	var req profileSeedRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeNavivoxError(w, http.StatusBadRequest, "", "bad_request", "Invalid profile seed request")
		return
	}
	seed := strings.TrimSpace(req.Seed)
	if seed == "" {
		writeNavivoxError(w, http.StatusBadRequest, "", "bad_request", "Seed is required")
		return
	}
	draft, err := profileseed.NewDraft(seed, profileseed.DraftOptions{})
	if err != nil {
		writeNavivoxError(w, http.StatusBadRequest, "", "profile_seed_invalid", "Profile seed is unsafe or invalid")
		return
	}
	if !req.Apply {
		writeNavivoxJSON(w, http.StatusOK, profileSeedResponse{Action: "profile_seed_draft", Status: "draft", Draft: draft})
		return
	}
	workspaces := append([]string{}, req.Workspaces...)
	workspaces = append(workspaces, req.WorkspaceRoots...)
	result, err := profileseed.Apply(seed, profileseed.ApplyOptions{ConfirmedWorkspaces: workspaces})
	if err != nil {
		writeNavivoxError(w, http.StatusConflict, "", "profile_seed_apply_failed", "Profile seed could not be applied")
		return
	}
	contact := c.profileContactFromRoot(result.ProfileID, result.Root)
	c.broadcastProfileContact(contact)
	writeNavivoxJSON(w, http.StatusOK, profileSeedResponse{
		Action:         "profile_seed_applied",
		Status:         "applied",
		Applied:        result.Applied,
		ProfileID:      result.ProfileID,
		Root:           redactNavivoxPathTail(result.Root),
		WorkspaceCount: result.WorkspaceCount,
		Draft:          result.Draft,
		Contact:        &contact,
	})
}

func redactNavivoxPathTail(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	parts := strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' })
	if len(parts) == 0 {
		return "..."
	}
	return ".../" + parts[len(parts)-1]
}

func (c *Channel) handleRunRecord(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodGet {
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/navivox/run-records/")
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") {
		writeNavivoxError(w, http.StatusNotFound, "", "not_found", "Run record not found")
		return
	}
	record, ok := c.lookupRunRecord(id)
	if !ok {
		writeNavivoxError(w, http.StatusNotFound, "", "not_found", "Run record not found")
		return
	}
	writeNavivoxJSON(w, http.StatusOK, map[string]any{"run_record": record})
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
