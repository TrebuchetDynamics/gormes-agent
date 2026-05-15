package tools

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type streamableHTTPRecord struct {
	Method     string
	Path       string
	RPCMethod  string
	SessionID  string
	Accept     string
	StatusPath string
}

func TestMCPStreamableHTTP_CapturesAndReplaysSessionID(t *testing.T) {
	const sessionID = "sess-streamable-123"
	var mu sync.Mutex
	var records []streamableHTTPRecord

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeStreamableHTTPRequest(t, r)
		mu.Lock()
		records = append(records, streamableHTTPRecord{
			Method:    r.Method,
			Path:      r.URL.Path,
			RPCMethod: req.Method,
			SessionID: r.Header.Get("Mcp-Session-Id"),
			Accept:    r.Header.Get("Accept"),
		})
		mu.Unlock()

		switch r.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPost:
			switch req.Method {
			case "initialize":
				w.Header().Set("Mcp-Session-Id", sessionID)
				writeStreamableJSONResult(t, w, req.ID, `{"protocolVersion":"2025-06-18","capabilities":{}}`)
			case "tools/list":
				if got := r.Header.Get("Mcp-Session-Id"); got != sessionID {
					t.Fatalf("tools/list Mcp-Session-Id = %q, want %q", got, sessionID)
				}
				writeStreamableJSONResult(t, w, req.ID, `{"tools":[{"name":"alpha","description":"first"}]}`)
			case "tools/call":
				if got := r.Header.Get("Mcp-Session-Id"); got != sessionID {
					t.Fatalf("tools/call Mcp-Session-Id = %q, want %q", got, sessionID)
				}
				writeStreamableJSONResult(t, w, req.ID, `{"content":[{"type":"text","text":"called"}],"isError":false}`)
			default:
				t.Fatalf("unexpected JSON-RPC method %q", req.Method)
			}
		default:
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()

	client := newTestHTTPClient(t, MCPServerDefinition{
		Name:      "streamable",
		Enabled:   true,
		Transport: MCPTransportHTTP,
		URL:       srv.URL + "/mcp",
	}, srv.Client().Transport)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if got := client.SessionID(); got != sessionID {
		t.Fatalf("SessionID = %q, want %q", got, sessionID)
	}
	if _, err := client.ListTools(ctx); err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if _, err := client.CallTool(ctx, "alpha", map[string]any{"q": "x"}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(records) < 3 {
		t.Fatalf("records len = %d, want at least initialize/list/call: %+v", len(records), records)
	}
	if records[0].RPCMethod != "initialize" || records[0].SessionID != "" {
		t.Fatalf("initialize record = %+v, want no session header", records[0])
	}
	for _, rec := range records[1:3] {
		if rec.SessionID != sessionID {
			t.Fatalf("%s session = %q, want %q (records=%+v)", rec.RPCMethod, rec.SessionID, sessionID, records)
		}
	}
}

func TestMCPStreamableHTTP_SingleEndpointAcceptsJSONOrSSE(t *testing.T) {
	const endpoint = "/mcp"
	var accepts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != endpoint {
			t.Fatalf("request path = %q, want single endpoint %q", r.URL.Path, endpoint)
		}
		req := decodeStreamableHTTPRequest(t, r)
		accepts = append(accepts, r.Header.Get("Accept"))
		switch req.Method {
		case "initialize":
			writeStreamableJSONResult(t, w, req.ID, `{"protocolVersion":"2025-06-18","capabilities":{}}`)
		case "tools/list":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "event: message\n")
			_, _ = io.WriteString(w, "data: {\"jsonrpc\":\"2.0\",\"id\":"+strconv.FormatInt(req.ID, 10)+",\"result\":{\"tools\":[{\"name\":\"sse-tool\",\"description\":\"from sse\"}]}}\n\n")
		default:
			t.Fatalf("unexpected JSON-RPC method %q", req.Method)
		}
	}))
	defer srv.Close()

	client := newTestHTTPClient(t, MCPServerDefinition{
		Name:      "streamable",
		Enabled:   true,
		Transport: MCPTransportHTTP,
		URL:       srv.URL + endpoint,
	}, srv.Client().Transport)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "sse-tool" {
		t.Fatalf("tools = %+v, want SSE-decoded sse-tool", tools)
	}
	for _, accept := range accepts {
		if !strings.Contains(accept, "application/json") || !strings.Contains(accept, "text/event-stream") {
			t.Fatalf("Accept = %q, want JSON and SSE support", accept)
		}
	}
}

func TestMCPStreamableHTTP_ExpiredSessionReinitializes(t *testing.T) {
	const firstSession = "sess-expired"
	const secondSession = "sess-new"
	var initializeSessionHeaders []string
	initCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeStreamableHTTPRequest(t, r)
		switch req.Method {
		case "initialize":
			initializeSessionHeaders = append(initializeSessionHeaders, r.Header.Get("Mcp-Session-Id"))
			initCount++
			if initCount == 1 {
				w.Header().Set("Mcp-Session-Id", firstSession)
			} else {
				w.Header().Set("Mcp-Session-Id", secondSession)
			}
			writeStreamableJSONResult(t, w, req.ID, `{"protocolVersion":"2025-06-18","capabilities":{}}`)
		case "tools/list":
			if r.Header.Get("Mcp-Session-Id") != firstSession {
				t.Fatalf("tools/list session = %q, want expired session %q", r.Header.Get("Mcp-Session-Id"), firstSession)
			}
			http.Error(w, "session expired", http.StatusNotFound)
		default:
			t.Fatalf("unexpected JSON-RPC method %q", req.Method)
		}
	}))
	defer srv.Close()

	client := newTestHTTPClient(t, MCPServerDefinition{
		Name:      "streamable",
		Enabled:   true,
		Transport: MCPTransportHTTP,
		URL:       srv.URL + "/mcp",
	}, srv.Client().Transport)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if _, err := client.ListTools(ctx); !errors.Is(err, ErrMCPSessionExpired) {
		t.Fatalf("ListTools err = %v, want ErrMCPSessionExpired", err)
	} else if got := MCPHTTPErrorEvidence(err); got != MCPHTTPEvidenceSessionExpired {
		t.Fatalf("error evidence = %q, want %q", got, MCPHTTPEvidenceSessionExpired)
	}
	if got := client.SessionID(); got != "" {
		t.Fatalf("SessionID after 404 = %q, want cleared before reinitialize", got)
	}
	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("second Initialize: %v", err)
	}
	if got := client.SessionID(); got != secondSession {
		t.Fatalf("second SessionID = %q, want %q", got, secondSession)
	}
	if len(initializeSessionHeaders) != 2 || initializeSessionHeaders[0] != "" || initializeSessionHeaders[1] != "" {
		t.Fatalf("initialize session headers = %+v, want both empty", initializeSessionHeaders)
	}
}

func TestMCPStreamableHTTP_SessionRequiredEvidence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := decodeStreamableHTTPRequest(t, r)
		if req.Method != "tools/list" {
			t.Fatalf("unexpected JSON-RPC method %q", req.Method)
		}
		if got := r.Header.Get("Mcp-Session-Id"); got != "" {
			t.Fatalf("Mcp-Session-Id = %q, want empty before Initialize", got)
		}
		http.Error(w, "missing session", http.StatusBadRequest)
	}))
	defer srv.Close()

	client := newTestHTTPClient(t, MCPServerDefinition{
		Name:      "streamable",
		Enabled:   true,
		Transport: MCPTransportHTTP,
		URL:       srv.URL + "/mcp",
	}, srv.Client().Transport)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.ListTools(ctx)
	if !errors.Is(err, ErrMCPSessionRequired) {
		t.Fatalf("ListTools err = %v, want ErrMCPSessionRequired", err)
	}
	if got := MCPHTTPErrorEvidence(err); got != MCPHTTPEvidenceSessionRequired {
		t.Fatalf("error evidence = %q, want %q", got, MCPHTTPEvidenceSessionRequired)
	}
}

func TestMCPLegacySSEEndpoint_SessionIDCompatibilityEvidence(t *testing.T) {
	var gotLegacyGET bool
	var messagePosts []url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/sse":
			http.Error(w, "old transport does not accept POST initialize", http.StatusMethodNotAllowed)
		case r.Method == http.MethodGet && r.URL.Path == "/sse":
			gotLegacyGET = true
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "event: endpoint\n")
			_, _ = io.WriteString(w, "data: /message?sessionId=legacy-123\n\n")
		case r.Method == http.MethodPost && r.URL.Path == "/message":
			messagePosts = append(messagePosts, r.URL.Query())
			http.Error(w, "legacy message endpoint should not be used by streamable client", http.StatusBadRequest)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer srv.Close()

	client := newTestHTTPClient(t, MCPServerDefinition{
		Name:      "legacy",
		Enabled:   true,
		Transport: MCPTransportHTTP,
		URL:       srv.URL + "/sse",
	}, srv.Client().Transport)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Initialize(ctx)
	if !errors.Is(err, ErrMCPLegacySSEUnsupported) {
		t.Fatalf("Initialize err = %v, want ErrMCPLegacySSEUnsupported", err)
	}
	if got := MCPHTTPErrorEvidence(err); got != MCPHTTPEvidenceLegacySSEUnsupported {
		t.Fatalf("error evidence = %q, want %q", got, MCPHTTPEvidenceLegacySSEUnsupported)
	}
	if !gotLegacyGET {
		t.Fatal("client did not issue GET fallback to detect legacy SSE endpoint")
	}
	if len(messagePosts) != 0 {
		t.Fatalf("client posted to legacy /message endpoint with query %+v; want explicit unsupported evidence instead", messagePosts)
	}
}

func TestMCPStreamableHTTP_DeleteSessionHeader(t *testing.T) {
	for _, tc := range []struct {
		name         string
		deleteStatus int
	}{
		{name: "no content", deleteStatus: http.StatusNoContent},
		{name: "method not allowed", deleteStatus: http.StatusMethodNotAllowed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const sessionID = "sess-delete"
			var deleteSession string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodPost:
					req := decodeStreamableHTTPRequest(t, r)
					if req.Method != "initialize" {
						t.Fatalf("unexpected JSON-RPC method %q", req.Method)
					}
					w.Header().Set("Mcp-Session-Id", sessionID)
					writeStreamableJSONResult(t, w, req.ID, `{"protocolVersion":"2025-06-18","capabilities":{}}`)
				case http.MethodDelete:
					deleteSession = r.Header.Get("Mcp-Session-Id")
					w.WriteHeader(tc.deleteStatus)
				default:
					http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
				}
			}))
			defer srv.Close()

			client, err := NewHTTPClient(MCPServerDefinition{
				Name:      "streamable",
				Enabled:   true,
				Transport: MCPTransportHTTP,
				URL:       srv.URL + "/mcp",
			}, HTTPClientOpts{Transport: srv.Client().Transport})
			if err != nil {
				t.Fatalf("NewHTTPClient: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			if err := client.Initialize(ctx); err != nil {
				t.Fatalf("Initialize: %v", err)
			}
			if err := client.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}
			if deleteSession != sessionID {
				t.Fatalf("DELETE Mcp-Session-Id = %q, want %q", deleteSession, sessionID)
			}
		})
	}
}

func decodeStreamableHTTPRequest(t *testing.T, r *http.Request) httpRequest {
	t.Helper()
	if r.Method != http.MethodPost {
		return httpRequest{}
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	var req httpRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("decode request body %q: %v", string(body), err)
	}
	return req
}

func writeStreamableJSONResult(t *testing.T, w http.ResponseWriter, id int64, result string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":`+strconv.FormatInt(id, 10)+`,"result":`+result+`}`)
}
