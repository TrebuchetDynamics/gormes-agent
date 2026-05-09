package acp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode"

	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

var ErrJSONRPCSessionNotFound = errors.New("acp jsonrpc: session not found")

type SessionRuntimeConfig struct {
	Provider    string
	Model       string
	Runner      PromptRunner
	SessionMap  session.Map
	IDGenerator func() string
	Permissions *PermissionBroker
}

type SessionRuntime struct {
	mu          sync.Mutex
	provider    string
	model       string
	runner      PromptRunner
	sessionMap  session.Map
	idGenerator func() string
	permissions *PermissionBroker
	sessions    map[string]*runtimeSessionState
}

type RuntimeSession struct {
	ID       string
	CWD      string
	Provider string
	Model    string
	Title    string
}

type runtimeSessionState struct {
	RuntimeSession
	history       []RuntimeMessage
	queuedPrompts []string
	running       bool
	cancelled     bool
}

type RuntimeMessage struct {
	Role    string
	Content string
}

type ACPContentBlock struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	URI      string `json:"uri,omitempty"`
	MIMEType string `json:"mimeType,omitempty"`
}

type RuntimePromptRequest struct {
	SessionID string
	Text      string
	CWD       string
	Blocks    []ACPContentBlock
}

type PromptEventKind string

const (
	PromptEventAgentMessageChunk PromptEventKind = "agent_message_chunk"
	PromptEventUserMessageChunk  PromptEventKind = "user_message_chunk"
	PromptEventSessionTitle      PromptEventKind = "session_title_update"
	PromptEventUsage             PromptEventKind = "usage_update"
)

type PromptEvent struct {
	Kind     PromptEventKind
	Text     string
	Title    string
	Usage    *ACPUsage
	ToolCall *ToolCallStart
}

type ACPUsage struct {
	InputTokens      int `json:"inputTokens,omitempty"`
	OutputTokens     int `json:"outputTokens,omitempty"`
	TotalTokens      int `json:"totalTokens,omitempty"`
	ThoughtTokens    int `json:"thoughtTokens,omitempty"`
	CachedReadTokens int `json:"cachedReadTokens,omitempty"`
}

type PromptResult struct {
	Final      string
	StopReason string
	Usage      *ACPUsage
	Title      string
}

type PromptRunner interface {
	RunPrompt(ctx context.Context, req RuntimePromptRequest, emit func(PromptEvent)) (PromptResult, error)
}

type PromptRunnerFunc func(ctx context.Context, req RuntimePromptRequest, emit func(PromptEvent)) (PromptResult, error)

func (f PromptRunnerFunc) RunPrompt(ctx context.Context, req RuntimePromptRequest, emit func(PromptEvent)) (PromptResult, error) {
	return f(ctx, req, emit)
}

func NewSessionRuntime(cfg SessionRuntimeConfig) *SessionRuntime {
	provider := strings.TrimSpace(cfg.Provider)
	if provider == "" {
		provider = "openrouter"
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "gpt-5.4"
	}
	runner := cfg.Runner
	if runner == nil {
		runner = PromptRunnerFunc(func(_ context.Context, req RuntimePromptRequest, _ func(PromptEvent)) (PromptResult, error) {
			return PromptResult{
				Final:      strings.TrimSpace(req.Text),
				StopReason: "end_turn",
			}, nil
		})
	}
	permissions := cfg.Permissions
	if permissions == nil {
		permissions = NewPermissionBroker()
	}
	return &SessionRuntime{
		provider:    provider,
		model:       model,
		runner:      runner,
		sessionMap:  cfg.SessionMap,
		idGenerator: cfg.IDGenerator,
		permissions: permissions,
		sessions:    make(map[string]*runtimeSessionState),
	}
}

func (r *SessionRuntime) Provider() string {
	if r == nil {
		return ""
	}
	return r.provider
}

func (r *SessionRuntime) NewSession(ctx context.Context, cwd string) (RuntimeSession, error) {
	if r == nil {
		return RuntimeSession{}, errors.New("acp jsonrpc: nil runtime")
	}
	id := callIDGenerator(r.idGenerator)
	state := &runtimeSessionState{
		RuntimeSession: RuntimeSession{
			ID:       id,
			CWD:      TranslateACPCWD(cwd),
			Provider: r.provider,
			Model:    r.model,
		},
	}
	r.mu.Lock()
	r.sessions[id] = state
	r.mu.Unlock()
	if r.sessionMap != nil {
		if err := r.sessionMap.Put(ctx, acpSessionMapKey(id), id); err != nil {
			return RuntimeSession{}, err
		}
	}
	return state.RuntimeSession, nil
}

func (r *SessionRuntime) LoadSession(ctx context.Context, sessionID, cwd string) (*RuntimeSession, error) {
	return r.loadSession(ctx, sessionID, cwd)
}

func (r *SessionRuntime) ResumeSession(ctx context.Context, sessionID, cwd string) (*RuntimeSession, error) {
	return r.loadSession(ctx, sessionID, cwd)
}

func (r *SessionRuntime) CancelSession(_ context.Context, sessionID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	state := r.sessions[strings.TrimSpace(sessionID)]
	if state == nil {
		return false
	}
	state.cancelled = true
	return true
}

func (r *SessionRuntime) Prompt(ctx context.Context, req RuntimePromptRequest, emit func(PromptEvent)) (PromptResult, error) {
	if r == nil {
		return PromptResult{}, errors.New("acp jsonrpc: nil runtime")
	}
	if emit == nil {
		emit = func(PromptEvent) {}
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		text = ExtractACPText(req.Blocks)
	}
	return r.promptText(ctx, strings.TrimSpace(req.SessionID), text, req.Blocks, emit)
}

func (r *SessionRuntime) PermissionBroker() *PermissionBroker {
	if r == nil {
		return nil
	}
	return r.permissions
}

func (r *SessionRuntime) loadSession(ctx context.Context, sessionID, cwd string) (*RuntimeSession, error) {
	if r == nil {
		return nil, errors.New("acp jsonrpc: nil runtime")
	}
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return nil, ErrJSONRPCSessionNotFound
	}
	translated := TranslateACPCWD(cwd)
	r.mu.Lock()
	if state := r.sessions[id]; state != nil {
		state.CWD = translated
		out := state.RuntimeSession
		r.mu.Unlock()
		return &out, nil
	}
	r.mu.Unlock()

	if r.sessionMap == nil {
		return nil, nil
	}
	stored, err := r.sessionMap.Get(ctx, acpSessionMapKey(id))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(stored) != id {
		return nil, nil
	}
	state := &runtimeSessionState{
		RuntimeSession: RuntimeSession{
			ID:       id,
			CWD:      translated,
			Provider: r.provider,
			Model:    r.model,
		},
	}
	r.mu.Lock()
	r.sessions[id] = state
	r.mu.Unlock()
	out := state.RuntimeSession
	return &out, nil
}

func (r *SessionRuntime) promptText(ctx context.Context, sessionID, text string, blocks []ACPContentBlock, emit func(PromptEvent)) (PromptResult, error) {
	r.mu.Lock()
	state := r.sessions[sessionID]
	if state == nil {
		r.mu.Unlock()
		return PromptResult{StopReason: "refusal"}, ErrJSONRPCSessionNotFound
	}
	if strings.HasPrefix(text, "/queue") {
		queued := strings.TrimSpace(strings.TrimPrefix(text, "/queue"))
		if queued != "" {
			state.queuedPrompts = append(state.queuedPrompts, queued)
		}
		depth := len(state.queuedPrompts)
		r.mu.Unlock()
		emit(PromptEvent{Kind: PromptEventAgentMessageChunk, Text: fmt.Sprintf("Queued for the next turn. (%d queued)", depth)})
		return PromptResult{StopReason: "end_turn"}, nil
	}
	if strings.HasPrefix(text, "/steer") && !state.running {
		steer := strings.TrimSpace(strings.TrimPrefix(text, "/steer"))
		if steer != "" {
			text = steer
		}
	}
	if state.running {
		state.queuedPrompts = append(state.queuedPrompts, text)
		depth := len(state.queuedPrompts)
		r.mu.Unlock()
		emit(PromptEvent{Kind: PromptEventAgentMessageChunk, Text: fmt.Sprintf("Queued for the next turn. (%d queued)", depth)})
		return PromptResult{StopReason: "end_turn"}, nil
	}
	state.running = true
	state.cancelled = false
	state.history = append(state.history, RuntimeMessage{Role: "user", Content: text})
	cwd := state.CWD
	r.mu.Unlock()

	streamed := false
	result, err := r.runner.RunPrompt(ctx, RuntimePromptRequest{SessionID: sessionID, Text: text, CWD: cwd, Blocks: blocks}, func(event PromptEvent) {
		if event.Kind == PromptEventAgentMessageChunk && event.Text != "" {
			streamed = true
		}
		emit(event)
	})
	if err != nil {
		result = PromptResult{Final: "Error: " + err.Error(), StopReason: "end_turn"}
	}
	if result.StopReason == "" {
		result.StopReason = "end_turn"
	}
	if result.Final != "" {
		r.mu.Lock()
		if state := r.sessions[sessionID]; state != nil {
			state.history = append(state.history, RuntimeMessage{Role: "assistant", Content: result.Final})
			if result.Title != "" {
				state.Title = result.Title
			}
		}
		r.mu.Unlock()
		if !streamed {
			emit(PromptEvent{Kind: PromptEventAgentMessageChunk, Text: result.Final})
		}
	}
	if result.Title != "" {
		emit(PromptEvent{Kind: PromptEventSessionTitle, Title: result.Title})
	}
	if result.Usage != nil {
		emit(PromptEvent{Kind: PromptEventUsage, Usage: result.Usage})
	}

	r.mu.Lock()
	if state := r.sessions[sessionID]; state != nil {
		if state.cancelled {
			result.StopReason = "cancelled"
		}
		state.running = false
	}
	r.mu.Unlock()

	for {
		r.mu.Lock()
		state := r.sessions[sessionID]
		if state == nil || len(state.queuedPrompts) == 0 {
			r.mu.Unlock()
			break
		}
		next := state.queuedPrompts[0]
		state.queuedPrompts = state.queuedPrompts[1:]
		r.mu.Unlock()
		emit(PromptEvent{Kind: PromptEventUserMessageChunk, Text: next})
		if _, err := r.promptText(ctx, sessionID, next, []ACPContentBlock{{Type: "text", Text: next}}, emit); err != nil {
			return result, err
		}
	}
	return result, err
}

func ExtractACPText(blocks []ACPContentBlock) string {
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if strings.EqualFold(block.Type, "text") && strings.TrimSpace(block.Text) != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func TranslateACPCWD(cwd string) string {
	raw := strings.TrimSpace(cwd)
	if len(raw) < 2 || raw[1] != ':' || !isASCIILetter(rune(raw[0])) {
		return raw
	}
	drive := unicode.ToLower(rune(raw[0]))
	rest := strings.ReplaceAll(raw[2:], "\\", "/")
	rest = strings.TrimLeft(rest, "/")
	if rest == "" {
		return "/mnt/" + string(drive)
	}
	return "/mnt/" + string(drive) + "/" + rest
}

func acpSessionMapKey(id string) string {
	return "acp:" + strings.TrimSpace(id)
}

func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

type PermissionRequest struct {
	Key         string
	Command     string
	Description string
}

type PermissionDecision struct {
	Outcome PermissionOptionKind
	Result  string
	Reason  string
	Cached  bool
}

type PermissionRequester interface {
	RequestPermission(ctx context.Context, req PermissionRequest) (PermissionDecision, error)
}

type PermissionRequesterFunc func(ctx context.Context, req PermissionRequest) (PermissionDecision, error)

func (f PermissionRequesterFunc) RequestPermission(ctx context.Context, req PermissionRequest) (PermissionDecision, error) {
	return f(ctx, req)
}

type PermissionBroker struct {
	mu        sync.Mutex
	requester map[string]PermissionRequester
	cache     map[string]map[string]PermissionDecision
}

func NewPermissionBroker() *PermissionBroker {
	return &PermissionBroker{
		requester: make(map[string]PermissionRequester),
		cache:     make(map[string]map[string]PermissionDecision),
	}
}

func (b *PermissionBroker) SetRequester(sessionID string, requester PermissionRequester) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.requester[strings.TrimSpace(sessionID)] = requester
}

func (b *PermissionBroker) Request(ctx context.Context, sessionID string, req PermissionRequest) (PermissionDecision, error) {
	if b == nil {
		return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: "permission_broker_unavailable"}, nil
	}
	sessionID = strings.TrimSpace(sessionID)
	key := strings.TrimSpace(req.Key)
	if key == "" {
		key = strings.TrimSpace(req.Command)
	}
	if key == "" {
		key = "default"
	}
	b.mu.Lock()
	if cached := b.cache[sessionID][key]; cached.Result != "" {
		cached.Cached = true
		b.mu.Unlock()
		return cached, nil
	}
	requester := b.requester[sessionID]
	b.mu.Unlock()
	if requester == nil {
		return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: "permission_requester_unavailable"}, nil
	}
	decision, err := requester.RequestPermission(ctx, req)
	if err != nil {
		reason := "permission_error"
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			reason = "permission_timeout"
		}
		return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: reason}, nil
	}
	if decision.Outcome == "" {
		decision.Outcome = PermissionRejectOnce
	}
	decision.Result = PermissionOutcomeForKind(decision.Outcome)
	if decision.Outcome == PermissionAllowAlways || decision.Outcome == PermissionRejectAlways {
		b.mu.Lock()
		if b.cache[sessionID] == nil {
			b.cache[sessionID] = make(map[string]PermissionDecision)
		}
		b.cache[sessionID][key] = decision
		b.mu.Unlock()
	}
	return decision, nil
}
