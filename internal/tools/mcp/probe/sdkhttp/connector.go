package sdkhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/callresult"
	mcpconfig "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/descriptor"
	mcpprobe "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/probe"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/remote"
)

const (
	maxResponseBytes = 4 << 20
	maxTools         = 1000
)

var (
	ErrUnsupportedTransport = errors.New("mcp probe unsupported transport")
	ErrDeadlineRequired     = errors.New("mcp probe deadline required")
	ErrReservedHeader       = errors.New("mcp probe reserved header")
	ErrResponseTooLarge     = errors.New("mcp probe response too large")
	ErrTooManyTools         = errors.New("mcp probe too many tools")
)

// NewConnector adapts the official MCP Go SDK Streamable HTTP client to the
// package-level probe seam. The returned connector is request-response only:
// it disables retries, redirects, and standalone SSE and applies the caller's
// absolute context deadline to initialization, list requests, and cleanup.
func NewConnector(base *http.Client) mcpprobe.Connector {
	return remote.ProbeConnector(NewSessionConnector(base))
}

// NewSessionConnector returns the live discovery+invocation form of the same
// bounded Streamable HTTP adapter used by NewConnector.
func NewSessionConnector(base *http.Client) remote.Connector {
	return func(ctx context.Context, def mcpconfig.MCPServerDefinition) (remote.Session, error) {
		if def.Transport != mcpconfig.MCPTransportHTTP {
			return nil, ErrUnsupportedTransport
		}
		deadline, ok := ctx.Deadline()
		if !ok {
			return nil, ErrDeadlineRequired
		}
		if hasReservedHeader(def.Headers) {
			return nil, ErrReservedHeader
		}
		client := boundedHTTPClient(base, deadline, def.Headers)
		sdkClient := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "gormes", Version: "1"}, &mcpsdk.ClientOptions{
			Capabilities: &mcpsdk.ClientCapabilities{},
			Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		})
		transport := &mcpsdk.StreamableClientTransport{
			Endpoint:             def.URL,
			HTTPClient:           client,
			MaxRetries:           -1,
			DisableStandaloneSSE: true,
		}
		session, err := sdkClient.Connect(ctx, transport, nil)
		if err != nil {
			return nil, err
		}
		return &sessionAdapter{session: session}, nil
	}
}

type sessionAdapter struct {
	session *mcpsdk.ClientSession
}

func (session *sessionAdapter) ListTools(ctx context.Context) ([]descriptor.RawTool, error) {
	tools := make([]descriptor.RawTool, 0)
	for tool, err := range session.session.Tools(ctx, nil) {
		if err != nil {
			return nil, err
		}
		if len(tools) >= maxTools {
			return nil, ErrTooManyTools
		}
		schema, err := json.Marshal(tool.InputSchema)
		if err != nil {
			return nil, err
		}
		tools = append(tools, descriptor.RawTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schema,
		})
	}
	return tools, nil
}

func (session *sessionAdapter) CallTool(ctx context.Context, name string, arguments map[string]any) (callresult.Result, error) {
	if arguments == nil {
		arguments = map[string]any{}
	}
	result, err := session.session.CallTool(ctx, &mcpsdk.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		return callresult.Result{}, err
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return callresult.Result{}, err
	}
	return callresult.Parse(raw)
}

func (session *sessionAdapter) Close() error {
	return session.session.Close()
}

func boundedHTTPClient(base *http.Client, deadline time.Time, headers map[string]string) *http.Client {
	client := &http.Client{}
	if base != nil {
		*client = *base
	}
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	client.Transport = &policyTransport{
		base:     transport,
		deadline: deadline,
		headers:  cloneHeaders(headers),
	}
	client.Jar = nil
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return client
}

type policyTransport struct {
	base     http.RoundTripper
	deadline time.Time
	headers  map[string]string
}

func (transport *policyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	request := req.Clone(req.Context())
	for name, value := range transport.headers {
		request.Header.Set(name, value)
	}
	var cancel context.CancelFunc
	if current, ok := request.Context().Deadline(); !ok || transport.deadline.Before(current) {
		var ctx context.Context
		ctx, cancel = context.WithDeadline(request.Context(), transport.deadline)
		request = request.Clone(ctx)
	}
	response, err := transport.base.RoundTrip(request)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, err
	}
	bounded := &boundedBody{body: response.Body, remaining: maxResponseBytes, cancel: cancel}
	if request.Method == http.MethodDelete {
		_, readErr := io.Copy(io.Discard, bounded)
		closeErr := bounded.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		response.Body = io.NopCloser(strings.NewReader(""))
		return response, nil
	}
	response.Body = bounded
	return response, nil
}

type boundedBody struct {
	body      io.ReadCloser
	remaining int64
	cancel    context.CancelFunc
}

func (body *boundedBody) Read(dst []byte) (int, error) {
	if body.remaining == 0 {
		var probe [1]byte
		n, err := body.body.Read(probe[:])
		if n > 0 {
			return 0, ErrResponseTooLarge
		}
		return 0, err
	}
	if int64(len(dst)) > body.remaining {
		dst = dst[:body.remaining]
	}
	n, err := body.body.Read(dst)
	body.remaining -= int64(n)
	return n, err
}

func (body *boundedBody) Close() error {
	err := body.body.Close()
	if body.cancel != nil {
		body.cancel()
	}
	return err
}

func hasReservedHeader(headers map[string]string) bool {
	for name := range headers {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "accept", "connection", "content-length", "content-type", "host", "mcp-protocol-version", "mcp-session-id", "proxy-authorization", "transfer-encoding", "upgrade":
			return true
		}
	}
	return false
}

func cloneHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return nil
	}
	out := make(map[string]string, len(headers))
	for name, value := range headers {
		out[name] = value
	}
	return out
}
