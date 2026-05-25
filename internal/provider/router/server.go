package router

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// ChatProvider is the fakeable upstream boundary used by the local Router HTTP
// service. Production wiring can adapt Gormes provider clients here without the
// HTTP handler becoming a blind reverse proxy.
type ChatProvider interface {
	ChatCompletion(context.Context, Route, ChatCompletionRequest) (ChatCompletionResult, error)
}

type StreamChatProvider interface {
	StreamChatCompletion(context.Context, Route, ChatCompletionRequest) (ChatStreamResult, error)
}

type ProviderError struct {
	Class   string
	Message string
}

func (e ProviderError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if strings.TrimSpace(e.Class) != "" {
		return strings.TrimSpace(e.Class)
	}
	return "provider_error"
}

type HandlerOptions struct {
	LookupEnv    func(string) (string, bool)
	Provider     ChatProvider
	Probe        ProbeFunc
	ProbeContext context.Context
	NowUnix      func() int64
	Logf         func(string, ...any)
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResult struct {
	ID           string
	Content      string
	FinishReason string
	Usage        *Usage
}

type ChatStreamResult struct {
	Chunks []string
	Err    error
	Usage  *Usage
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Server struct {
	cfg      config.Config
	model    ReadModel
	registry Registry
	keys     []string
	provider ChatProvider
	nowUnix  func() int64
	logf     func(string, ...any)

	mu       sync.Mutex
	counters routerCounters
}

type routerCounters struct {
	Attempts       int    `json:"attempts"`
	Successes      int    `json:"successes"`
	Failures       int    `json:"failures"`
	Fallbacks      int    `json:"fallbacks"`
	LastErrorClass string `json:"last_error_class,omitempty"`
	Usage          Usage  `json:"usage"`
}

func NewHandler(cfg config.Config, opts HandlerOptions) http.Handler {
	lookupEnv := opts.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	nowUnix := opts.NowUnix
	if nowUnix == nil {
		nowUnix = func() int64 { return time.Now().Unix() }
	}
	model := BuildReadModel(cfg, Options{LookupEnv: lookupEnv, Probe: opts.Probe, ProbeContext: opts.ProbeContext})
	s := &Server{
		cfg:      cfg,
		model:    model,
		registry: NewRegistry(model),
		keys:     routerInboundKeys(cfg.Router, lookupEnv),
		provider: opts.Provider,
		nowUnix:  nowUnix,
		logf:     opts.Logf,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/v1/models", s.withAuth(s.handleModels))
	mux.HandleFunc("/v1/chat/completions", s.withAuth(s.handleChatCompletions))
	mux.HandleFunc("/v1/status", s.withAuth(s.handleStatus))
	mux.HandleFunc("/router/status", s.withAuth(s.handleStatus))
	return mux
}

func routerInboundKeys(cfg config.RouterCfg, lookupEnv func(string) (string, bool)) []string {
	keys := compactStrings(cfg.APIKeys)
	if env := strings.TrimSpace(cfg.APIKeyEnv); env != "" {
		if value, ok := lookupEnv(env); ok && strings.TrimSpace(value) != "" {
			keys = append(keys, strings.TrimSpace(value))
		}
	}
	return keys
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeRouterJSON(w, http.StatusOK, map[string]any{
		"ok":          s.model.Enabled && s.model.Status.State != RouterStatusInvalidRoute,
		"service":     "gormes-router",
		"state":       s.model.Status.State,
		"route_count": s.model.Status.RouteCount,
	})
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authorized(r) {
			writeRouterError(w, http.StatusUnauthorized, "router_unauthorized")
			return
		}
		next(w, r)
	}
}

func (s *Server) authorized(r *http.Request) bool {
	if len(s.keys) == 0 {
		return false
	}
	provided := routerAuthValues(r)
	for _, candidate := range provided {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		for _, key := range s.keys {
			if constantTimeStringEqual(candidate, key) {
				return true
			}
		}
	}
	return false
}

func routerAuthValues(r *http.Request) []string {
	values := []string{}
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); auth != "" {
		parts := strings.Fields(auth)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			values = append(values, parts[1])
		}
	}
	if key := strings.TrimSpace(r.Header.Get("X-Api-Key")); key != "" {
		values = append(values, key)
	}
	return values
}

func constantTimeStringEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRouterError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	data := []openAIModel{}
	for _, model := range s.registry.Models() {
		if !routeStatusServable(model.Status) {
			continue
		}
		data = append(data, openAIModel{ID: model.ID, Object: "model", OwnedBy: model.Provider})
	}
	writeRouterJSON(w, http.StatusOK, openAIModelList{Object: "list", Data: data})
}

type openAIModelList struct {
	Object string        `json:"object"`
	Data   []openAIModel `json:"data"`
}

type openAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeRouterError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if s.provider == nil {
		writeRouterError(w, http.StatusServiceUnavailable, "router_provider_unavailable")
		return
	}
	var req ChatCompletionRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		writeRouterError(w, http.StatusBadRequest, "malformed_request")
		return
	}
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		writeRouterError(w, http.StatusBadRequest, "model_required")
		return
	}
	route, ok := s.resolveServableRoute(req.Model)
	if !ok {
		writeRouterError(w, http.StatusNotFound, "model_not_found")
		return
	}
	if req.Stream {
		s.handleStreamingChat(w, r, route, req)
		return
	}
	result, _, errClass, err := s.chatCompletionWithFallback(r.Context(), route, req)
	if err != nil {
		writeRouterError(w, http.StatusBadGateway, errClass)
		return
	}
	finish := strings.TrimSpace(result.FinishReason)
	if finish == "" {
		finish = "stop"
	}
	id := strings.TrimSpace(result.ID)
	if id == "" {
		id = "chatcmpl-gormes-router"
	}
	writeRouterJSON(w, http.StatusOK, openAIChatCompletionResponse{
		ID:      id,
		Object:  "chat.completion",
		Created: s.nowUnix(),
		Model:   req.Model,
		Choices: []openAIChatChoice{{
			Index:        0,
			Message:      ChatMessage{Role: "assistant", Content: result.Content},
			FinishReason: finish,
		}},
		Usage: result.Usage,
	})
}

func (s *Server) resolveServableRoute(model string) (Route, bool) {
	for _, route := range s.registry.Resolve(model) {
		if routeStatusServable(route.Status) {
			return route, true
		}
	}
	return Route{}, false
}

func routeStatusServable(status RouteStatus) bool {
	return status == RouteStatusConfigured || status == RouteStatusAvailable
}

type openAIChatCompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []openAIChatChoice `json:"choices"`
	Usage   *Usage             `json:"usage,omitempty"`
}

type openAIChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

func (s *Server) chatCompletionWithFallback(ctx context.Context, route Route, req ChatCompletionRequest) (ChatCompletionResult, Route, string, error) {
	current := route
	for {
		s.recordAttempt(current, req)
		result, err := s.provider.ChatCompletion(ctx, current, req)
		if err == nil {
			s.recordSuccess(current, result.Usage)
			return result, current, "", nil
		}
		class := providerErrorClass(err)
		s.recordFailure(current, class)
		next, ok := s.nextFallbackRoute(current, class)
		if !ok {
			return ChatCompletionResult{}, current, class, err
		}
		s.recordFallback(current, next, class)
		current = next
	}
}

func (s *Server) handleStreamingChat(w http.ResponseWriter, r *http.Request, route Route, req ChatCompletionRequest) {
	streamer, ok := s.provider.(StreamChatProvider)
	if !ok {
		writeRouterError(w, http.StatusServiceUnavailable, "router_streaming_unavailable")
		return
	}
	current := route
	for {
		s.recordAttempt(current, req)
		result, err := streamer.StreamChatCompletion(r.Context(), current, req)
		if err != nil {
			class := providerErrorClass(err)
			s.recordFailure(current, class)
			next, ok := s.nextFallbackRoute(current, class)
			if ok {
				s.recordFallback(current, next, class)
				current = next
				continue
			}
			writeRouterError(w, http.StatusBadGateway, class)
			return
		}
		writeRouterSSEHeaders(w)
		flusher, _ := w.(http.Flusher)
		for _, chunk := range result.Chunks {
			writeRouterSSE(w, openAIChatChunk{
				ID:      "chatcmpl-gormes-router",
				Object:  "chat.completion.chunk",
				Created: s.nowUnix(),
				Model:   req.Model,
				Choices: []openAIChatChunkChoice{{Index: 0, Delta: openAIChatDelta{Content: chunk}}},
			})
			if flusher != nil {
				flusher.Flush()
			}
		}
		if result.Err != nil {
			class := providerErrorClass(result.Err)
			s.recordFailure(current, class)
			writeRouterSSE(w, map[string]any{"error": map[string]any{"code": "upstream_stream_interrupted", "type": "server_error"}})
			fmt.Fprint(w, "data: [DONE]\n\n")
			if flusher != nil {
				flusher.Flush()
			}
			return
		}
		s.recordSuccess(current, result.Usage)
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		return
	}
}

type openAIChatChunk struct {
	ID      string                  `json:"id"`
	Object  string                  `json:"object"`
	Created int64                   `json:"created"`
	Model   string                  `json:"model"`
	Choices []openAIChatChunkChoice `json:"choices"`
}

type openAIChatChunkChoice struct {
	Index int             `json:"index"`
	Delta openAIChatDelta `json:"delta"`
}

type openAIChatDelta struct {
	Content string `json:"content,omitempty"`
}

func writeRouterSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
}

func writeRouterSSE(w http.ResponseWriter, body any) {
	raw, err := json.Marshal(body)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", raw)
}

func (s *Server) nextFallbackRoute(route Route, class string) (Route, bool) {
	if !fallbackClassRetryable(class) {
		return Route{}, false
	}
	for _, rule := range s.model.Fallback {
		if !strings.EqualFold(rule.From, route.Alias) || !stringInSlice(class, rule.On) {
			continue
		}
		if next, ok := s.routeByAlias(rule.To); ok && routeStatusServable(next.Status) {
			return next, true
		}
	}
	return Route{}, false
}

func (s *Server) routeByAlias(alias string) (Route, bool) {
	for _, route := range s.model.Routes {
		if strings.EqualFold(route.Alias, alias) || strings.EqualFold(route.Name, alias) {
			return route, true
		}
	}
	return Route{}, false
}

func providerErrorClass(err error) string {
	var providerErr ProviderError
	if errors.As(err, &providerErr) {
		class := strings.ToLower(strings.TrimSpace(providerErr.Class))
		if class != "" {
			return class
		}
	}
	return "upstream_request_failed"
}

func fallbackClassRetryable(class string) bool {
	switch strings.ToLower(strings.TrimSpace(class)) {
	case "rate_limit", "server_error", "timeout", "connection_failure":
		return true
	default:
		return false
	}
}

func stringInSlice(value string, values []string) bool {
	for _, candidate := range values {
		if strings.EqualFold(strings.TrimSpace(candidate), strings.TrimSpace(value)) {
			return true
		}
	}
	return false
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeRouterError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	writeRouterJSON(w, http.StatusOK, routerStatusResponse{
		Enabled:  s.model.Enabled,
		Listen:   s.model.Listen,
		Status:   s.model.Status,
		Counters: s.counterSnapshot(),
		Routes:   routerStatusRoutes(s.model.Routes),
	})
}

type routerStatusResponse struct {
	Enabled  bool                `json:"enabled"`
	Listen   string              `json:"listen"`
	Status   Status              `json:"status"`
	Counters routerCounters      `json:"counters"`
	Routes   []routerStatusRoute `json:"routes"`
}

type routerStatusRoute struct {
	Name             string           `json:"name"`
	Alias            string           `json:"alias"`
	Provider         string           `json:"provider"`
	Model            string           `json:"model"`
	Status           RouteStatus      `json:"status"`
	CredentialStatus CredentialStatus `json:"credential_status"`
	Optional         bool             `json:"optional,omitempty"`
	QuotaStatus      string           `json:"quota_status,omitempty"`
	Evidence         []string         `json:"evidence,omitempty"`
}

func routerStatusRoutes(routes []Route) []routerStatusRoute {
	out := make([]routerStatusRoute, 0, len(routes))
	for _, route := range routes {
		out = append(out, routerStatusRoute{
			Name:             route.Name,
			Alias:            route.Alias,
			Provider:         route.Provider,
			Model:            route.Model,
			Status:           route.Status,
			CredentialStatus: route.CredentialStatus,
			Optional:         route.Optional,
			QuotaStatus:      route.QuotaStatus,
			Evidence:         append([]string(nil), route.Evidence...),
		})
	}
	return out
}

func (s *Server) recordAttempt(route Route, req ChatCompletionRequest) {
	s.mu.Lock()
	s.counters.Attempts++
	s.mu.Unlock()
	s.safeLog("router_event=chat_attempt route=%s provider=%s model=%s", route.Alias, route.Provider, req.Model)
}

func (s *Server) recordSuccess(route Route, usage *Usage) {
	s.mu.Lock()
	s.counters.Successes++
	if usage != nil {
		s.counters.Usage.PromptTokens += usage.PromptTokens
		s.counters.Usage.CompletionTokens += usage.CompletionTokens
		s.counters.Usage.TotalTokens += usage.TotalTokens
	}
	s.mu.Unlock()
	s.safeLog("router_event=chat_success route=%s provider=%s", route.Alias, route.Provider)
}

func (s *Server) recordFailure(route Route, class string) {
	s.mu.Lock()
	s.counters.Failures++
	s.counters.LastErrorClass = class
	s.mu.Unlock()
	s.safeLog("router_event=chat_failure route=%s provider=%s error_class=%s", route.Alias, route.Provider, class)
}

func (s *Server) recordFallback(from, to Route, class string) {
	s.mu.Lock()
	s.counters.Fallbacks++
	s.mu.Unlock()
	s.safeLog("router_event=fallback from=%s to=%s error_class=%s", from.Alias, to.Alias, class)
}

func (s *Server) counterSnapshot() routerCounters {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.counters
}

func (s *Server) safeLog(format string, args ...any) {
	if s.logf != nil {
		s.logf(format, args...)
	}
}

func routerLogLine(format string, args ...any) string {
	return fmt.Sprintf(format, args...) + "\n"
}

func writeRouterJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeRouterError(w http.ResponseWriter, status int, code string) {
	writeRouterJSON(w, status, map[string]any{
		"error": map[string]any{
			"message": code,
			"type":    routerErrorType(status),
			"code":    code,
		},
	})
}

func routerErrorType(status int) string {
	switch {
	case status == http.StatusUnauthorized:
		return "authentication_error"
	case status >= 500:
		return "server_error"
	default:
		return "invalid_request_error"
	}
}

func (r ChatCompletionRequest) String() string {
	return fmt.Sprintf("model=%s messages=%d stream=%t", r.Model, len(r.Messages), r.Stream)
}
