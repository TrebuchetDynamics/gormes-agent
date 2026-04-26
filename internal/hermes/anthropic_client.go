package hermes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	anthropicVersion             = "2023-06-01"
	defaultAnthropicMessagesPath = "/v1/messages"
	defaultAnthropicModelsPath   = "/v1/models"
	defaultAnthropicMaxTokens    = 1024
	defaultAzureAnthropicVersion = "2025-04-15"
)

type anthropicClient struct {
	baseURL      string
	apiKey       string
	defaultQuery url.Values
	authMode     anthropicAuthMode
	azure        bool
	apiVersion   string
	http         *http.Client
}

type anthropicAuthMode int

const (
	anthropicAuthAuto anthropicAuthMode = iota
	anthropicAuthXAPIKey
)

type AzureAnthropicClientConfig struct {
	BaseURL    string
	APIKey     string
	APIVersion string
	LookupEnv  func(string) (string, bool)
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Stream    bool               `json:"stream"`
	System    any                `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type anthropicTextBlock struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

type anthropicToolUseBlock struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input any    `json:"input"`
}

type anthropicToolResultBlock struct {
	Type         string        `json:"type"`
	ToolUseID    string        `json:"tool_use_id"`
	Content      string        `json:"content"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// NewAnthropicClient returns a Client that talks directly to Anthropic's
// Messages API over HTTP+SSE.
func NewAnthropicClient(baseURL, apiKey string) Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 5 * time.Second
	return &anthropicClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 0, Transport: transport},
	}
}

func NewAzureAnthropicClient(cfg AzureAnthropicClientConfig) Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 5 * time.Second
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		lookupEnv := cfg.LookupEnv
		if lookupEnv == nil {
			lookupEnv = os.LookupEnv
		}
		if value, ok := lookupEnv("AZURE_ANTHROPIC_KEY"); ok {
			apiKey = strings.TrimSpace(value)
		}
	}
	baseURL, query, apiVersion := normalizeAzureAnthropicBaseURL(cfg.BaseURL, cfg.APIVersion)
	return &anthropicClient{
		baseURL:      baseURL,
		apiKey:       apiKey,
		defaultQuery: query,
		authMode:     anthropicAuthXAPIKey,
		azure:        true,
		apiVersion:   apiVersion,
		http:         &http.Client{Timeout: 0, Transport: transport},
	}
}

func (c *anthropicClient) ProviderStatus() ProviderStatus {
	if c.azure {
		return c.azureProviderStatus()
	}
	return anthropicProviderStatus()
}

func (c *anthropicClient) OpenStream(ctx context.Context, req ChatRequest) (Stream, error) {
	if err := c.requireReady(); err != nil {
		return nil, err
	}
	descriptors := SanitizeToolDescriptors(req.Tools)
	req.Tools = descriptors
	payload, err := buildAnthropicRequest(req)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	endpoint, err := c.endpoint(defaultAnthropicMessagesPath)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	c.applyAuth(httpReq)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, newHTTPError(resp.StatusCode, string(raw), resp.Header)
	}
	return newAnthropicStream(resp.Body, descriptors), nil
}

func (c *anthropicClient) OpenRunEvents(context.Context, string) (RunEventStream, error) {
	return nil, ErrRunEventsNotSupported
}

func (c *anthropicClient) Health(ctx context.Context) error {
	if err := c.requireReady(); err != nil {
		return err
	}
	endpoint, err := c.endpoint(defaultAnthropicModelsPath)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("anthropic-version", anthropicVersion)
	c.applyAuth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return newHTTPError(resp.StatusCode, string(raw), resp.Header)
	}
	return nil
}

func (c *anthropicClient) applyAuth(req *http.Request) {
	if c.apiKey == "" {
		return
	}
	if c.authMode == anthropicAuthXAPIKey {
		req.Header.Set("x-api-key", c.apiKey)
		return
	}
	if strings.HasPrefix(c.apiKey, "sk-ant-api") {
		req.Header.Set("x-api-key", c.apiKey)
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
}

func (c *anthropicClient) endpoint(path string) (string, error) {
	raw := c.baseURL + path
	if len(c.defaultQuery) == 0 {
		return raw, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	for key, values := range c.defaultQuery {
		query.Del(key)
		for _, value := range values {
			query.Add(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (c *anthropicClient) requireReady() error {
	if c.azure && strings.TrimSpace(c.apiKey) == "" {
		return fmt.Errorf("azure_anthropic_key_missing: set AZURE_ANTHROPIC_KEY or configure an Azure Anthropic static key")
	}
	return nil
}

func (c *anthropicClient) azureProviderStatus() ProviderStatus {
	evidence := []string{
		"azure_api_version_query: api-version=" + c.apiVersion,
		"azure_oauth_bypassed: static Azure API key auth",
	}
	keyReady := strings.TrimSpace(c.apiKey) != ""
	if !keyReady {
		evidence = append([]string{"azure_anthropic_key_missing: set AZURE_ANTHROPIC_KEY or configure an Azure Anthropic static key"}, evidence...)
	}
	reason := strings.Join(evidence, "; ")
	return ProviderStatus{
		Provider: "anthropic",
		Runtime:  "azure_anthropic_messages",
		Capabilities: ProviderCapabilities{
			PromptCache:     CapabilityStatus{Available: keyReady, Reason: reason},
			ReasoningEcho:   unavailableCapability("reasoning_content echo padding is not required by azure anthropic messages"),
			RateGuard:       unavailableCapability("provider rate guard not implemented"),
			BudgetTelemetry: unavailableCapability("budget telemetry not implemented"),
		},
	}
}

func normalizeAzureAnthropicBaseURL(rawBaseURL, requestedAPIVersion string) (string, url.Values, string) {
	rawBaseURL = strings.TrimSpace(rawBaseURL)
	apiVersion := strings.TrimSpace(requestedAPIVersion)
	query := url.Values{}
	if parsed, err := url.Parse(rawBaseURL); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		baseQuery := parsed.Query()
		if apiVersion == "" {
			apiVersion = strings.TrimSpace(baseQuery.Get("api-version"))
		}
		if apiVersion == "" {
			apiVersion = defaultAzureAnthropicVersion
		}
		query.Set("api-version", apiVersion)
		parsed.RawQuery = ""
		parsed.ForceQuery = false
		parsed.Path = stripTrailingAzureAnthropicV1(parsed.Path)
		return strings.TrimRight(parsed.String(), "/"), query, apiVersion
	}
	if apiVersion == "" {
		apiVersion = defaultAzureAnthropicVersion
	}
	query.Set("api-version", apiVersion)
	return strings.TrimRight(stripTrailingAzureAnthropicV1(rawBaseURL), "/"), query, apiVersion
}

func stripTrailingAzureAnthropicV1(rawPath string) string {
	trimmed := strings.TrimRight(rawPath, "/")
	if strings.HasSuffix(strings.ToLower(trimmed), "/v1") {
		trimmed = trimmed[:len(trimmed)-len("/v1")]
	}
	return trimmed
}

func buildAnthropicRequest(req ChatRequest) (anthropicRequest, error) {
	system, messages, err := convertAnthropicMessages(req.Messages)
	if err != nil {
		return anthropicRequest{}, err
	}
	tools := make([]anthropicTool, 0, len(req.Tools))
	for _, tool := range SanitizeToolDescriptors(req.Tools) {
		tools = append(tools, anthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.Schema,
		})
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultAnthropicMaxTokens
	}
	return anthropicRequest{
		Model:     req.Model,
		MaxTokens: maxTokens,
		Stream:    true,
		System:    system,
		Messages:  messages,
		Tools:     tools,
	}, nil
}

func convertAnthropicMessages(messages []Message) (any, []anthropicMessage, error) {
	var (
		systemBlocks []anthropicTextBlock
		systemText   []string
		out          []anthropicMessage
	)
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			if msg.CacheControl != nil {
				systemBlocks = append(systemBlocks, textBlock(msg.Content, msg.CacheControl))
				continue
			}
			systemText = append(systemText, msg.Content)
		case "assistant":
			content, err := assistantContentBlocks(msg)
			if err != nil {
				return nil, nil, err
			}
			out = append(out, anthropicMessage{Role: "assistant", Content: content})
		case "tool":
			out = appendAnthropicToolResult(out, msg)
		default:
			if msg.CacheControl != nil {
				out = append(out, anthropicMessage{
					Role:    msg.Role,
					Content: []any{textBlock(msg.Content, msg.CacheControl)},
				})
				continue
			}
			out = append(out, anthropicMessage{Role: msg.Role, Content: msg.Content})
		}
	}
	if len(systemBlocks) > 0 {
		return toAnySlice(systemBlocks), out, nil
	}
	if len(systemText) > 0 {
		return strings.Join(systemText, "\n"), out, nil
	}
	return nil, out, nil
}

func assistantContentBlocks(msg Message) ([]any, error) {
	blocks := make([]any, 0, 1+len(msg.ToolCalls))
	if msg.Content != "" {
		blocks = append(blocks, anthropicTextBlock{Type: "text", Text: msg.Content})
	}
	for _, tc := range msg.ToolCalls {
		input := map[string]any{}
		if len(tc.Arguments) > 0 {
			if err := json.Unmarshal(tc.Arguments, &input); err != nil {
				return nil, err
			}
		}
		blocks = append(blocks, anthropicToolUseBlock{
			Type:  "tool_use",
			ID:    tc.ID,
			Name:  tc.Name,
			Input: input,
		})
	}
	if len(blocks) == 0 {
		blocks = append(blocks, anthropicTextBlock{Type: "text", Text: "(empty)"})
	}
	return blocks, nil
}

func appendAnthropicToolResult(out []anthropicMessage, msg Message) []anthropicMessage {
	block := anthropicToolResultBlock{
		Type:         "tool_result",
		ToolUseID:    msg.ToolCallID,
		Content:      msg.Content,
		CacheControl: msg.CacheControl,
	}
	if block.Content == "" {
		block.Content = "(no output)"
	}
	if len(out) > 0 && out[len(out)-1].Role == "user" {
		if blocks, ok := out[len(out)-1].Content.([]any); ok && startsWithAnthropicToolResult(blocks) {
			out[len(out)-1].Content = append(blocks, block)
			return out
		}
	}
	return append(out, anthropicMessage{Role: "user", Content: []any{block}})
}

func startsWithAnthropicToolResult(blocks []any) bool {
	if len(blocks) == 0 {
		return false
	}
	first, ok := blocks[0].(anthropicToolResultBlock)
	return ok && first.Type == "tool_result"
}

func textBlock(text string, cache *CacheControl) anthropicTextBlock {
	return anthropicTextBlock{Type: "text", Text: text, CacheControl: cache}
}

func toAnySlice[T any](in []T) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, item)
	}
	return out
}
