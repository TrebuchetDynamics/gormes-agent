package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// httpProtocolVersion is the MCP protocol version Gormes negotiates over the
// HTTP transport. It mirrors stdioProtocolVersion so both clients present a
// consistent capability surface to the server.
const httpProtocolVersion = "2024-11-05"

// ErrAuthRequired is returned when the HTTP MCP server replies with 401
// Unauthorized. The OAuth follow-up row keys recovery off this typed error.
var ErrAuthRequired = errors.New("mcp http: authentication required")

// ErrMCPSessionRequired is returned when a Streamable HTTP server requires a
// session ID but the client does not have one yet.
var ErrMCPSessionRequired = errors.New("mcp http: session required")

// ErrMCPSessionExpired is returned when a Streamable HTTP server reports that
// the current session ID is no longer valid and the client must reinitialize.
var ErrMCPSessionExpired = errors.New("mcp http: session expired")

// ErrMCPLegacySSEUnsupported is returned when a deprecated HTTP+SSE endpoint
// is detected but the Streamable HTTP client cannot safely upgrade it.
var ErrMCPLegacySSEUnsupported = errors.New("mcp http: legacy sse unsupported")

// MCPHTTPEvidence is a stable degraded-mode label for MCP HTTP transport
// failures surfaced to operator/status layers.
type MCPHTTPEvidence string

const (
	MCPHTTPEvidenceNone                 MCPHTTPEvidence = ""
	MCPHTTPEvidenceSessionRequired      MCPHTTPEvidence = "mcp_session_required"
	MCPHTTPEvidenceSessionExpired       MCPHTTPEvidence = "mcp_session_expired"
	MCPHTTPEvidenceLegacySSEUnsupported MCPHTTPEvidence = "legacy_sse_unsupported"
)

type mcpHTTPTransportError struct {
	evidence MCPHTTPEvidence
	status   int
	message  string
	err      error
}

func (e *mcpHTTPTransportError) Error() string {
	if e == nil {
		return "mcp http: transport error"
	}
	if e.status > 0 {
		return fmt.Sprintf("mcp http: %s: status %d", e.message, e.status)
	}
	return "mcp http: " + e.message
}

func (e *mcpHTTPTransportError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// MCPHTTPErrorEvidence returns the stable evidence label carried by MCP HTTP
// transport errors. Unknown errors return MCPHTTPEvidenceNone.
func MCPHTTPErrorEvidence(err error) MCPHTTPEvidence {
	var transportErr *mcpHTTPTransportError
	if errors.As(err, &transportErr) {
		return transportErr.evidence
	}
	return MCPHTTPEvidenceNone
}

// HTTPClientOpts injects the http.RoundTripper plus optional observability
// hooks. Tests pass an httptest.Server-backed transport so no real socket is
// opened outside httptest.
type HTTPClientOpts struct {
	Transport http.RoundTripper
	Logger    *slog.Logger
	Now       func() time.Time
}

// HTTPClient speaks JSON-RPC over HTTP to an MCP server. It is the minimal
// MCP HTTP surface needed for `initialize` plus `tools/list`; SSE response
// streaming, OAuth, and structured content normalization live in follow-up
// rows.
type HTTPClient struct {
	def    MCPServerDefinition
	http   *http.Client
	logger *slog.Logger
	now    func() time.Time

	nextID atomic.Int64

	closeMu sync.Mutex
	closed  bool

	versionMu       sync.RWMutex
	protocolVersion string

	sessionMu sync.RWMutex
	sessionID string
}

// NewHTTPClient constructs an HTTPClient over the supplied transport. The
// def.URL must be a non-empty HTTP(S) endpoint; transport choice is left to
// the caller so tests can inject httptest.Server's RoundTripper without ever
// opening a real socket.
func NewHTTPClient(def MCPServerDefinition, opts HTTPClientOpts) (*HTTPClient, error) {
	if def.URL == "" {
		return nil, errors.New("mcp http: empty URL")
	}
	transport := opts.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &HTTPClient{
		def:    def,
		http:   &http.Client{Transport: transport},
		logger: logger,
		now:    now,
	}, nil
}

// ProtocolVersion returns the protocol version the server reported during
// Initialize. Empty before Initialize succeeds.
func (c *HTTPClient) ProtocolVersion() string {
	c.versionMu.RLock()
	defer c.versionMu.RUnlock()
	return c.protocolVersion
}

// SessionID returns the Streamable HTTP session ID captured during
// Initialize. Empty means the server did not assign a session or the previous
// session has expired and must be reinitialized.
func (c *HTTPClient) SessionID() string {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	return c.sessionID
}

func (c *HTTPClient) setSessionID(sessionID string) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	c.sessionID = sessionID
}

func (c *HTTPClient) clearSessionID() {
	c.setSessionID("")
}

// Initialize performs the MCP handshake over HTTP and records the negotiated
// protocol version.
func (c *HTTPClient) Initialize(ctx context.Context) error {
	params := map[string]any{
		"protocolVersion": httpProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "gormes",
			"version": "0.0.0",
		},
	}
	var result struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := c.call(ctx, "initialize", params, &result); err != nil {
		var rpcErr *jsonRPCError
		if errors.As(err, &rpcErr) {
			return fmt.Errorf("%w: %s", ErrInitializeFailed, rpcErr.Message)
		}
		return err
	}
	c.versionMu.Lock()
	c.protocolVersion = result.ProtocolVersion
	c.versionMu.Unlock()
	return nil
}

// ListTools fetches the server's tools/list response and returns the verbatim
// tool envelopes (no schema normalization).
func (c *HTTPClient) ListTools(ctx context.Context) ([]MCPRawTool, error) {
	var result struct {
		Tools []struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			InputSchema json.RawMessage `json:"inputSchema,omitempty"`
		} `json:"tools"`
	}
	if err := c.call(ctx, "tools/list", map[string]any{}, &result); err != nil {
		return nil, err
	}
	out := make([]MCPRawTool, 0, len(result.Tools))
	for _, t := range result.Tools {
		out = append(out, MCPRawTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return out, nil
}

// Close marks the client closed and releases idle connections held by the
// underlying transport. Safe to call multiple times.
func (c *HTTPClient) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if c.closed {
		return nil
	}
	sessionID := c.SessionID()
	if sessionID != "" {
		if err := c.terminateSession(context.Background(), sessionID); err != nil {
			return err
		}
		c.clearSessionID()
	}
	c.closed = true
	if t, ok := c.http.Transport.(*http.Transport); ok {
		t.CloseIdleConnections()
	}
	return nil
}

func (c *HTTPClient) terminateSession(ctx context.Context, sessionID string) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.def.URL, nil)
	if err != nil {
		return fmt.Errorf("mcp http: build delete request: %w", err)
	}
	c.applyRequestHeaders(httpReq, sessionID)
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("mcp http: terminate session: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode == http.StatusMethodNotAllowed {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mcp http: terminate session: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// call posts a single JSON-RPC request and returns the decoded result.
// HTTP-level failures are mapped to typed errors so callers can branch on
// auth/timeout cases without inspecting status codes themselves.
func (c *HTTPClient) call(ctx context.Context, method string, params any, out any) error {
	c.closeMu.Lock()
	closed := c.closed
	c.closeMu.Unlock()
	if closed {
		return io.ErrClosedPipe
	}

	id := c.nextID.Add(1)
	req := jsonRPCRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("mcp http: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.def.URL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("mcp http: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	sessionID := ""
	if method != "initialize" {
		sessionID = c.SessionID()
	}
	c.applyRequestHeaders(httpReq, sessionID)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%w: %v", ErrConnectTimeout, err)
		}
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return ctx.Err()
		}
		return fmt.Errorf("mcp http: do request: %w", err)
	}
	defer func() {
		// Drain and close even on error paths; do not panic if the server
		// hung up mid-stream.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("%w: status %d", ErrAuthRequired, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound && sessionID != "" {
		c.clearSessionID()
		return &mcpHTTPTransportError{
			evidence: MCPHTTPEvidenceSessionExpired,
			status:   resp.StatusCode,
			message:  "session expired; reinitialize required",
			err:      ErrMCPSessionExpired,
		}
	}
	if resp.StatusCode == http.StatusBadRequest && method != "initialize" && sessionID == "" {
		return &mcpHTTPTransportError{
			evidence: MCPHTTPEvidenceSessionRequired,
			status:   resp.StatusCode,
			message:  "session required",
			err:      ErrMCPSessionRequired,
		}
	}
	if method == "initialize" && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed) {
		if err := c.detectLegacySSEEndpoint(ctx); err != nil {
			return err
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mcp http: unexpected status %d", resp.StatusCode)
	}
	sessionIDHeader := resp.Header.Get("Mcp-Session-Id")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%w: %v", ErrConnectTimeout, err)
		}
		return fmt.Errorf("mcp http: read response: %w", err)
	}

	rpcResp, err := decodeMCPHTTPResponse(resp.Header.Get("Content-Type"), body, id)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidJSONRPCResponse, err)
	}
	if rpcResp.Error != nil {
		return rpcResp.Error
	}
	if out != nil && len(rpcResp.Result) > 0 {
		if err := json.Unmarshal(rpcResp.Result, out); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidJSONRPCResponse, err)
		}
	}
	if method == "initialize" {
		c.setSessionID(sessionIDHeader)
	}
	return nil
}

func (c *HTTPClient) applyRequestHeaders(req *http.Request, sessionID string) {
	for k, v := range c.def.Headers {
		req.Header.Set(k, v)
	}
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	if version := c.ProtocolVersion(); version != "" {
		req.Header.Set("MCP-Protocol-Version", version)
	}
}

func decodeMCPHTTPResponse(contentType string, body []byte, requestID int64) (jsonRPCResponse, error) {
	if strings.Contains(strings.ToLower(contentType), "text/event-stream") {
		return decodeMCPHTTPSSE(body, requestID)
	}
	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return jsonRPCResponse{}, err
	}
	return rpcResp, nil
}

func decodeMCPHTTPSSE(body []byte, requestID int64) (jsonRPCResponse, error) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
	var dataLines []string
	flush := func() (jsonRPCResponse, bool, error) {
		if len(dataLines) == 0 {
			return jsonRPCResponse{}, false, nil
		}
		data := strings.TrimSpace(strings.Join(dataLines, "\n"))
		dataLines = nil
		if data == "" || data == "[DONE]" || !strings.HasPrefix(data, "{") {
			return jsonRPCResponse{}, false, nil
		}
		var rpcResp jsonRPCResponse
		if err := json.Unmarshal([]byte(data), &rpcResp); err != nil {
			return jsonRPCResponse{}, false, err
		}
		if rpcResp.ID == nil || *rpcResp.ID != requestID {
			return jsonRPCResponse{}, false, nil
		}
		return rpcResp, true, nil
	}
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			if rpcResp, ok, err := flush(); ok || err != nil {
				return rpcResp, err
			}
			continue
		}
		if after, ok := strings.CutPrefix(line, "data:"); ok {
			dataLines = append(dataLines, strings.TrimSpace(after))
		}
	}
	if err := scanner.Err(); err != nil {
		return jsonRPCResponse{}, err
	}
	if rpcResp, ok, err := flush(); ok || err != nil {
		return rpcResp, err
	}
	return jsonRPCResponse{}, fmt.Errorf("no JSON-RPC response for id %d in SSE stream", requestID)
}

func (c *HTTPClient) detectLegacySSEEndpoint(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.def.URL, nil)
	if err != nil {
		return fmt.Errorf("mcp http: build legacy sse probe: %w", err)
	}
	httpReq.Header.Set("Accept", "text/event-stream")
	c.applyRequestHeaders(httpReq, "")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("mcp http: legacy sse probe: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("mcp http: legacy sse probe read: %w", err)
	}
	if legacySSEEndpointCarriesSessionID(body) {
		return &mcpHTTPTransportError{
			evidence: MCPHTTPEvidenceLegacySSEUnsupported,
			status:   http.StatusMethodNotAllowed,
			message:  "legacy HTTP+SSE endpoint detected; streamable client requires explicit unsupported evidence",
			err:      ErrMCPLegacySSEUnsupported,
		}
	}
	return nil
}

func legacySSEEndpointCarriesSessionID(body []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimRight(scanner.Text(), "\r"))
		if after, ok := strings.CutPrefix(line, "data:"); ok {
			if strings.Contains(after, "sessionId=") {
				return true
			}
		}
	}
	return false
}
