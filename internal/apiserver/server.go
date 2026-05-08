package apiserver

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kanban"
	pluginmeta "github.com/TrebuchetDynamics/gormes-agent/internal/plugins"
)

const (
	defaultModelName              = "gormes-agent"
	defaultMaxRequestBytes  int64 = 1_000_000
	maxNormalizedTextLength       = 65_536
	maxContentListSize            = 1_000
)

// Config wires the native API server HTTP surface.
type Config struct {
	APIKey                string
	DashboardSessionToken string
	DashboardBoundHost    string
	ModelName             string
	ProviderName          string
	MaxBodyBytes          int64
	Loop                  TurnLoop
	ResponseStore         *ResponseStore
	RunTTL                time.Duration
	ModelProviders        []DashboardModelProvider
	OAuthProviders        []DashboardOAuthProvider
	PluginInventory       pluginmeta.Inventory
	ChatTransport         ChatTransportStatus
	// DetailedHealth produces the input for the unauthenticated detailed
	// health endpoint. Callers fill it from already-available status reads;
	// when nil the endpoint returns a degraded zero-value snapshot.
	DetailedHealth func() DetailedHealthSnapshotInput
	// CronJobs is the read facade for the native cron job store used by the
	// authenticated read-only cron admin endpoints. *cron.Store satisfies
	// this interface. When nil, the endpoints respond with the shared error
	// envelope and code "cron_store_unavailable".
	CronJobs CronJobReader
	// CronRuns is the read facade for the cron run audit log used by the
	// run-history endpoint. *cron.RunStore satisfies this interface. When
	// nil, run-history responds with code "cron_runs_unavailable".
	CronRuns CronRunReader
	// CronJobMutator is the write facade for the authenticated cron admin
	// mutating endpoints (create / update / delete / pause / resume). When
	// nil, those endpoints respond with code "cron_store_unavailable".
	CronJobMutator CronJobMutator
	// CronTrigger is the trigger seam used by /v1/admin/cron/jobs/{id}/trigger.
	// When nil, the endpoint records trigger_delivery_unavailable and 503.
	CronTrigger CronTriggerHandler
	// CronAdminAuditor receives a redacted audit event for each mutation. It
	// is optional; when nil the endpoints continue to work but emit no audit.
	CronAdminAuditor CronAdminAuditor
	// KanbanStore is the read facade for the kanban task board used by the
	// authenticated dashboard kanban endpoints. *kanban.Store satisfies this
	// interface. When nil, the kanban dashboard panel is disabled and
	// endpoints respond with code "kanban_store_unavailable".
	KanbanStore KanbanStore
	// BuildInfo carries the binary attribution (semver version, git
	// commit, dirty flag, Go toolchain) the dashboard /api/status
	// endpoint surfaces under `build`. Fleet automation querying
	// dashboards across machines uses this to attribute responses to a
	// specific binary. Zero-value is safe — fields default to empty
	// strings and false.
	BuildInfo BuildInfo
}

// BuildInfo is the binary attribution payload surfaced by the
// dashboard /api/status endpoint. cmd/gormes/dashboard.go injects the
// values from cmd/gormes' Version/Commit/Dirty/GoVersion at server
// construction time. Same semantic as the CLI's `--json` build
// envelope — a single source of truth for which binary produced a
// given response.
type BuildInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	GitDirty  bool   `json:"git_dirty"`
	GoVersion string `json:"go_version"`
}

// KanbanStore is the read-only kanban surface consumed by the dashboard.
type KanbanStore interface {
	ListTasks(ctx context.Context, filter kanban.ListFilter) ([]kanban.Task, error)
	GetTask(ctx context.Context, id string) (kanban.Task, error)
}

// ChatTransportStatus describes the dashboard's embedded chat transports
// (PTY-over-WebSocket plus the structured tool-event sidecar). The two
// channels are intentionally separate so a sidecar publication failure can
// be reported without taking the PTY session down with it.
type ChatTransportStatus struct {
	PTYAvailable     bool
	PTYReason        string
	SidecarAvailable bool
	SidecarReason    string
}

// Server exposes the OpenAI-compatible HTTP routes that can be mounted by the
// gateway binary.
type Server struct {
	apiKey                 string
	dashboardSessionToken  string
	dashboardBoundHost     string
	modelName              string
	providerName           string
	maxBodyBytes           int64
	loop                   TurnLoop
	responseStore          *ResponseStore
	runs                   *runRegistry
	modelProviders         []DashboardModelProvider
	oauthProviders         []DashboardOAuthProvider
	pluginInventory        pluginmeta.Inventory
	chatTransport          ChatTransportStatus
	detailedHealth         func() DetailedHealthSnapshotInput
	cronJobs               CronJobReader
	cronRuns               CronRunReader
	cronMutator            CronJobMutator
	cronTrigger            CronTriggerHandler
	cronAuditor            CronAdminAuditor
	kanbanStore            KanbanStore
	buildInfo              BuildInfo
	statusMu               sync.Mutex
	previousResponseMisses int
	now                    func() time.Time
	mux                    *http.ServeMux
	logStore               *LogStore
	sseMu                  sync.Mutex
	sseClients             []chan string
}

// ChatMessage is the normalized text shape passed from HTTP into gateway turns.
type ChatMessage struct {
	Role         string                      `json:"role"`
	Content      string                      `json:"content"`
	ContentParts []hermes.MessageContentPart `json:"content_parts,omitempty"`
	ToolCalls    []ToolCall                  `json:"tool_calls,omitempty"`
	ToolCallID   string                      `json:"tool_call_id,omitempty"`
	Name         string                      `json:"name,omitempty"`
}

// ToolCall is the OpenAI function-call metadata preserved in response chains.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// TurnRequest is the chat-completions request after OpenAI message/content
// normalization and session-handle resolution.
type TurnRequest struct {
	Model            string
	UserMessage      string
	UserContentParts []hermes.MessageContentPart
	History          []ChatMessage
	SystemPrompt     string
	SessionID        string
}

// Usage is the OpenAI-compatible token accounting shape used by both normal
// and streaming chat-completion responses.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// TurnResult is the native turn-loop result consumed by HTTP response writers.
type TurnResult struct {
	Content      string
	SessionID    string
	Usage        Usage
	FinishReason string
	Messages     []ChatMessage
}

// StreamCallbacks receives token deltas from a streaming native turn.
type StreamCallbacks struct {
	OnToken        func(string) error
	OnToolProgress func(ToolProgressEvent) error
}

// ToolProgressEvent is the dashboard-facing progress item emitted by native
// run streams for long-running tool activity.
type ToolProgressEvent struct {
	Name    string `json:"name,omitempty"`
	Preview string `json:"preview,omitempty"`
	Status  string `json:"status,omitempty"`
}

// TurnLoop is the minimal adapter seam between HTTP and the native Gormes turn
// loop. NewKernelTurnLoop provides the production implementation.
type TurnLoop interface {
	RunTurn(ctx context.Context, req TurnRequest) (TurnResult, error)
	StreamTurn(ctx context.Context, req TurnRequest, cb StreamCallbacks) (TurnResult, error)
}

// NewServer constructs the route set without binding a socket.
func NewServer(cfg Config) *Server {
	model := strings.TrimSpace(cfg.ModelName)
	if model == "" {
		model = defaultModelName
	}
	provider := strings.TrimSpace(cfg.ProviderName)
	if provider == "" {
		provider = "native"
	}
	maxBody := cfg.MaxBodyBytes
	if maxBody <= 0 {
		maxBody = defaultMaxRequestBytes
	}
	responseStore := cfg.ResponseStore
	if responseStore == nil {
		responseStore = NewResponseStore(defaultMaxStoredResponses)
	}
	runTTL := cfg.RunTTL
	if runTTL <= 0 {
		runTTL = defaultRunStreamTTL
	}
	s := &Server{
		apiKey:                cfg.APIKey,
		dashboardSessionToken: strings.TrimSpace(cfg.DashboardSessionToken),
		dashboardBoundHost:    strings.TrimSpace(cfg.DashboardBoundHost),
		modelName:             model,
		providerName:          provider,
		maxBodyBytes:          maxBody,
		loop:                  cfg.Loop,
		responseStore:         responseStore,
		runs:                  newRunRegistry(runTTL, time.Now),
		modelProviders:        cloneDashboardModelProviders(cfg.ModelProviders),
		oauthProviders:        cloneDashboardOAuthProviders(cfg.OAuthProviders),
		pluginInventory:       clonePluginInventory(cfg.PluginInventory),
		chatTransport:         cfg.ChatTransport,
		detailedHealth:        cfg.DetailedHealth,
		cronJobs:              cfg.CronJobs,
		cronRuns:              cfg.CronRuns,
		cronMutator:           cfg.CronJobMutator,
		cronTrigger:           cfg.CronTrigger,
		cronAuditor:           cfg.CronAdminAuditor,
		kanbanStore:           cfg.KanbanStore,
		buildInfo:             cfg.BuildInfo,
		now:                   time.Now,
		mux:                   http.NewServeMux(),
		logStore:              NewLogStore(200),
	}
	s.routes()
	return s
}

// NewServerChecked validates startup security before constructing the route set.
// Use this on real listeners so network-exposed deployments cannot boot with
// known placeholder keys.
func NewServerChecked(cfg Config) (*Server, error) {
	if err := ValidateStartupSecurity(cfg); err != nil {
		return nil, err
	}
	return NewServer(cfg), nil
}

// ValidateStartupSecurity rejects placeholder API keys on wildcard or
// network-accessible dashboard hosts. Loopback-only development remains
// allowed so local dashboards can use deterministic fixture keys.
func ValidateStartupSecurity(cfg Config) error {
	key := strings.TrimSpace(cfg.APIKey)
	if key == "" || !weakAPIKeyPlaceholder(key) {
		return nil
	}
	host := strings.TrimSpace(cfg.DashboardBoundHost)
	if host == "" || isLoopbackHost(hostNameOnly(host)) {
		return nil
	}
	return errors.New("api_server_weak_key: refusing network-exposed API server with weak development key")
}

func weakAPIKeyPlaceholder(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "***", "changeme", "your_api_key", "placeholder":
		return true
	default:
		return false
	}
}

// Handler returns an http.Handler suitable for httptest or http.Server.
func (s *Server) Handler() http.Handler {
	return securityHeaders(s.hostGuard(s.mux))
}

func (s *Server) hostGuard(next http.Handler) http.Handler {
	boundHost := strings.TrimSpace(s.dashboardBoundHost)
	if boundHost == "" || boundHost == "0.0.0.0" || boundHost == "::" || boundHost == "[::]" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.hostAllowed(r.Host) {
			next.ServeHTTP(w, r)
			return
		}
		writeOpenAIError(w, http.StatusBadRequest, "Invalid Host header", "invalid_request_error", "", "invalid_host_header")
	})
}

func (s *Server) hostAllowed(hostHeader string) bool {
	bound := hostNameOnly(s.dashboardBoundHost)
	host := hostNameOnly(hostHeader)
	if bound == "" || host == "" {
		return true
	}
	if isLoopbackHost(bound) {
		return isLoopbackHost(host)
	}
	return strings.EqualFold(host, bound)
}

func hostNameOnly(raw string) string {
	host := strings.TrimSpace(raw)
	if host == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.Trim(strings.ToLower(host), "[]")
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) routes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/v1/health", s.handleHealth)
	s.mux.HandleFunc("/health/detailed", s.handleDetailedHealth)
	s.mux.HandleFunc("/v1/health/detailed", s.handleDetailedHealth)
	s.mux.HandleFunc("/v1/models", s.handleModels)
	s.mux.HandleFunc("/v1/capabilities", s.handleCapabilities)
	s.mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	s.mux.HandleFunc("/v1/responses", s.handleResponses)
	s.mux.HandleFunc("/v1/responses/", s.handleResponseByID)
	s.mux.HandleFunc("/v1/runs", s.handleRuns)
	s.mux.HandleFunc("/v1/runs/", s.handleRunEvents)
	s.mux.HandleFunc("/api/status", s.handleDashboardStatus)
	s.mux.HandleFunc("/api/model/info", s.handleDashboardModelInfo)
	s.mux.HandleFunc("/api/model/options", s.handleDashboardModelOptions)
	s.mux.HandleFunc("/api/providers/oauth", s.handleDashboardOAuthProviders)
	s.mux.HandleFunc("/api/dashboard/plugins", s.handleDashboardPlugins)
	s.mux.HandleFunc("/api/sessions", s.handleDashboardSessions)
	s.mux.HandleFunc("/api/sessions/", s.handleDashboardSessionByID)
	s.mux.HandleFunc("/api/kanban", s.handleDashboardKanban)
	s.mux.HandleFunc("/api/kanban/tasks", s.handleDashboardKanbanTasks)
	s.mux.HandleFunc("/api/kanban/tasks/", s.handleDashboardKanbanTaskByID)
	s.mux.HandleFunc("/api/logs", s.handleDashboardLogs)
	s.mux.Handle("/static/", staticHandler())
	s.mux.HandleFunc("/dashboard", s.handleWebDashboard)
	s.mux.HandleFunc("/dashboard/", s.handleWebDashboard)
	s.mux.HandleFunc("/v1/admin/cron/jobs", s.handleCronAdminJobs)
	s.mux.HandleFunc("/v1/admin/cron/jobs/", s.handleCronAdminJobByID)
	s.mux.HandleFunc("/api/jobs", s.handleLegacyAPIJobs)
	s.mux.HandleFunc("/api/jobs/", s.handleLegacyAPIJobByID)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"platform":  "gormes-agent",
		"responses": s.responseHealthStatus(),
		"runs":      s.runHealthStatus(),
	})
}

func (s *Server) handleDetailedHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
		return
	}
	var input DetailedHealthSnapshotInput
	if s.detailedHealth != nil {
		input = s.detailedHealth()
	}
	writeJSON(w, http.StatusOK, DetailedHealthSnapshot(input))
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
		return
	}
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data": []map[string]any{
			{
				"id":         s.modelName,
				"object":     "model",
				"created":    s.now().Unix(),
				"owned_by":   "gormes",
				"permission": []any{},
				"root":       s.modelName,
				"parent":     nil,
			},
		},
	})
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
		return
	}
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object":   "hermes.api_server.capabilities",
		"platform": "gormes-agent",
		"model":    s.modelName,
		"auth": map[string]any{
			"type":     "bearer",
			"required": s.apiKey != "",
		},
		"features": map[string]any{
			"chat_completions":           true,
			"chat_completions_streaming": true,
			"responses_api":              true,
			"responses_streaming":        true,
			"run_submission":             true,
			"run_status":                 true,
			"run_events_sse":             true,
			"run_stop":                   true,
			"tool_progress_events":       true,
			"session_continuity_header":  "X-Hermes-Session-Id",
			"cors":                       false,
		},
		"endpoints": map[string]map[string]string{
			"health":           {"method": "GET", "path": "/health"},
			"health_detailed":  {"method": "GET", "path": "/health/detailed"},
			"models":           {"method": "GET", "path": "/v1/models"},
			"chat_completions": {"method": "POST", "path": "/v1/chat/completions"},
			"responses":        {"method": "POST", "path": "/v1/responses"},
			"runs":             {"method": "POST", "path": "/v1/runs"},
			"run_status":       {"method": "GET", "path": "/v1/runs/{run_id}"},
			"run_events":       {"method": "GET", "path": "/v1/runs/{run_id}/events"},
			"run_stop":         {"method": "POST", "path": "/v1/runs/{run_id}/stop"},
		},
	})
}

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
		return
	}
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if s.loop == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "Native turn loop is not configured", "server_error", "", "turn_loop_unavailable")
		return
	}

	body, err := readLimitedBody(w, r, s.maxBodyBytes)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) || errors.Is(err, errBodyTooLarge) {
			writeOpenAIError(w, http.StatusRequestEntityTooLarge, "Request body too large.", "invalid_request_error", "", "body_too_large")
			return
		}
		writeOpenAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "", "invalid_request_body")
		return
	}

	var bodyReq chatCompletionRequest
	if err := json.Unmarshal(body, &bodyReq); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "Invalid JSON in request body", "invalid_request_error", "", "invalid_json")
		return
	}
	if len(bodyReq.Messages) == 0 {
		writeOpenAIError(w, http.StatusBadRequest, "Missing or invalid 'messages' field", "invalid_request_error", "messages", "invalid_messages")
		return
	}

	turnReq, errResp := s.buildTurnRequest(r, bodyReq)
	if errResp != nil {
		writeOpenAIError(w, errResp.status, errResp.message, "invalid_request_error", errResp.param, errResp.code)
		return
	}
	model := strings.TrimSpace(bodyReq.Model)
	if model == "" {
		model = s.modelName
	}
	turnReq.Model = model

	completionID := "chatcmpl-" + randomHexFromTime(s.now())
	created := s.now().Unix()
	if bodyReq.Stream {
		s.writeStreamingChatCompletion(w, r, completionID, created, model, turnReq)
		return
	}

	result, err := s.loop.RunTurn(r.Context(), turnReq)
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "Internal server error: "+err.Error(), "server_error", "", "turn_failed")
		return
	}
	sessionID := result.SessionID
	if sessionID == "" {
		sessionID = turnReq.SessionID
	}
	if sessionID != "" {
		w.Header().Set("X-Hermes-Session-Id", sessionID)
	}
	writeJSON(w, http.StatusOK, chatCompletionResponse(completionID, created, model, result))
}

func (s *Server) writeStreamingChatCompletion(w http.ResponseWriter, r *http.Request, completionID string, created int64, model string, turnReq TurnRequest) {
	sessionID := turnReq.SessionID
	if sessionID != "" {
		w.Header().Set("X-Hermes-Session-Id", sessionID)
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	writeSSEData(w, chatCompletionChunk{
		ID:      completionID,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   model,
		Choices: []chatCompletionChunkChoice{{
			Index: 0,
			Delta: map[string]string{"role": "assistant"},
		}},
	})
	flush(w)

	result, err := s.loop.StreamTurn(r.Context(), turnReq, StreamCallbacks{
		OnToken: func(token string) error {
			writeSSEData(w, chatCompletionChunk{
				ID:      completionID,
				Object:  "chat.completion.chunk",
				Created: created,
				Model:   model,
				Choices: []chatCompletionChunkChoice{{
					Index: 0,
					Delta: map[string]string{"content": token},
				}},
			})
			flush(w)
			return nil
		},
	})
	if err != nil {
		writeSSEEvent(w, "error", openAIErrorEnvelope("Internal server error: "+err.Error(), "server_error", "", "stream_failed"))
		writeSSEDone(w)
		flush(w)
		return
	}
	if result.SessionID != "" && sessionID == "" {
		// Header-phase streaming cannot publish a late provider session handle,
		// but keeping this branch documents the intended continuity fallback.
		sessionID = result.SessionID
	}
	writeSSEData(w, chatCompletionChunk{
		ID:      completionID,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   model,
		Choices: []chatCompletionChunkChoice{{
			Index:        0,
			Delta:        map[string]string{},
			FinishReason: stringPtr("stop"),
		}},
		Usage: usagePayload(result.Usage),
	})
	writeSSEDone(w)
	flush(w)
}

func (s *Server) authorized(r *http.Request) bool {
	if s.apiKey == "" {
		return true
	}
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if hmac.Equal([]byte(token), []byte(s.apiKey)) {
			return true
		}
	}
	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		return hmac.Equal([]byte(key), []byte(s.apiKey))
	}
	return false
}

func (s *Server) dashboardAuthorized(r *http.Request) bool {
	token := strings.TrimSpace(s.dashboardSessionToken)
	if token == "" {
		return s.authorized(r)
	}
	if got := strings.TrimSpace(r.Header.Get("X-Hermes-Session-Token")); got != "" && hmac.Equal([]byte(got), []byte(token)) {
		return true
	}
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(auth, "Bearer ") {
		got := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if hmac.Equal([]byte(got), []byte(token)) {
			return true
		}
	}
	return s.apiKey != "" && s.authorized(r)
}

type chatCompletionRequest struct {
	Model    string            `json:"model"`
	Messages []incomingMessage `json:"messages"`
	Stream   bool              `json:"stream"`
}

type incomingMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type requestError struct {
	status  int
	message string
	param   string
	code    string
}

func (s *Server) buildTurnRequest(r *http.Request, req chatCompletionRequest) (TurnRequest, *requestError) {
	var (
		systemParts  []string
		conversation []ChatMessage
		firstUser    string
	)
	for idx, msg := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		content, err := normalizeChatContentForTurn(msg.Content)
		if err != nil {
			return TurnRequest{}, &requestError{
				status:  http.StatusBadRequest,
				message: err.message,
				param:   fmt.Sprintf("messages[%d].content", idx),
				code:    err.code,
			}
		}
		switch role {
		case "system", "developer":
			if strings.TrimSpace(content.Text) != "" {
				systemParts = append(systemParts, content.Text)
			}
		case "user", "assistant":
			conversation = append(conversation, ChatMessage{Role: role, Content: content.Text, ContentParts: content.Parts})
			if role == "user" && firstUser == "" {
				firstUser = chatContentFingerprint(conversation[len(conversation)-1])
			}
		}
	}

	lastUser := -1
	for i := len(conversation) - 1; i >= 0; i-- {
		if conversation[i].Role == "user" {
			lastUser = i
			break
		}
	}
	if lastUser < 0 || !hasVisibleChatMessage(conversation[lastUser]) {
		return TurnRequest{}, &requestError{
			status:  http.StatusBadRequest,
			message: "No user message found in messages",
			code:    "missing_user_message",
		}
	}

	systemPrompt := strings.Join(systemParts, "\n")
	sessionID := strings.TrimSpace(r.Header.Get("X-Hermes-Session-Id"))
	if strings.ContainsAny(sessionID, "\r\n\x00") {
		return TurnRequest{}, &requestError{
			status:  http.StatusBadRequest,
			message: "Invalid session ID",
			param:   "X-Hermes-Session-Id",
			code:    "invalid_session_id",
		}
	}
	if sessionID == "" {
		sessionID = deriveChatSessionID(systemPrompt, firstUser)
	}

	return TurnRequest{
		UserMessage:      conversation[lastUser].Content,
		UserContentParts: cloneContentParts(conversation[lastUser].ContentParts),
		History:          append([]ChatMessage(nil), conversation[:lastUser]...),
		SystemPrompt:     systemPrompt,
		SessionID:        sessionID,
	}, nil
}

type normalizedChatContent struct {
	Text  string
	Parts []hermes.MessageContentPart
}

type contentNormalizeError struct {
	code    string
	message string
}

func normalizeChatContent(content any) (string, *contentNormalizeError) {
	normalized, err := normalizeChatContentForTurn(content)
	return normalized.Text, err
}

func normalizeChatContentForTurn(content any) (normalizedChatContent, *contentNormalizeError) {
	return normalizeChatContentDepth(content, 0)
}

func normalizeChatContentDepth(content any, depth int) (normalizedChatContent, *contentNormalizeError) {
	if depth > 10 || content == nil {
		return normalizedChatContent{}, nil
	}
	switch v := content.(type) {
	case string:
		return normalizedChatContent{Text: truncateText(v)}, nil
	case []any:
		limit := len(v)
		if limit > maxContentListSize {
			limit = maxContentListSize
		}
		textParts := make([]string, 0, limit)
		contentParts := make([]hermes.MessageContentPart, 0, limit)
		total := 0
		for _, item := range v[:limit] {
			var normalized normalizedChatContent
			switch p := item.(type) {
			case string:
				normalized = normalizedChatContent{Text: p}
			case []any:
				nested, err := normalizeChatContentDepth(p, depth+1)
				if err != nil {
					return normalizedChatContent{}, err
				}
				normalized = nested
			case map[string]any:
				partContent, err := normalizeContentPart(p)
				if err != nil {
					return normalizedChatContent{}, err
				}
				normalized = partContent
			default:
				continue
			}
			if len(normalized.Parts) > 0 {
				contentParts = append(contentParts, normalized.Parts...)
			} else if normalized.Text != "" {
				contentParts = append(contentParts, hermes.MessageContentPart{Type: "text", Text: truncateText(normalized.Text)})
			}
			if normalized.Text != "" {
				trimmed := truncateText(normalized.Text)
				textParts = append(textParts, trimmed)
				total += len(trimmed)
				if total >= maxNormalizedTextLength {
					break
				}
			}
		}
		text := truncateText(strings.Join(textParts, "\n"))
		if hasImageContentPart(contentParts) {
			return normalizedChatContent{Text: text, Parts: normalizeTextParts(contentParts)}, nil
		}
		return normalizedChatContent{Text: text}, nil
	default:
		return normalizedChatContent{Text: truncateText(fmt.Sprint(v))}, nil
	}
}

func normalizeContentPart(part map[string]any) (normalizedChatContent, *contentNormalizeError) {
	rawType, ok := part["type"]
	partType := ""
	if ok && rawType != nil {
		partType = strings.ToLower(strings.TrimSpace(fmt.Sprint(rawType)))
	}
	switch partType {
	case "text", "input_text", "output_text":
		text, ok := part["text"]
		if !ok || text == nil {
			return normalizedChatContent{}, nil
		}
		return normalizedChatContent{Text: fmt.Sprint(text), Parts: []hermes.MessageContentPart{{Type: "text", Text: fmt.Sprint(text)}}}, nil
	case "image_url", "input_image":
		image, err := normalizeImageContentPart(partType, part)
		if err != nil {
			return normalizedChatContent{}, err
		}
		return normalizedChatContent{Parts: []hermes.MessageContentPart{image}}, nil
	case "file", "input_file":
		return normalizedChatContent{}, &contentNormalizeError{
			code:    "unsupported_content_type",
			message: "Uploaded files and document inputs are not supported on this endpoint.",
		}
	case "":
		return normalizedChatContent{}, &contentNormalizeError{
			code:    "invalid_content_part",
			message: "Content parts must include a type.",
		}
	default:
		return normalizedChatContent{}, &contentNormalizeError{
			code:    "unsupported_content_type",
			message: fmt.Sprintf("Unsupported content part type %q. Only text and image_url/input_image parts are supported.", part["type"]),
		}
	}
}

func normalizeImageContentPart(partType string, part map[string]any) (hermes.MessageContentPart, *contentNormalizeError) {
	imageURL, detail := "", ""
	switch raw := part["image_url"].(type) {
	case string:
		imageURL = strings.TrimSpace(raw)
	case map[string]any:
		imageURL = strings.TrimSpace(fmt.Sprint(raw["url"]))
		if imageURL == "<nil>" {
			imageURL = ""
		}
		detail = strings.TrimSpace(fmt.Sprint(raw["detail"]))
		if detail == "<nil>" {
			detail = ""
		}
	default:
		if partType == "input_image" {
			imageURL = strings.TrimSpace(fmt.Sprint(part["image_url"]))
			if imageURL == "<nil>" {
				imageURL = ""
			}
		}
	}
	if strings.TrimSpace(fmt.Sprint(part["detail"])) != "" && strings.TrimSpace(fmt.Sprint(part["detail"])) != "<nil>" {
		detail = strings.TrimSpace(fmt.Sprint(part["detail"]))
	}
	if imageURL == "" {
		return hermes.MessageContentPart{}, &contentNormalizeError{
			code:    "invalid_image_url",
			message: "Image content parts must include an image_url.url value.",
		}
	}
	if strings.HasPrefix(imageURL, "data:") && !strings.HasPrefix(strings.ToLower(imageURL), "data:image/") {
		return hermes.MessageContentPart{}, &contentNormalizeError{
			code:    "unsupported_content_type",
			message: "Only image data URLs are supported on this endpoint.",
		}
	}
	if strings.Contains(imageURL, "://") && !(strings.HasPrefix(strings.ToLower(imageURL), "http://") || strings.HasPrefix(strings.ToLower(imageURL), "https://")) {
		return hermes.MessageContentPart{}, &contentNormalizeError{
			code:    "invalid_image_url",
			message: "Image URLs must use http, https, or data:image schemes.",
		}
	}
	return hermes.MessageContentPart{Type: "image_url", ImageURL: imageURL, Detail: detail}, nil
}

func normalizeTextParts(parts []hermes.MessageContentPart) []hermes.MessageContentPart {
	out := make([]hermes.MessageContentPart, 0, len(parts))
	for _, part := range parts {
		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case "text", "input_text", "output_text":
			if part.Text != "" {
				out = append(out, hermes.MessageContentPart{Type: "text", Text: truncateText(part.Text)})
			}
		case "image_url", "input_image":
			if part.ImageURL != "" {
				out = append(out, hermes.MessageContentPart{Type: "image_url", ImageURL: part.ImageURL, Detail: strings.TrimSpace(part.Detail)})
			}
		}
	}
	return out
}

func hasImageContentPart(parts []hermes.MessageContentPart) bool {
	for _, part := range parts {
		partType := strings.ToLower(strings.TrimSpace(part.Type))
		if (partType == "image_url" || partType == "input_image") && strings.TrimSpace(part.ImageURL) != "" {
			return true
		}
	}
	return false
}

func truncateText(s string) string {
	if len(s) <= maxNormalizedTextLength {
		return s
	}
	return s[:maxNormalizedTextLength]
}

func hasVisibleText(s string) bool {
	return strings.TrimSpace(s) != ""
}

func hasVisibleChatMessage(msg ChatMessage) bool {
	if hasVisibleText(msg.Content) {
		return true
	}
	return hasImageContentPart(msg.ContentParts)
}

func chatContentFingerprint(msg ChatMessage) string {
	if strings.TrimSpace(msg.Content) != "" {
		return msg.Content
	}
	var parts []string
	for _, part := range msg.ContentParts {
		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case "text", "input_text", "output_text":
			if strings.TrimSpace(part.Text) != "" {
				parts = append(parts, "text:"+part.Text)
			}
		case "image_url", "input_image":
			if strings.TrimSpace(part.ImageURL) != "" {
				parts = append(parts, "image:"+part.ImageURL)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func cloneContentParts(parts []hermes.MessageContentPart) []hermes.MessageContentPart {
	if len(parts) == 0 {
		return nil
	}
	return append([]hermes.MessageContentPart(nil), parts...)
}

func deriveChatSessionID(systemPrompt, firstUserMessage string) string {
	sum := sha256.Sum256([]byte(systemPrompt + "\n" + firstUserMessage))
	return "api-" + hex.EncodeToString(sum[:])[:16]
}

func randomHexFromTime(t time.Time) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d", t.UnixNano())))
	return hex.EncodeToString(sum[:])[:29]
}

type chatCompletionChunk struct {
	ID      string                      `json:"id"`
	Object  string                      `json:"object"`
	Created int64                       `json:"created"`
	Model   string                      `json:"model"`
	Choices []chatCompletionChunkChoice `json:"choices"`
	Usage   map[string]int              `json:"usage,omitempty"`
}

type chatCompletionChunkChoice struct {
	Index        int               `json:"index"`
	Delta        map[string]string `json:"delta"`
	Logprobs     any               `json:"logprobs"`
	FinishReason *string           `json:"finish_reason"`
}

func chatCompletionResponse(id string, created int64, model string, result TurnResult) map[string]any {
	finish := strings.TrimSpace(result.FinishReason)
	if finish == "" {
		finish = "stop"
	}
	return map[string]any{
		"id":      id,
		"object":  "chat.completion",
		"created": created,
		"model":   model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": result.Content,
				},
				"logprobs":      nil,
				"finish_reason": finish,
			},
		},
		"usage": usagePayload(result.Usage),
	}
}

func usagePayload(u Usage) map[string]int {
	return map[string]int{
		"prompt_tokens":     u.PromptTokens,
		"completion_tokens": u.CompletionTokens,
		"total_tokens":      u.TotalTokens,
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeOpenAIError(w http.ResponseWriter, status int, message, errType, param, code string) {
	writeJSON(w, status, openAIErrorEnvelope(message, errType, param, code))
}

func openAIErrorEnvelope(message, errType, param, code string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    errType,
			"param":   nullableString(param),
			"code":    nullableString(code),
		},
	}
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func stringPtr(s string) *string { return &s }

func writeSSEData(w http.ResponseWriter, body any) error {
	raw, _ := json.Marshal(body)
	_, err := fmt.Fprintf(w, "data: %s\n\n", raw)
	return err
}

func writeSSEEvent(w http.ResponseWriter, event string, body any) error {
	raw, _ := json.Marshal(body)
	_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, raw)
	return err
}

func writeSSEDone(w http.ResponseWriter) error {
	_, err := io.WriteString(w, "data: [DONE]\n\n")
	return err
}

func flush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

var errBodyTooLarge = errors.New("api server: request body too large")

func readLimitedBody(w http.ResponseWriter, r *http.Request, maxBytes int64) ([]byte, error) {
	if r.ContentLength > maxBytes {
		return nil, errBodyTooLarge
	}
	reader := http.MaxBytesReader(w, r.Body, maxBytes)
	defer reader.Close()
	return io.ReadAll(reader)
}
