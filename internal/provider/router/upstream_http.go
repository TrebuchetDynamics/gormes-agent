package router

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultHTTPUpstreamTimeout = 30 * time.Second

// HTTPUpstreamProvider adapts a configured OpenAI-compatible upstream route
// (including a user-run CLIProxyAPI endpoint) to the Router's fakeable provider
// boundary. It only talks to the standard /v1/models and /v1/chat/completions
// surfaces for this slice; management APIs, OAuth automation, account pooling,
// WebSockets, and token scraping are intentionally out of scope.
type HTTPUpstreamProvider struct {
	client    *http.Client
	lookupEnv func(string) (string, bool)
}

type HTTPUpstreamProviderOptions struct {
	Client    *http.Client
	LookupEnv func(string) (string, bool)
}

func NewHTTPUpstreamProvider(opts HTTPUpstreamProviderOptions) *HTTPUpstreamProvider {
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPUpstreamTimeout}
	}
	lookupEnv := opts.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	return &HTTPUpstreamProvider{client: client, lookupEnv: lookupEnv}
}

func (p *HTTPUpstreamProvider) Probe(ctx context.Context, route Route) ProbeResult {
	if p == nil {
		return ProbeResult{Evidence: []string{"models_probe_unavailable"}}
	}
	endpoint, err := routeEndpoint(route.BaseURL, "/models")
	if err != nil {
		return ProbeResult{Evidence: []string{"models_probe_invalid_base_url"}}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ProbeResult{Evidence: []string{"models_probe_invalid_request"}}
	}
	if ok := p.applyRouteAuth(req, route); !ok {
		return ProbeResult{Evidence: []string{"models_probe_missing_credential"}}
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return ProbeResult{Evidence: []string{classifyHTTPTransportError(err)}}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return ProbeResult{Available: true, Evidence: []string{"models_probe_available"}}
	}
	return ProbeResult{Evidence: []string{"models_probe_" + httpStatusClass(resp.StatusCode)}}
}

func (p *HTTPUpstreamProvider) ChatCompletion(ctx context.Context, route Route, req ChatCompletionRequest) (ChatCompletionResult, error) {
	if p == nil {
		return ChatCompletionResult{}, ProviderError{Class: "router_provider_unavailable"}
	}
	endpoint, err := routeEndpoint(route.BaseURL, "/chat/completions")
	if err != nil {
		return ChatCompletionResult{}, ProviderError{Class: "invalid_route"}
	}
	body, err := json.Marshal(upstreamChatCompletionRequest(route, req, false))
	if err != nil {
		return ChatCompletionResult{}, ProviderError{Class: "malformed_request"}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return ChatCompletionResult{}, ProviderError{Class: "malformed_request"}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if ok := p.applyRouteAuth(httpReq, route); !ok {
		return ChatCompletionResult{}, ProviderError{Class: "auth"}
	}
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return ChatCompletionResult{}, ProviderError{Class: classifyHTTPTransportError(err)}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return ChatCompletionResult{}, ProviderError{Class: httpStatusClass(resp.StatusCode)}
	}
	var upstream upstreamChatCompletionResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&upstream); err != nil {
		return ChatCompletionResult{}, ProviderError{Class: "upstream_response_invalid"}
	}
	if len(upstream.Choices) == 0 {
		return ChatCompletionResult{}, ProviderError{Class: "upstream_response_invalid"}
	}
	choice := upstream.Choices[0]
	return ChatCompletionResult{
		ID:           strings.TrimSpace(upstream.ID),
		Content:      choice.Message.Content,
		FinishReason: strings.TrimSpace(choice.FinishReason),
		Usage:        upstream.Usage,
	}, nil
}

func (p *HTTPUpstreamProvider) StreamChatCompletion(ctx context.Context, route Route, req ChatCompletionRequest) (ChatStreamResult, error) {
	if p == nil {
		return ChatStreamResult{}, ProviderError{Class: "router_provider_unavailable"}
	}
	endpoint, err := routeEndpoint(route.BaseURL, "/chat/completions")
	if err != nil {
		return ChatStreamResult{}, ProviderError{Class: "invalid_route"}
	}
	body, err := json.Marshal(upstreamChatCompletionRequest(route, req, true))
	if err != nil {
		return ChatStreamResult{}, ProviderError{Class: "malformed_request"}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return ChatStreamResult{}, ProviderError{Class: "malformed_request"}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if ok := p.applyRouteAuth(httpReq, route); !ok {
		return ChatStreamResult{}, ProviderError{Class: "auth"}
	}
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return ChatStreamResult{}, ProviderError{Class: classifyHTTPTransportError(err)}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return ChatStreamResult{}, ProviderError{Class: httpStatusClass(resp.StatusCode)}
	}
	chunks, usage, err := readOpenAICompatibleSSE(resp.Body)
	if err != nil {
		return ChatStreamResult{}, ProviderError{Class: "upstream_response_invalid"}
	}
	return ChatStreamResult{Chunks: chunks, Usage: usage}, nil
}

func upstreamChatCompletionRequest(route Route, req ChatCompletionRequest, stream bool) ChatCompletionRequest {
	out := req
	out.Model = route.Model
	out.Stream = stream
	return out
}

func (p *HTTPUpstreamProvider) applyRouteAuth(req *http.Request, route Route) bool {
	key, ok := p.routeAPIKey(route)
	if !ok {
		return false
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	return true
}

func (p *HTTPUpstreamProvider) routeAPIKey(route Route) (string, bool) {
	resolution := resolveProviderCredential(route, p.lookupEnv)
	if !resolution.Available {
		return "", false
	}
	return resolution.Value, true
}

func routeEndpoint(baseURL, suffix string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", fmt.Errorf("base_url_required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid_base_url")
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + strings.TrimLeft(suffix, "/")
	return parsed.String(), nil
}

func httpStatusClass(status int) string {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return "auth"
	case http.StatusTooManyRequests:
		return "rate_limit"
	case http.StatusRequestTimeout:
		return "timeout"
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return "malformed_request"
	}
	if status >= 500 {
		return "server_error"
	}
	return "upstream_request_failed"
}

func classifyHTTPTransportError(err error) string {
	if err == nil {
		return "upstream_request_failed"
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || os.IsTimeout(err) {
		return "timeout"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	return "connection_failure"
}

type upstreamChatCompletionResponse struct {
	ID      string                         `json:"id"`
	Choices []upstreamChatCompletionChoice `json:"choices"`
	Usage   *Usage                         `json:"usage,omitempty"`
}

type upstreamChatCompletionChoice struct {
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type upstreamChatCompletionChunk struct {
	Choices []struct {
		Delta        openAIChatDelta `json:"delta"`
		FinishReason string          `json:"finish_reason"`
	} `json:"choices"`
	Usage *Usage `json:"usage,omitempty"`
}

func readOpenAICompatibleSSE(body io.Reader) ([]string, *Usage, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	chunks := []string{}
	var usage *Usage
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var chunk upstreamChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, nil, err
		}
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				chunks = append(chunks, choice.Delta.Content)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return chunks, usage, nil
}
