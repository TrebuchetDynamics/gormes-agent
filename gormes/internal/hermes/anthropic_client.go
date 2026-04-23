package hermes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	anthropicMessagesPath = "/v1/messages"
	anthropicModelsPath   = "/v1/models"
	anthropicVersion      = "2023-06-01"
	anthropicMaxTokens    = 4096
)

type anthropicClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func newAnthropicClient(baseURL, apiKey string) Client {
	return &anthropicClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    newStreamingHTTPClient(),
	}
}

type anthropicRequest struct {
	Model     string                    `json:"model"`
	MaxTokens int                       `json:"max_tokens"`
	System    string                    `json:"system,omitempty"`
	Messages  []anthropicMessage        `json:"messages"`
	Stream    bool                      `json:"stream"`
	Tools     []anthropicToolDescriptor `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicContentBlock struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
	Content   string         `json:"content,omitempty"`
}

type anthropicToolDescriptor struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

func (c *anthropicClient) OpenStream(ctx context.Context, req ChatRequest) (Stream, error) {
	body, err := buildAnthropicRequest(req)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+anthropicMessagesPath, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	if c.apiKey != "" {
		httpReq.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		rawBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, &HTTPError{Status: resp.StatusCode, Body: string(rawBody)}
	}
	return newAnthropicStream(resp.Body), nil
}

func (c *anthropicClient) OpenRunEvents(context.Context, string) (RunEventStream, error) {
	return nil, ErrRunEventsNotSupported
}

func (c *anthropicClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+anthropicModelsPath, nil)
	if err != nil {
		return err
	}
	req.Header.Set("anthropic-version", anthropicVersion)
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return &HTTPError{Status: resp.StatusCode, Body: string(body)}
	}
	return nil
}

func buildAnthropicRequest(req ChatRequest) (anthropicRequest, error) {
	system, messages, err := translateAnthropicMessages(req.Messages)
	if err != nil {
		return anthropicRequest{}, err
	}
	tools := make([]anthropicToolDescriptor, len(req.Tools))
	for i, t := range req.Tools {
		tools[i] = anthropicToolDescriptor{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Schema,
		}
	}
	return anthropicRequest{
		Model:     req.Model,
		MaxTokens: anthropicMaxTokens,
		System:    system,
		Messages:  messages,
		Stream:    true,
		Tools:     tools,
	}, nil
}

func translateAnthropicMessages(messages []Message) (string, []anthropicMessage, error) {
	systemParts := make([]string, 0, len(messages))
	out := make([]anthropicMessage, 0, len(messages))

	for i := 0; i < len(messages); i++ {
		msg := messages[i]
		switch msg.Role {
		case "system":
			if strings.TrimSpace(msg.Content) != "" {
				systemParts = append(systemParts, msg.Content)
			}
		case "user":
			out = append(out, anthropicMessage{
				Role:    "user",
				Content: []anthropicContentBlock{{Type: "text", Text: msg.Content}},
			})
		case "assistant":
			blocks := make([]anthropicContentBlock, 0, 1+len(msg.ToolCalls))
			if msg.Content != "" {
				blocks = append(blocks, anthropicContentBlock{Type: "text", Text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				input, err := parseAnthropicToolInput(tc.Arguments)
				if err != nil {
					return "", nil, fmt.Errorf("anthropic assistant tool %q: %w", tc.Name, err)
				}
				blocks = append(blocks, anthropicContentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: input,
				})
			}
			if len(blocks) == 0 {
				continue
			}
			out = append(out, anthropicMessage{Role: "assistant", Content: blocks})
		case "tool":
			blocks := make([]anthropicContentBlock, 0, 1)
			for ; i < len(messages) && messages[i].Role == "tool"; i++ {
				toolMsg := messages[i]
				if strings.TrimSpace(toolMsg.ToolCallID) == "" {
					return "", nil, fmt.Errorf("anthropic tool result missing tool_call_id")
				}
				blocks = append(blocks, anthropicContentBlock{
					Type:      "tool_result",
					ToolUseID: toolMsg.ToolCallID,
					Content:   toolMsg.Content,
				})
			}
			i--
			out = append(out, anthropicMessage{Role: "user", Content: blocks})
		default:
			return "", nil, fmt.Errorf("anthropic unsupported role %q", msg.Role)
		}
	}

	return strings.Join(systemParts, "\n\n"), out, nil
}

func parseAnthropicToolInput(raw json.RawMessage) (map[string]any, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return map[string]any{}, nil
	}
	var input map[string]any
	if err := json.Unmarshal(trimmed, &input); err != nil {
		return nil, err
	}
	if input == nil {
		return map[string]any{}, nil
	}
	return input, nil
}
