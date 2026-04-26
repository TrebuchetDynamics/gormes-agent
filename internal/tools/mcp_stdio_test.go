package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeStdioRequest is the partial JSON-RPC envelope the in-process fake
// server reads from the client side of a net.Pipe pair.
type fakeStdioRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// fakeStdioServer wires a net.Pipe pair so tests can inject responses without
// spawning a real subprocess. The handler returns the JSON-RPC response bytes
// for each request, or nil to drop without responding (used by the timeout
// test).
type fakeStdioServer struct {
	server net.Conn
	client net.Conn
	done   chan struct{}
}

func startFakeStdioServer(t *testing.T, handler func(req fakeStdioRequest) []byte) *fakeStdioServer {
	t.Helper()
	s, c := net.Pipe()
	f := &fakeStdioServer{server: s, client: c, done: make(chan struct{})}
	go func() {
		defer close(f.done)
		rd := bufio.NewReader(s)
		for {
			line, err := rd.ReadBytes('\n')
			if err != nil {
				return
			}
			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			var req fakeStdioRequest
			if err := json.Unmarshal(line, &req); err != nil {
				return
			}
			resp := handler(req)
			if resp == nil {
				continue
			}
			if !bytes.HasSuffix(resp, []byte("\n")) {
				resp = append(resp, '\n')
			}
			if _, err := s.Write(resp); err != nil {
				return
			}
		}
	}()
	return f
}

func (f *fakeStdioServer) Close() {
	_ = f.server.Close()
	_ = f.client.Close()
	<-f.done
}

func newTestStdioClient(t *testing.T, server *fakeStdioServer) *StdioClient {
	t.Helper()
	def := MCPServerDefinition{
		Name:      "fake",
		Enabled:   true,
		Transport: MCPTransportStdio,
		Command:   "fake-mcp",
	}
	client, err := NewStdioClient(def, StdioClientOpts{Conn: server.client})
	if err != nil {
		t.Fatalf("NewStdioClient: %v", err)
	}
	if client == nil {
		t.Fatal("NewStdioClient returned nil client")
	}
	return client
}

func TestStdioClient_InitializeNegotiatesProtocolVersion(t *testing.T) {
	server := startFakeStdioServer(t, func(req fakeStdioRequest) []byte {
		if req.Method != "initialize" {
			t.Errorf("unexpected method %q; want initialize", req.Method)
			return nil
		}
		body, err := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{},
				"serverInfo":      map[string]any{"name": "fake", "version": "0"},
			},
		})
		if err != nil {
			t.Errorf("marshal response: %v", err)
			return nil
		}
		return body
	})
	defer server.Close()

	client := newTestStdioClient(t, server)
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if got := client.ProtocolVersion(); got != "2024-11-05" {
		t.Errorf("ProtocolVersion = %q; want %q", got, "2024-11-05")
	}
}

func TestStdioClient_InitializeReturnsErrInitializeFailedOnError(t *testing.T) {
	server := startFakeStdioServer(t, func(req fakeStdioRequest) []byte {
		body, err := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"error": map[string]any{
				"code":    -32000,
				"message": "boom-during-initialize",
			},
		})
		if err != nil {
			t.Errorf("marshal response: %v", err)
			return nil
		}
		return body
	})
	defer server.Close()

	client := newTestStdioClient(t, server)
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Initialize(ctx)
	if !errors.Is(err, ErrInitializeFailed) {
		t.Fatalf("Initialize err = %v; want errors.Is ErrInitializeFailed", err)
	}
	if !strings.Contains(err.Error(), "boom-during-initialize") {
		t.Errorf("Initialize err = %q; want it to wrap server error message", err.Error())
	}
}

func TestStdioClient_ListToolsParsesToolList(t *testing.T) {
	const schemaA = `{"type":"object","properties":{"x":{"type":"string"}}}`
	server := startFakeStdioServer(t, func(req fakeStdioRequest) []byte {
		if req.Method != "tools/list" {
			t.Errorf("unexpected method %q; want tools/list", req.Method)
			return nil
		}
		// Build the response JSON manually so we can preserve the exact
		// inputSchema bytes the test expects to round-trip verbatim.
		return []byte(`{"jsonrpc":"2.0","id":` + strconv.FormatInt(req.ID, 10) +
			`,"result":{"tools":[` +
			`{"name":"alpha","description":"first tool","inputSchema":` + schemaA + `},` +
			`{"name":"beta","description":"second tool"}` +
			`]}}`)
	})
	defer server.Close()

	client := newTestStdioClient(t, server)
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("got %d tools; want 2", len(tools))
	}
	if tools[0].Name != "alpha" {
		t.Errorf("tools[0].Name = %q; want alpha", tools[0].Name)
	}
	if tools[0].Description != "first tool" {
		t.Errorf("tools[0].Description = %q; want %q", tools[0].Description, "first tool")
	}
	if string(tools[0].InputSchema) != schemaA {
		t.Errorf("tools[0].InputSchema = %s; want %s", string(tools[0].InputSchema), schemaA)
	}
	if tools[1].Name != "beta" {
		t.Errorf("tools[1].Name = %q; want beta", tools[1].Name)
	}
	if tools[1].Description != "second tool" {
		t.Errorf("tools[1].Description = %q; want %q", tools[1].Description, "second tool")
	}
	if len(tools[1].InputSchema) != 0 {
		t.Errorf("tools[1].InputSchema = %s; want empty", string(tools[1].InputSchema))
	}
}

func TestStdioClient_ConnectTimeoutHonoredViaContext(t *testing.T) {
	// Server reads requests but never responds, mimicking a hung MCP child.
	server := startFakeStdioServer(t, func(req fakeStdioRequest) []byte { return nil })
	defer server.Close()

	client := newTestStdioClient(t, server)
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.ListTools(ctx)
	if err == nil {
		t.Fatal("ListTools err = nil; want context.DeadlineExceeded or ErrConnectTimeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, ErrConnectTimeout) {
		t.Fatalf("ListTools err = %v; want errors.Is DeadlineExceeded or ErrConnectTimeout", err)
	}
}

func TestStdioClient_CloseIsIdempotent(t *testing.T) {
	server := startFakeStdioServer(t, func(req fakeStdioRequest) []byte { return nil })
	defer server.Close()

	client := newTestStdioClient(t, server)

	if err := client.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
