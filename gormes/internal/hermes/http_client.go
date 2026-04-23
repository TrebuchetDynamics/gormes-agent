package hermes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultChatCompletionsPath = "/v1/chat/completions"
const defaultHealthPath = "/health"

type httpClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewHTTPClient returns a Client that talks HTTP+SSE to a Hermes-compatible
// api_server. baseURL example: "http://127.0.0.1:8642".
// The returned client streams without a global timeout so long turns
// (minutes, with tool use) are not truncated; see per-phase timeouts inside.
func NewHTTPClient(baseURL, apiKey string) Client {
	return &httpClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    newStreamingHTTPClient(),
	}
}

func (c *httpClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+defaultHealthPath, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return newHTTPError(resp, body)
	}
	return nil
}

type orMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type orToolDescriptor struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

type orChatRequest struct {
	Model    string             `json:"model"`
	Messages []orMessage        `json:"messages"`
	Stream   bool               `json:"stream"`
	Tools    []orToolDescriptor `json:"tools,omitempty"`
}

func (c *httpClient) OpenStream(ctx context.Context, req ChatRequest) (Stream, error) {
	msgs := make([]orMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = orMessage{Role: m.Role, Content: m.Content}
	}
	tools := make([]orToolDescriptor, len(req.Tools))
	for i, t := range req.Tools {
		tools[i] = orToolDescriptor{
			Type: "function",
			Function: struct {
				Name        string          `json:"name"`
				Description string          `json:"description"`
				Parameters  json.RawMessage `json:"parameters"`
			}{Name: t.Name, Description: t.Description, Parameters: t.Schema},
		}
	}
	body, err := json.Marshal(orChatRequest{Model: req.Model, Messages: msgs, Stream: true, Tools: tools})
	if err != nil {
		return nil, err
	}

	// Header-phase budget enforced by Transport.ResponseHeaderTimeout (5s).
	// The request ctx governs the full response lifetime including body reads —
	// do NOT cancel it after Do returns or streaming breaks.
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+defaultChatCompletionsPath, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if req.SessionID != "" {
		httpReq.Header.Set("X-Hermes-Session-Id", req.SessionID)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, newHTTPError(resp, raw)
	}
	// The body stays open for streaming; chatStream owns the Close.
	return newChatStream(resp.Body, resp.Header.Get("X-Hermes-Session-Id")), nil
}

// OpenRunEvents subscribes to SSE stream for a run's events.
// 404 returns ErrRunEventsNotSupported for non-Hermes servers.
func (c *httpClient) OpenRunEvents(ctx context.Context, runID string) (RunEventStream, error) {
	// Header-phase budget enforced by Transport.ResponseHeaderTimeout (5s).
	// The request ctx governs the full response lifetime including body reads —
	// do NOT cancel it after Do returns or streaming breaks.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v1/runs/%s/events", c.baseURL, runID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 404 {
		_ = resp.Body.Close()
		return nil, ErrRunEventsNotSupported
	}
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, newHTTPError(resp, raw)
	}
	return newRunEventStream(resp.Body), nil
}
