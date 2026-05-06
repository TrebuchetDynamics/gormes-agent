package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrMCPBreakerOpen = errors.New("mcp circuit breaker: open")

type MCPCircuitEvidence string

const (
	MCPCircuitEvidenceOK                MCPCircuitEvidence = "mcp_ok"
	MCPCircuitEvidenceServerUnreachable MCPCircuitEvidence = "mcp_server_unreachable"
	MCPCircuitEvidenceBreakerOpen       MCPCircuitEvidence = "mcp_breaker_open"
	MCPCircuitEvidenceHalfOpenFailed    MCPCircuitEvidence = "mcp_half_open_failed"
	MCPCircuitEvidenceReconnectRequired MCPCircuitEvidence = "mcp_reconnect_required"
	MCPCircuitEvidenceReconnectReset    MCPCircuitEvidence = "mcp_reconnect_reset"
)

const (
	defaultMCPCircuitBreakerThreshold = 3
	defaultMCPCircuitBreakerCooldown  = 60 * time.Second
	defaultMCPServerName              = "default"
)

type MCPCircuitBreakerOptions struct {
	Threshold int
	Cooldown  time.Duration
	Now       func() time.Time
}

type MCPCircuitBreaker struct {
	mu          sync.Mutex
	threshold   int
	cooldown    time.Duration
	now         func() time.Time
	errorCounts map[string]int
	openedAt    map[string]time.Time
	halfOpen    map[string]bool
}

func NewMCPCircuitBreaker(opts MCPCircuitBreakerOptions) *MCPCircuitBreaker {
	threshold := opts.Threshold
	if threshold <= 0 {
		threshold = defaultMCPCircuitBreakerThreshold
	}
	cooldown := opts.Cooldown
	if cooldown <= 0 {
		cooldown = defaultMCPCircuitBreakerCooldown
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &MCPCircuitBreaker{
		threshold:   threshold,
		cooldown:    cooldown,
		now:         now,
		errorCounts: map[string]int{},
		openedAt:    map[string]time.Time{},
		halfOpen:    map[string]bool{},
	}
}

func (b *MCPCircuitBreaker) ErrorCount(server string) int {
	if b == nil {
		return 0
	}
	server = normalizeMCPBreakerServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.errorCounts[server]
}

func (b *MCPCircuitBreaker) ResetAfterReconnect(server string) MCPCircuitEvidence {
	if b == nil {
		return MCPCircuitEvidenceReconnectReset
	}
	server = normalizeMCPBreakerServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.errorCounts, server)
	delete(b.openedAt, server)
	delete(b.halfOpen, server)
	return MCPCircuitEvidenceReconnectReset
}

func (b *MCPCircuitBreaker) RecordSuccess(server string) MCPCircuitEvidence {
	if b == nil {
		return MCPCircuitEvidenceOK
	}
	server = normalizeMCPBreakerServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.errorCounts, server)
	delete(b.openedAt, server)
	delete(b.halfOpen, server)
	return MCPCircuitEvidenceOK
}

func (b *MCPCircuitBreaker) RecordFailure(server string, _ error) MCPCircuitEvidence {
	if b == nil {
		return MCPCircuitEvidenceServerUnreachable
	}
	server = normalizeMCPBreakerServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	if b.halfOpen[server] {
		b.errorCounts[server] = b.threshold
		b.openedAt[server] = now
		delete(b.halfOpen, server)
		return MCPCircuitEvidenceHalfOpenFailed
	}
	b.errorCounts[server]++
	if b.errorCounts[server] >= b.threshold {
		b.openedAt[server] = now
	}
	return MCPCircuitEvidenceServerUnreachable
}

func (b *MCPCircuitBreaker) beforeCall(server string) (bool, MCPCircuitEvidence) {
	if b == nil {
		return true, MCPCircuitEvidenceOK
	}
	server = normalizeMCPBreakerServer(server)
	b.mu.Lock()
	defer b.mu.Unlock()
	count := b.errorCounts[server]
	if count < b.threshold {
		return true, MCPCircuitEvidenceOK
	}
	now := b.now()
	opened := b.openedAt[server]
	if opened.IsZero() {
		b.openedAt[server] = now
		return false, MCPCircuitEvidenceBreakerOpen
	}
	if now.Sub(opened) >= b.cooldown {
		b.halfOpen[server] = true
		return true, MCPCircuitEvidenceOK
	}
	return false, MCPCircuitEvidenceBreakerOpen
}

func normalizeMCPBreakerServer(server string) string {
	if server == "" {
		return defaultMCPServerName
	}
	return server
}

type MCPToolCallFunc func(context.Context) (MCPCallResult, error)

func CallMCPWithCircuitBreaker(ctx context.Context, breaker *MCPCircuitBreaker, server string, call MCPToolCallFunc) (MCPCallResult, MCPCircuitEvidence, error) {
	if call == nil {
		err := errors.New("mcp call: nil tool call")
		return MCPCallResult{}, MCPCircuitEvidenceServerUnreachable, err
	}
	if allow, evidence := breaker.beforeCall(server); !allow {
		return MCPCallResult{}, evidence, ErrMCPBreakerOpen
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

type MCPLifecycleEvent string

const (
	MCPLifecycleEventNone      MCPLifecycleEvent = ""
	MCPLifecycleEventReconnect MCPLifecycleEvent = "reconnect"
	MCPLifecycleEventShutdown  MCPLifecycleEvent = "shutdown"
)

type MCPServerLifecycle struct {
	mu        sync.Mutex
	reconnect bool
	shutdown  bool
}

func NewMCPServerLifecycle() *MCPServerLifecycle {
	return &MCPServerLifecycle{}
}

func (l *MCPServerLifecycle) SignalReconnect() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reconnect = true
}

func (l *MCPServerLifecycle) SignalShutdown() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.shutdown = true
}

func (l *MCPServerLifecycle) ReconnectPending() bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reconnect
}

func (l *MCPServerLifecycle) NextEvent() MCPLifecycleEvent {
	if l == nil {
		return MCPLifecycleEventNone
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		l.reconnect = false
		l.shutdown = false
		return MCPLifecycleEventShutdown
	}
	if l.reconnect {
		l.reconnect = false
		return MCPLifecycleEventReconnect
	}
	return MCPLifecycleEventNone
}

type MCPProbeSession interface {
	ListTools(context.Context) ([]MCPRawTool, error)
	Close() error
}

type MCPProbeConnector func(context.Context, MCPServerDefinition) (MCPProbeSession, error)

func ProbeMCPServerTools(ctx context.Context, servers []MCPServerDefinition, connect MCPProbeConnector) map[string][]MCPRawTool {
	out := map[string][]MCPRawTool{}
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
		out[server.Name] = append([]MCPRawTool(nil), tools...)
	}
	return out
}

// MCPCallResult is the normalized envelope produced by an MCP `tools/call`
// response. Content captures the structured body in the same StructuredContent
// shape used by NormalizeTools/RenderToolCallResult so call sites do not need
// transport-specific decoders. IsError mirrors the protocol's `isError`
// boolean: a true value means the tool reported a failure inside an otherwise
// successful JSON-RPC response (transport-level errors stay separate).
type MCPCallResult struct {
	Content []StructuredContent
	IsError bool
}

// rawToolCallResult mirrors the on-the-wire shape of an MCP tools/call
// response. Content blocks are decoded into a representation-agnostic
// StructuredContent slice via parseMCPCallResult.
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

// parseMCPCallResult turns a raw `result` JSON document into a
// transport-free MCPCallResult. Empty bodies are valid: tools that report
// success without content come back with IsError=false and zero Content
// blocks so callers can render them as a no-op.
func parseMCPCallResult(raw json.RawMessage) (MCPCallResult, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return MCPCallResult{}, nil
	}
	var decoded rawToolCallResult
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return MCPCallResult{}, fmt.Errorf("mcp call: parse result: %w", err)
	}
	out := MCPCallResult{IsError: decoded.IsError}
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

// CallTool invokes the named tool over the HTTP transport and decodes the
// response into the shared MCPCallResult shape. Arguments may be nil; the
// MCP server then receives an empty object so providers that require an
// `arguments` field do not see a malformed request.
//
// Transport-level failures (connectivity, auth, timeouts) bubble up as the
// underlying error so callers can branch on errors.Is(ErrAuthRequired) and
// friends. Application-level failures (the server reports `isError: true`)
// surface inside the returned MCPCallResult instead so structured content
// is preserved for the caller.
func (c *HTTPClient) CallTool(ctx context.Context, name string, arguments map[string]any) (MCPCallResult, error) {
	if arguments == nil {
		arguments = map[string]any{}
	}
	params := map[string]any{
		"name":      name,
		"arguments": arguments,
	}
	var raw json.RawMessage
	if err := c.call(ctx, "tools/call", params, &raw); err != nil {
		return MCPCallResult{}, err
	}
	return parseMCPCallResult(raw)
}
