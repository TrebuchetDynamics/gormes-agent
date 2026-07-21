package sdkhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	mcpconfig "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
	mcpprobe "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/probe"
)

func TestConnectorUsesOfficialStreamableHTTPPaginationHeadersAndClose(t *testing.T) {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "fixture", Version: "1"}, &mcpsdk.ServerOptions{PageSize: 1})
	server.AddTool(&mcpsdk.Tool{Name: "beta", Description: "second", InputSchema: json.RawMessage(`{"type":"object"}`)}, noopToolHandler)
	server.AddTool(&mcpsdk.Tool{Name: "alpha", Description: "first", InputSchema: json.RawMessage(`{"type":"object"}`)}, noopToolHandler)
	var requests atomic.Int64
	var deletes atomic.Int64
	var gets atomic.Int64
	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return server
	}, &mcpsdk.StreamableHTTPOptions{JSONResponse: true})
	counted := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if got := request.Header.Get("Authorization"); got != "Bearer header-secret" {
			t.Errorf("Authorization = %q", got)
		}
		if request.Method == http.MethodDelete {
			deletes.Add(1)
		}
		if request.Method == http.MethodGet {
			gets.Add(1)
		}
		handler.ServeHTTP(w, request)
	})
	httpServer := httptest.NewServer(counted)
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	tools, err := mcpprobe.One(ctx, mcpconfig.MCPServerDefinition{
		Name:      "fixture",
		Enabled:   true,
		Transport: mcpconfig.MCPTransportHTTP,
		URL:       httpServer.URL,
		Headers:   map[string]string{"Authorization": "Bearer header-secret"},
	}, NewConnector(nil))
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if len(tools) != 2 || tools[0].Name != "alpha" || tools[1].Name != "beta" {
		t.Fatalf("tools = %+v", tools)
	}
	if string(tools[0].InputSchema) != `{"type":"object"}` {
		t.Fatalf("schema = %s", tools[0].InputSchema)
	}
	if requests.Load() < 4 { // initialize, initialized notification, two tools/list pages, and close
		t.Fatalf("requests = %d, want pagination exchange", requests.Load())
	}
	if deletes.Load() != 1 {
		t.Fatalf("DELETE requests = %d, want 1", deletes.Load())
	}
	if gets.Load() != 0 {
		t.Fatalf("GET requests = %d, standalone SSE must be disabled", gets.Load())
	}
}

func TestConnectorDoesNotRetryTransientHTTPFailure(t *testing.T) {
	var posts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPost {
			posts.Add(1)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := NewConnector(nil)(ctx, httpDefinition(server.URL))
	if err == nil {
		t.Fatal("503 probe succeeded")
	}
	if posts.Load() != 1 {
		t.Fatalf("POST requests = %d, want exactly one without retries", posts.Load())
	}
}

func TestConnectorRejectsRedirectWithoutReachingTarget(t *testing.T) {
	var reached atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached.Store(true)
	}))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, &http.Request{}, target.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := NewConnector(nil)(ctx, httpDefinition(redirect.URL))
	if err == nil {
		t.Fatal("redirect probe succeeded")
	}
	if reached.Load() {
		t.Fatal("redirect target was reached")
	}
}

func TestConnectorDeadlineCoversStalledInitialize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(100 * time.Millisecond):
			w.WriteHeader(http.StatusGatewayTimeout)
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := NewConnector(nil)(ctx, httpDefinition(server.URL))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
}

func TestConnectorCapsResponseBodies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.CopyN(w, zeroReader{}, maxResponseBytes+1)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := NewConnector(nil)(ctx, httpDefinition(server.URL))
	if err == nil || !strings.Contains(err.Error(), ErrResponseTooLarge.Error()) {
		t.Fatalf("error = %v, want response-too-large", err)
	}
}

func TestConnectorRejectsReservedHeadersAndMissingDeadlineBeforeNetwork(t *testing.T) {
	transport := &countingTransport{}
	client := &http.Client{Transport: transport}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	definition := httpDefinition("https://example.invalid/mcp")
	definition.Headers = map[string]string{"Mcp-Session-Id": "private"}
	_, err := NewConnector(client)(ctx, definition)
	if !errors.Is(err, ErrReservedHeader) || transport.calls.Load() != 0 {
		t.Fatalf("reserved header error=%v calls=%d", err, transport.calls.Load())
	}
	definition.Headers = nil
	_, err = NewConnector(client)(context.Background(), definition)
	if !errors.Is(err, ErrDeadlineRequired) || transport.calls.Load() != 0 {
		t.Fatalf("missing deadline error=%v calls=%d", err, transport.calls.Load())
	}
}

func noopToolHandler(context.Context, *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	return &mcpsdk.CallToolResult{}, nil
}

func httpDefinition(endpoint string) mcpconfig.MCPServerDefinition {
	return mcpconfig.MCPServerDefinition{Name: "fixture", Enabled: true, Transport: mcpconfig.MCPTransportHTTP, URL: endpoint}
}

type countingTransport struct{ calls atomic.Int64 }

func (transport *countingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	transport.calls.Add(1)
	return nil, errors.New("unexpected network")
}

type zeroReader struct{}

func (zeroReader) Read(dst []byte) (int, error) {
	for i := range dst {
		dst[i] = 'x'
	}
	return len(dst), nil
}
