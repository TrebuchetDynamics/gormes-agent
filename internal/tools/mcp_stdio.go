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
	"sync"
	"sync/atomic"
	"time"
)

// stdioProtocolVersion is the MCP protocol version Gormes negotiates over
// stdio. Hermes' donor pins the same version through its mcp client SDK.
const stdioProtocolVersion = "2024-11-05"

// MCPRawTool is the verbatim tool envelope returned by an MCP server's
// tools/list response. InputSchema is preserved as raw JSON so downstream
// schema normalization can run separately without lossy round-tripping.
type MCPRawTool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

// StdioClientOpts injects the ReadWriteCloser-backed transport plus optional
// observability hooks. Tests pass an in-process net.Pipe pair so no real
// subprocess is spawned.
type StdioClientOpts struct {
	Conn           io.ReadWriteCloser
	Logger         *slog.Logger
	Now            func() time.Time
	ProcessPID     int
	ProcessTracker *MCPStdioProcessTracker
}

// StdioClient speaks JSON-RPC over a stdio.ReadWriteCloser. It is the minimal
// MCP stdio surface needed for `initialize` plus `tools/list`; structured
// content, sampling, OAuth, and `tools/list_changed` notifications live in
// follow-up rows.
type StdioClient struct {
	def    MCPServerDefinition
	conn   io.ReadWriteCloser
	logger *slog.Logger
	now    func() time.Time

	reader  *bufio.Reader
	writeMu sync.Mutex
	readMu  sync.Mutex
	nextID  atomic.Int64

	closeMu sync.Mutex
	closed  bool

	processPID     int
	processTracker *MCPStdioProcessTracker
	processExit    sync.Once

	versionMu       sync.RWMutex
	protocolVersion string
}

// ErrConnectTimeout is reported when the stdio handshake or a tool listing
// fails to complete before the configured connect timeout fires.
var ErrConnectTimeout = errors.New("mcp stdio: connect timeout")

// ErrInitializeFailed wraps any JSON-RPC error returned during the
// `initialize` handshake.
var ErrInitializeFailed = errors.New("mcp stdio: initialize failed")

// ErrInvalidJSONRPCResponse is returned when the peer emits a frame that
// cannot be parsed as JSON-RPC 2.0.
var ErrInvalidJSONRPCResponse = errors.New("mcp stdio: invalid jsonrpc response")

// NewStdioClient constructs a StdioClient over the supplied connection.
func NewStdioClient(def MCPServerDefinition, opts StdioClientOpts) (*StdioClient, error) {
	if opts.Conn == nil {
		return nil, errors.New("mcp stdio: nil conn")
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	processTracker := opts.ProcessTracker
	if processTracker == nil && opts.ProcessPID > 0 {
		processTracker = DefaultMCPStdioProcessTracker
	}
	if processTracker != nil && opts.ProcessPID > 0 {
		processTracker.TrackActivePID(def.Name, opts.ProcessPID)
	}
	return &StdioClient{
		def:            def,
		conn:           opts.Conn,
		logger:         logger,
		now:            now,
		reader:         bufio.NewReader(opts.Conn),
		processPID:     opts.ProcessPID,
		processTracker: processTracker,
	}, nil
}

// ProtocolVersion returns the protocol version the server reported during
// Initialize. Empty before Initialize succeeds.
func (c *StdioClient) ProtocolVersion() string {
	c.versionMu.RLock()
	defer c.versionMu.RUnlock()
	return c.protocolVersion
}

// Initialize performs the MCP handshake and records the negotiated protocol
// version.
func (c *StdioClient) Initialize(ctx context.Context) error {
	params := map[string]any{
		"protocolVersion": stdioProtocolVersion,
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
func (c *StdioClient) ListTools(ctx context.Context) ([]MCPRawTool, error) {
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

// Close releases the underlying transport. Safe to call multiple times.
func (c *StdioClient) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	err := c.conn.Close()
	c.markProcessExit()
	return err
}

func (c *StdioClient) markProcessExit() {
	c.processExit.Do(func() {
		if c.processTracker != nil && c.processPID > 0 {
			c.processTracker.MarkSessionExit(c.def.Name, c.processPID)
		}
	})
}

// jsonRPCRequest is the minimal client-side JSON-RPC 2.0 envelope for the
// stdio MCP request frames Gormes emits.
type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// jsonRPCResponse is the minimal client-side JSON-RPC 2.0 envelope for the
// frames Gormes consumes from the stdio MCP peer.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCError mirrors the error object produced by JSON-RPC 2.0 peers.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *jsonRPCError) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

// call writes a single JSON-RPC request and reads frames until it sees the
// matching response ID. ctx cancellation aborts without blocking on the
// peer; the read goroutine is allowed to drain on its own when conn is
// eventually closed.
func (c *StdioClient) call(ctx context.Context, method string, params any, out any) error {
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
		return fmt.Errorf("mcp stdio: marshal: %w", err)
	}
	data = append(data, '\n')

	type readResult struct {
		resp jsonRPCResponse
		err  error
	}
	ch := make(chan readResult, 1)

	writeDone := make(chan error, 1)
	go func() {
		c.writeMu.Lock()
		defer c.writeMu.Unlock()
		_, werr := c.conn.Write(data)
		writeDone <- werr
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case werr := <-writeDone:
		if werr != nil {
			return fmt.Errorf("mcp stdio: write: %w", werr)
		}
	}

	go func() {
		c.readMu.Lock()
		defer c.readMu.Unlock()
		for {
			line, readErr := c.reader.ReadBytes('\n')
			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				if readErr != nil {
					ch <- readResult{err: readErr}
					return
				}
				continue
			}
			var resp jsonRPCResponse
			if err := json.Unmarshal(line, &resp); err != nil {
				ch <- readResult{err: fmt.Errorf("%w: %v", ErrInvalidJSONRPCResponse, err)}
				return
			}
			if resp.ID == nil || *resp.ID != id {
				if readErr != nil {
					ch <- readResult{err: readErr}
					return
				}
				continue
			}
			ch <- readResult{resp: resp}
			return
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case r := <-ch:
		if r.err != nil {
			return r.err
		}
		if r.resp.Error != nil {
			return r.resp.Error
		}
		if out != nil && len(r.resp.Result) > 0 {
			if err := json.Unmarshal(r.resp.Result, out); err != nil {
				return fmt.Errorf("%w: %v", ErrInvalidJSONRPCResponse, err)
			}
		}
		return nil
	}
}
