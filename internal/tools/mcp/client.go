package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrBreakerOpen = errors.New("mcp circuit breaker: open")

type CircuitEvidence string

const (
	CircuitEvidenceOK                CircuitEvidence = "mcp_ok"
	CircuitEvidenceServerUnreachable CircuitEvidence = "mcp_server_unreachable"
	CircuitEvidenceBreakerOpen       CircuitEvidence = "mcp_breaker_open"
	CircuitEvidenceHalfOpenFailed    CircuitEvidence = "mcp_half_open_failed"
	CircuitEvidenceReconnectRequired CircuitEvidence = "mcp_reconnect_required"
	CircuitEvidenceReconnectReset    CircuitEvidence = "mcp_reconnect_reset"
)

const (
	DefaultCircuitBreakerThreshold = 3
	DefaultCircuitBreakerCooldown  = 60 * time.Second
	DefaultServerName              = "default"
)

type CircuitBreakerOptions struct {
	Threshold int
	Cooldown  time.Duration
	Now       func() time.Time
}

type CircuitBreaker struct {
	mu          sync.Mutex
	threshold   int
	cooldown    time.Duration
	now         func() time.Time
	errorCounts map[string]int
	openedAt    map[string]time.Time
	halfOpen    map[string]bool
}

func NewCircuitBreaker(opts CircuitBreakerOptions) *CircuitBreaker {
	threshold := opts.Threshold
	if threshold <= 0 {
		threshold = DefaultCircuitBreakerThreshold
	}
	cooldown := opts.Cooldown
	if cooldown <= 0 {
		cooldown = DefaultCircuitBreakerCooldown
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &CircuitBreaker{
		threshold:   threshold,
		cooldown:    cooldown,
		now:         now,
		errorCounts: map[string]int{},
		openedAt:    map[string]time.Time{},
		halfOpen:    map[string]bool{},
	}
}

func (b *CircuitBreaker) ErrorCount(server string) int {
	if b == nil {
		return 0
	}
	server = normalizeMCPBreakerServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.errorCounts[server]
}

func (b *CircuitBreaker) ResetAfterReconnect(server string) CircuitEvidence {
	if b == nil {
		return CircuitEvidenceReconnectReset
	}
	server = normalizeMCPBreakerServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.errorCounts, server)
	delete(b.openedAt, server)
	delete(b.halfOpen, server)
	return CircuitEvidenceReconnectReset
}

func (b *CircuitBreaker) RecordSuccess(server string) CircuitEvidence {
	if b == nil {
		return CircuitEvidenceOK
	}
	server = normalizeMCPBreakerServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.errorCounts, server)
	delete(b.openedAt, server)
	delete(b.halfOpen, server)
	return CircuitEvidenceOK
}

func (b *CircuitBreaker) RecordFailure(server string, _ error) CircuitEvidence {
	if b == nil {
		return CircuitEvidenceServerUnreachable
	}
	server = normalizeMCPBreakerServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	if b.halfOpen[server] {
		b.errorCounts[server] = b.threshold
		b.openedAt[server] = now
		delete(b.halfOpen, server)
		return CircuitEvidenceHalfOpenFailed
	}
	b.errorCounts[server]++
	if b.errorCounts[server] >= b.threshold {
		b.openedAt[server] = now
	}
	return CircuitEvidenceServerUnreachable
}

func (b *CircuitBreaker) beforeCall(server string) (bool, CircuitEvidence) {
	if b == nil {
		return true, CircuitEvidenceOK
	}
	server = normalizeMCPBreakerServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	count := b.errorCounts[server]
	if count < b.threshold {
		return true, CircuitEvidenceOK
	}
	now := b.now()
	opened := b.openedAt[server]
	if opened.IsZero() {
		b.openedAt[server] = now
		return false, CircuitEvidenceBreakerOpen
	}
	if now.Sub(opened) >= b.cooldown {
		b.halfOpen[server] = true
		return true, CircuitEvidenceOK
	}
	return false, CircuitEvidenceBreakerOpen
}

func normalizeMCPBreakerServer(server string) string {
	if server == "" {
		return DefaultServerName
	}
	return server
}

type ToolCallFunc func(context.Context) (CallResult, error)

func CallWithCircuitBreaker(ctx context.Context, breaker *CircuitBreaker, server string, call ToolCallFunc) (CallResult, CircuitEvidence, error) {
	if call == nil {
		err := errors.New("mcp call: nil tool call")
		return CallResult{}, CircuitEvidenceServerUnreachable, err
	}
	if allow, evidence := breaker.beforeCall(server); !allow {
		return CallResult{}, evidence, ErrBreakerOpen
	}
	result, err := call(ctx)
	if err != nil {
		return result, breaker.RecordFailure(server, err), err
	}
	if result.IsError {
		return result, breaker.RecordFailure(server, errors.New("mcp tool reported isError")), nil
	}
	return result, breaker.RecordSuccess(server), nil
}

type LifecycleEvent string

const (
	LifecycleEventNone      LifecycleEvent = ""
	LifecycleEventReconnect LifecycleEvent = "reconnect"
	LifecycleEventShutdown  LifecycleEvent = "shutdown"
)

type ServerLifecycle struct {
	mu        sync.Mutex
	reconnect bool
	shutdown  bool
}

func NewServerLifecycle() *ServerLifecycle {
	return &ServerLifecycle{}
}

func (l *ServerLifecycle) SignalReconnect() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reconnect = true
}

func (l *ServerLifecycle) SignalShutdown() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.shutdown = true
}

func (l *ServerLifecycle) ReconnectPending() bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reconnect
}

func (l *ServerLifecycle) NextEvent() LifecycleEvent {
	if l == nil {
		return LifecycleEventNone
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		l.reconnect = false
		l.shutdown = false
		return LifecycleEventShutdown
	}
	if l.reconnect {
		l.reconnect = false
		return LifecycleEventReconnect
	}
	return LifecycleEventNone
}

type ProbeSession interface {
	ListTools(context.Context) ([]RawTool, error)
	Close() error
}

type ProbeConnector func(context.Context, MCPServerDefinition) (ProbeSession, error)

func ProbeServerTools(ctx context.Context, servers []MCPServerDefinition, connect ProbeConnector) map[string][]RawTool {
	out := map[string][]RawTool{}
	if connect == nil {
		return out
	}
	for _, server := range servers {
		if !server.Enabled || server.Name == "" {
			continue
		}
		session, err := connect(ctx, server)
		if err != nil || session == nil {
			continue
		}
		tools, listErr := session.ListTools(ctx)
		_ = session.Close()
		if listErr != nil {
			continue
		}
		out[server.Name] = append([]RawTool(nil), tools...)
	}
	return out
}

// CallResult is the normalized envelope produced by an MCP `tools/call`
// response. Content captures the structured body in the same StructuredContent
// shape used by NormalizeTools/RenderToolCallResult so call sites do not need
// transport-specific decoders. IsError mirrors the protocol's `isError`
// boolean: a true value means the tool reported a failure inside an otherwise
// successful JSON-RPC response (transport-level errors stay separate).
type CallResult struct {
	Content []StructuredContent
	IsError bool
}

// rawToolCallResult mirrors the on-the-wire shape of an MCP tools/call
// response. Content blocks are decoded into a representation-agnostic
// StructuredContent slice via ParseCallResult.
type rawToolCallResult struct {
	Content []rawToolCallContent `json:"content"`
	IsError bool                 `json:"isError"`
}

// rawToolCallContent captures the fields the StructuredContent renderer needs
// from each MCP content block. Unknown fields are ignored so SDK extensions
// degrade gracefully instead of failing the parse.
type rawToolCallContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	URI      string `json:"uri,omitempty"`
	Resource *struct {
		URI      string `json:"uri,omitempty"`
		MimeType string `json:"mimeType,omitempty"`
	} `json:"resource,omitempty"`
}

// ParseCallResult turns a raw `result` JSON document into a
// transport-free CallResult. Empty bodies are valid: tools that report
// success without content come back with IsError=false and zero Content
// blocks so callers can render them as a no-op.
func ParseCallResult(raw json.RawMessage) (CallResult, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return CallResult{}, nil
	}
	var decoded rawToolCallResult
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return CallResult{}, fmt.Errorf("mcp call: parse result: %w", err)
	}
	out := CallResult{IsError: decoded.IsError}
	if len(decoded.Content) == 0 {
		return out, nil
	}
	out.Content = make([]StructuredContent, 0, len(decoded.Content))
	for _, block := range decoded.Content {
		out.Content = append(out.Content, normalizeCallContent(block))
	}
	return out, nil
}

// normalizeCallContent collapses a single content block into the shared
// StructuredContent shape. Unknown kinds keep their type label so callers
// can branch on it, and resource blocks merge their nested `resource.uri`
// into the top-level URI field that RenderToolCallResult inspects.
func normalizeCallContent(block rawToolCallContent) StructuredContent {
	out := StructuredContent{
		Kind:     block.Type,
		Text:     block.Text,
		MimeType: block.MimeType,
		URI:      block.URI,
	}
	if block.Resource != nil {
		if out.URI == "" {
			out.URI = block.Resource.URI
		}
		if out.MimeType == "" {
			out.MimeType = block.Resource.MimeType
		}
	}
	return out
}
