package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const defaultChatCompletionsPath = "/v1/chat/completions"
const defaultHealthPath = "/health"

type httpClient struct {
	baseURL                   string
	apiKey                    string
	provider                  string
	http                      *http.Client
	mu                        sync.Mutex
	temperatureRetry          ProviderTemperatureRetryStatus
	parameterRetry            ProviderUnsupportedParameterRetryStatus
	visionUnsupportedSessions map[string]bool
	rateGuardMu               sync.Mutex
	rateGuard                 GuardState
	rateLimitMu               sync.Mutex
	rateLimit                 RateLimitState
	onCredentialExhausted     func(statusCode int, reason string, headers http.Header)
}

// NewHTTPClient returns a Client that talks HTTP+SSE to a
// Hermes/Gormes-compatible provider endpoint.
// The returned client streams without a global timeout so long turns
// (minutes, with tool use) are not truncated; see per-phase timeouts inside.
func NewHTTPClient(baseURL, apiKey string) Client {
	return NewHTTPClientWithProvider(baseURL, apiKey, "")
}

// NewHTTPClientWithProvider returns an OpenAI-compatible HTTP client with a
// provider identity hint for providers whose replay rules differ from the
// generic Chat Completions shape.
func NewHTTPClientWithProvider(baseURL, apiKey, provider string) Client {
	// Clone the default transport and enforce the header-phase budget via
	// ResponseHeaderTimeout. This caps time-to-first-byte WITHOUT affecting
	// the streaming body read afterwards — unlike wrapping the request
	// context, which would cancel body reads mid-stream. Keep the budget
	// provider-friendly because routers such as OpenRouter can spend more than
	// a few seconds selecting an upstream before the first SSE byte arrives.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = defaultProviderResponseHeaderTimeout(provider, baseURL)
	return &httpClient{
		baseURL:  baseURL,
		apiKey:   apiKey,
		provider: strings.TrimSpace(provider),
		http:     &http.Client{Timeout: 0, Transport: transport},
	}
}

func defaultProviderResponseHeaderTimeout(provider, baseURL string) time.Duration {
	if IsOpenRouterRoute(provider, baseURL) {
		return 60 * time.Second
	}
	return 30 * time.Second
}

func (c *httpClient) ProviderStatus() ProviderStatus {
	var status ProviderStatus
	if c.usesCodexResponsesTransport() {
		status = codexResponsesProviderStatus()
	} else if c.usesGeminiNativeTransport() {
		status = geminiNativeProviderStatus()
	} else {
		status = openAICompatibleProviderStatus(c.provider, c.baseURL)
	}
	if !c.usesCodexResponsesTransport() && openAICompatibleIsAzureOpenAI(c.provider, c.baseURL) {
		evidence := []string{"azure_chat_completions"}
		if openAICompatibleBaseURLHasQuery(c.baseURL) {
			evidence = append([]string{"azure_query_preserved"}, evidence...)
		}
		status.Capabilities.BudgetTelemetry.Reason = appendProviderStatusEvidence(status.Capabilities.BudgetTelemetry.Reason, evidence...)
	}
	c.mu.Lock()
	status.TemperatureRetry = c.temperatureRetry
	status.UnsupportedParameterRetry = c.parameterRetry
	c.mu.Unlock()
	c.rateLimitMu.Lock()
	status.RateLimit = c.rateLimit
	c.rateLimitMu.Unlock()
	c.rateGuardMu.Lock()
	rg := c.rateGuard
	c.rateGuardMu.Unlock()
	if rg.LastKnownClass == RateLimitGenuineQuota {
		status.Capabilities.RateGuard = CapabilityStatus{
			Available: true,
			Reason:    string(StatusNousRateLimited),
		}
	} else if rg.LastKnownClass == RateLimitUpstreamCapacity {
		status.Capabilities.RateGuard = CapabilityStatus{
			Available: true,
			Reason:    string(StatusNousUpstreamCapacity),
		}
	}
	return status
}

func (c *httpClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+defaultHealthPath, nil)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	c.applyCodexCloudflareHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return newHTTPError(resp.StatusCode, string(body), resp.Header)
	}
	return nil
}

type orMessage struct {
	Role             string       `json:"role"`
	Content          any          `json:"content"`
	ReasoningContent *string      `json:"reasoning_content,omitempty"`
	ToolCalls        []orToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string       `json:"tool_call_id,omitempty"`
	Name             string       `json:"name,omitempty"`
}

type orToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function orToolFunction `json:"function"`
}

type orToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
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
	Model               string             `json:"model"`
	Messages            []orMessage        `json:"messages"`
	Stream              bool               `json:"stream"`
	MaxTokens           int                `json:"max_tokens,omitempty"`
	MaxCompletionTokens int                `json:"max_completion_tokens,omitempty"`
	Temperature         *float64           `json:"temperature,omitempty"`
	ReasoningEffort     *ReasoningEffort   `json:"reasoning_effort,omitempty"`
	ServiceTier         string             `json:"service_tier,omitempty"`
	ExtraBody           map[string]any     `json:"extra_body,omitempty"`
	Tools               []orToolDescriptor `json:"tools,omitempty"`
	// Verbosity routes reasoning effort for OpenRouter adaptive Anthropic models
	// (Claude 4.6+) that use adaptive thinking and ignore reasoning.effort.
	// Mirrors Hermes fix(openrouter): route reasoning_effort to verbosity for
	// adaptive Anthropic models (183d86b3e).
	Verbosity string `json:"verbosity,omitempty"`
}

func (c *httpClient) OpenStream(ctx context.Context, req ChatRequest) (Stream, error) {
	req = c.prepareVisionUnsupportedRequest(req)
	if c.usesCodexResponsesTransport() {
		return c.openCodexResponsesStream(ctx, req)
	}
	if c.usesGeminiNativeTransport() {
		return c.openGeminiNativeStream(ctx, req)
	}

	body, descriptors, err := c.buildOpenAICompatibleChatRequestBody(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.doChatCompletions(ctx, req, body)
	if err != nil {
		return nil, err
	}
	c.updateRateLimitFromHeaders(resp.Header)
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		httpErr := newHTTPError(resp.StatusCode, string(raw), resp.Header)
		if resp.StatusCode == http.StatusTooManyRequests {
			c.updateRateGuardFrom429(resp.Header)
		}
		if resp.StatusCode == http.StatusUnauthorized {
			c.handleUnauthorized(resp.Header)
		}
		if retryReq, ok := c.planVisionUnsupportedRetry(req, httpErr); ok {
			retryBody, retryDescriptors, err := c.buildOpenAICompatibleChatRequestBody(retryReq)
			if err != nil {
				return nil, err
			}
			retryResp, err := c.doChatCompletions(ctx, retryReq, retryBody)
			if err != nil {
				return nil, err
			}
			c.updateRateLimitFromHeaders(retryResp.Header)
			if retryResp.StatusCode >= 300 {
				retryRaw, _ := io.ReadAll(retryResp.Body)
				_ = retryResp.Body.Close()
				return nil, newHTTPError(retryResp.StatusCode, string(retryRaw), retryResp.Header)
			}
			return newChatStreamWithDiagnostics(retryResp.Body, retryResp.Header.Get("X-Hermes-Session-Id"), retryDescriptors, streamDiagnosticsFromResponse(retryResp)), nil
		}
		if req.Temperature != nil && requestBodyHasParameter(body, "temperature") && isUnsupportedTemperatureError(httpErr) {
			c.recordTemperatureRetry(req.Model, httpErr)
			retryReq := req
			retryReq.Temperature = nil
			retryBody, retryDescriptors, err := c.buildOpenAICompatibleChatRequestBody(retryReq)
			if err != nil {
				return nil, err
			}
			retryResp, err := c.doChatCompletions(ctx, retryReq, retryBody)
			if err != nil {
				return nil, err
			}
			c.updateRateLimitFromHeaders(retryResp.Header)
			if retryResp.StatusCode >= 300 {
				retryRaw, _ := io.ReadAll(retryResp.Body)
				_ = retryResp.Body.Close()
				return nil, newHTTPError(retryResp.StatusCode, string(retryRaw), retryResp.Header)
			}
			return newChatStreamWithDiagnostics(retryResp.Body, retryResp.Header.Get("X-Hermes-Session-Id"), retryDescriptors, streamDiagnosticsFromResponse(retryResp)), nil
		}
		if req.MaxTokens > 0 && requestBodyHasParameter(body, "max_tokens") && isUnsupportedParameterError(httpErr, "max_tokens") {
			c.recordUnsupportedParameterRetry(req.Model, "max_tokens", "max_completion_tokens", httpErr)
			retryBody, err := replaceMaxTokensWithMaxCompletionTokens(body, req.MaxTokens)
			if err != nil {
				return nil, err
			}
			retryResp, err := c.doChatCompletions(ctx, req, retryBody)
			if err != nil {
				return nil, err
			}
			c.updateRateLimitFromHeaders(retryResp.Header)
			if retryResp.StatusCode >= 300 {
				retryRaw, _ := io.ReadAll(retryResp.Body)
				_ = retryResp.Body.Close()
				return nil, newHTTPError(retryResp.StatusCode, string(retryRaw), retryResp.Header)
			}
			return newChatStreamWithDiagnostics(retryResp.Body, retryResp.Header.Get("X-Hermes-Session-Id"), descriptors, streamDiagnosticsFromResponse(retryResp)), nil
		}
		return nil, httpErr
	}
	// The body stays open for streaming; chatStream owns the Close.
	return newChatStreamWithDiagnostics(resp.Body, resp.Header.Get("X-Hermes-Session-Id"), descriptors, streamDiagnosticsFromResponse(resp)), nil
}

func streamDiagnosticsFromResponse(resp *http.Response) StreamDiagnostics {
	if resp == nil {
		return StreamDiagnostics{}
	}
	return StreamDiagnostics{
		HTTPStatus: resp.StatusCode,
		Headers:    captureStreamDiagnosticHeaders(resp.Header),
	}
}

func (c *httpClient) buildOpenAICompatibleChatRequestBody(req ChatRequest) ([]byte, []ToolDescriptor, error) {
	policy := PromptCachePolicyFor(PromptCachePolicyInput{
		Provider: c.provider,
		BaseURL:  c.baseURL,
		APIMode:  "chat_completions",
		Model:    req.Model,
	})
	msgs := makeOpenAICompatibleMessages(ApplyPromptCacheControl(req.Messages, policy), c.provider, req.Model, c.baseURL)
	reasoningEffort, err := validateReasoningEffort(req.ReasoningEffort)
	if err != nil {
		return nil, nil, err
	}
	descriptors := SanitizeToolDescriptorsForModel(req.Model, req.Tools)
	tools := make([]orToolDescriptor, len(descriptors))
	for i, t := range descriptors {
		tools[i] = orToolDescriptor{
			Type: "function",
			Function: struct {
				Name        string          `json:"name"`
				Description string          `json:"description"`
				Parameters  json.RawMessage `json:"parameters"`
			}{Name: t.Name, Description: t.Description, Parameters: t.Schema},
		}
	}
	maxTokens, maxCompletionTokens := openAICompatibleMaxTokenFields(req.MaxTokens, c.provider, c.baseURL, req.Model)
	// For OpenRouter adaptive Anthropic models (Claude 4.6+), reasoning.effort
	// is ignored; send verbosity instead and suppress reasoning_effort.
	verbosity := openRouterAdaptiveAnthropicVerbosity(c.provider, c.baseURL, req.Model, reasoningEffort)
	adaptiveAnthropicEffort := reasoningEffort
	if verbosity != "" {
		adaptiveAnthropicEffort = nil
	}
	body, err := json.Marshal(orChatRequest{
		Model:               req.Model,
		Messages:            msgs,
		Stream:              true,
		MaxTokens:           maxTokens,
		MaxCompletionTokens: maxCompletionTokens,
		Temperature:         req.Temperature,
		ReasoningEffort:     adaptiveAnthropicEffort,
		ServiceTier:         normalizeServiceTier(req.RequestOverrides.ServiceTier),
		ExtraBody:           buildOpenRouterParetoExtraBody(c.provider, c.baseURL, req.Model, req.RequestOverrides.OpenRouterMinCodingScore),
		Tools:               tools,
		Verbosity:           verbosity,
	})
	if err != nil {
		return nil, nil, err
	}
	return body, descriptors, nil
}

func normalizeServiceTier(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "priority":
		return "priority"
	default:
		return ""
	}
}

func (c *httpClient) openCodexResponsesStream(ctx context.Context, req ChatRequest) (Stream, error) {
	transport := codexResponsesTransport{}
	providerReq, err := transport.BuildRequest(req)
	if err != nil {
		return nil, err
	}

	chatGPTCodexBackend := codexChatGPTBackendBaseURL(c.baseURL)
	accept := "application/json"
	if chatGPTCodexBackend {
		providerReq.Body, err = codexResponsesStreamingBody(providerReq.Body)
		if err != nil {
			return nil, err
		}
		accept = "text/event-stream"
	}

	resp, err := c.doProviderPost(ctx, req.SessionID, req.Model, providerReq.Path, providerReq.Body, accept)
	if err != nil {
		return nil, err
	}
	c.updateRateLimitFromHeaders(resp.Header)
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		httpErr := newHTTPError(resp.StatusCode, string(raw), resp.Header)
		if resp.StatusCode == http.StatusTooManyRequests {
			c.updateRateGuardFrom429(resp.Header)
		}
		if resp.StatusCode == http.StatusUnauthorized {
			c.handleUnauthorized(resp.Header)
		}
		if retryReq, ok := c.planVisionUnsupportedRetry(req, httpErr); ok {
			retryProviderReq, err := transport.BuildRequest(retryReq)
			if err != nil {
				return nil, err
			}
			if chatGPTCodexBackend {
				retryProviderReq.Body, err = codexResponsesStreamingBody(retryProviderReq.Body)
				if err != nil {
					return nil, err
				}
			}
			retryResp, err := c.doProviderPost(ctx, retryReq.SessionID, retryReq.Model, retryProviderReq.Path, retryProviderReq.Body, accept)
			if err != nil {
				return nil, err
			}
			c.updateRateLimitFromHeaders(retryResp.Header)
			if retryResp.StatusCode >= 300 {
				retryRaw, _ := io.ReadAll(retryResp.Body)
				_ = retryResp.Body.Close()
				return nil, newHTTPError(retryResp.StatusCode, string(retryRaw), retryResp.Header)
			}
			if chatGPTCodexBackend {
				return newCodexResponsesSSEStream(ctx, retryResp.Body, retryProviderReq)
			}
			return transport.OpenFixtureStream(retryResp.Body, retryProviderReq)
		}
		return nil, httpErr
	}
	if chatGPTCodexBackend {
		return newCodexResponsesSSEStream(ctx, resp.Body, providerReq)
	}
	return transport.OpenFixtureStream(resp.Body, providerReq)
}

func validateReasoningEffort(effort *ReasoningEffort) (*ReasoningEffort, error) {
	if effort == nil {
		return nil, nil
	}
	normalized, ok := NormalizeReasoningEffort(*effort)
	if !ok {
		return nil, fmt.Errorf("invalid reasoning_effort %q; valid values are none, minimal, low, medium, high, xhigh", *effort)
	}
	return &normalized, nil
}

func requestBodyHasParameter(body []byte, param string) bool {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return false
	}
	_, ok := obj[param]
	return ok
}

func replaceMaxTokensWithMaxCompletionTokens(body []byte, maxTokens int) ([]byte, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, err
	}
	delete(obj, "max_tokens")
	rawMaxTokens, err := json.Marshal(maxTokens)
	if err != nil {
		return nil, err
	}
	obj["max_completion_tokens"] = rawMaxTokens
	return json.Marshal(obj)
}

func (c *httpClient) doChatCompletions(ctx context.Context, req ChatRequest, body []byte) (*http.Response, error) {
	return c.doProviderPost(ctx, req.SessionID, req.Model, defaultChatCompletionsPath, body, "text/event-stream")
}

func (c *httpClient) doProviderPost(ctx context.Context, sessionID, model, endpointPath string, body []byte, accept string) (*http.Response, error) {
	if err := c.checkRateGuard(); err != nil {
		return nil, err
	}
	// Header-phase budget enforced by Transport.ResponseHeaderTimeout (5s).
	// The request ctx governs the full response lifetime including body reads —
	// do NOT cancel it after Do returns or streaming breaks.
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.providerPostURL(endpointPath), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(accept) != "" {
		httpReq.Header.Set("Accept", accept)
	}
	if c.apiKey != "" {
		if c.usesGeminiNativeTransport() {
			httpReq.Header.Set("x-goog-api-key", c.apiKey)
		} else {
			httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		}
	}
	ApplyOpenRouterAttributionHeaders(httpReq, c.provider, c.baseURL)
	ApplyOpenRouterGrokPromptCacheAffinityHeader(httpReq, c.provider, c.baseURL, model, sessionID)
	c.applyCodexCloudflareHeaders(httpReq)
	if cacheScopeID := codexResponsesPromptCacheKey(sessionID); cacheScopeID != "" && c.usesCodexResponsesTransport() && codexChatGPTBackendBaseURL(c.baseURL) {
		httpReq.Header.Set("session_id", cacheScopeID)
		httpReq.Header.Set("x-client-request-id", cacheScopeID)
	}
	if sessionID != "" {
		httpReq.Header.Set("X-Hermes-Session-Id", sessionID)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *httpClient) providerPostURL(endpointPath string) string {
	if c.usesCodexResponsesTransport() && codexChatGPTBackendBaseURL(c.baseURL) && endpointPath == defaultCodexResponsesPath {
		return c.openAICompatibleURL("/responses")
	}
	return c.openAICompatibleURL(endpointPath)
}

func (c *httpClient) usesCodexResponsesTransport() bool {
	provider := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(c.provider)), "_", "-")
	return provider == "openai-codex" || provider == "codex"
}

func (c *httpClient) usesGeminiNativeTransport() bool {
	provider := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(c.provider)), "_", "-")
	if provider != "gemini" && provider != "google" && provider != "google-gemini" {
		return false
	}
	return !strings.Contains(strings.ToLower(strings.TrimSpace(c.baseURL)), "/openai")
}

func (c *httpClient) openGeminiNativeStream(ctx context.Context, req ChatRequest) (Stream, error) {
	transport := geminiNativeTransport{}
	providerReq, err := transport.BuildRequest(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.doProviderPost(ctx, req.SessionID, req.Model, providerReq.Path, providerReq.Body, "text/event-stream")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, newHTTPError(resp.StatusCode, string(raw), resp.Header)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return geminiNativeSSEStream(raw, providerReq.ToolDescriptors), nil
}

func geminiNativeSSEStream(raw []byte, descriptors []ToolDescriptor) Stream {
	var frames []geminiStreamEvent
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var frame geminiStreamEvent
		if err := json.Unmarshal([]byte(data), &frame); err == nil {
			frames = append(frames, frame)
		}
	}
	encoded, _ := json.Marshal(frames)
	return newGeminiCloudCodeStream(encoded, descriptors)
}

func (c *httpClient) openAICompatibleURL(endpointPath string) string {
	rawBaseURL := strings.TrimSpace(c.baseURL)
	if rawBaseURL == "" {
		return endpointPath
	}
	parsed, err := url.Parse(rawBaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return rawBaseURL + endpointPath
	}

	basePath := strings.TrimRight(parsed.Path, "/")
	endpointPath = "/" + strings.TrimLeft(endpointPath, "/")
	endpointRawQuery := ""
	if pathPart, queryPart, ok := strings.Cut(endpointPath, "?"); ok {
		endpointPath = pathPart
		endpointRawQuery = queryPart
	}
	// Collapse a `/v1` prefix when both basePath and endpointPath carry it.
	// Live regression 2026-05-10: operators copy-pasting OpenRouter's
	// documented base URL `https://openrouter.ai/api/v1` produced a final
	// request URL of `https://openrouter.ai/api/v1/v1/chat/completions` and
	// got "Not Found: provider returned HTML error body" with no
	// indication that the path was double-prefixed. Strip the basePath's
	// trailing `/v1` whenever the endpointPath starts with `/v1/` so both
	// shapes (`endpoint = '.../api'` and `endpoint = '.../api/v1'`)
	// resolve to the same correct URL. This matches operator intuition
	// across OpenAI-compatible providers (OpenAI itself, OpenRouter,
	// Together, Groq chat, DeepInfra, etc.) whose docs include /v1 in the
	// advertised base URL.
	if strings.HasPrefix(endpointPath, "/v1/") && strings.HasSuffix(basePath, "/v1") {
		basePath = strings.TrimSuffix(basePath, "/v1")
	}
	if basePath == "" {
		parsed.Path = endpointPath
	} else {
		parsed.Path = basePath + endpointPath
	}
	parsed.RawPath = ""
	if endpointRawQuery != "" {
		if parsed.RawQuery == "" {
			parsed.RawQuery = endpointRawQuery
		} else {
			parsed.RawQuery = parsed.RawQuery + "&" + endpointRawQuery
		}
	}
	return parsed.String()
}

func openAICompatibleMaxTokenFields(maxTokens int, provider, baseURL, model string) (int, int) {
	if maxTokens <= 0 {
		// Ollama/LM-Studio/local and custom providers default to a tiny
		// num_predict (128 tokens) when max_tokens is omitted, truncating
		// most agent responses. Send a generous floor so the provider behaves
		// like any cloud provider while still letting the operator set a
		// per-model cap. Mirrors Hermes fix(ollama): set default_max_tokens
		// for custom/Ollama provider (09ec26c66).
		if openAICompatibleNeedsDefaultMaxTokens(provider, baseURL) {
			maxTokens = 65536
		} else {
			return 0, 0
		}
	}
	// Pre-emptively send max_completion_tokens for model families that
	// reject max_tokens regardless of the endpoint hostname — mirrors
	// Hermes utils.py model_forces_max_completion_tokens (fix 19c07c403).
	if openAICompatibleUsesMaxCompletionTokens(model) {
		return 0, maxTokens
	}
	return maxTokens, 0
}

// openAICompatibleNeedsDefaultMaxTokens returns true for local/self-hosted
// providers that truncate responses at a tiny default when max_tokens is absent.
// Only matches explicit provider IDs — not generic loopback URLs — to avoid
// false positives for test servers that happen to run on localhost.
func openAICompatibleNeedsDefaultMaxTokens(provider, _ string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "custom", "ollama", "local", "lmstudio", "lm-studio", "lm_studio", "vllm":
		return true
	}
	return false
}

func openAICompatibleIsAzureOpenAI(provider, baseURL string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch strings.ReplaceAll(provider, "_", "-") {
	case "azure", "azure-openai", "openai-azure":
		return true
	}
	return strings.Contains(strings.ToLower(baseURL), "openai.azure.com")
}

// openAICompatibleUsesMaxCompletionTokens returns true when the model family
// requires max_completion_tokens instead of max_tokens. Hermes uses prefix
// matching on the normalized (lower-cased, vendor-prefix-stripped) name.
// Families: gpt-4o, gpt-4.1, gpt-5+, o1, o3, o4 (fix 19c07c403).
func openAICompatibleUsesMaxCompletionTokens(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	if slash := strings.LastIndex(model, "/"); slash >= 0 {
		model = model[slash+1:]
	}
	for _, prefix := range []string{"gpt-4o", "gpt-4.1", "gpt-5"} {
		if model == prefix || strings.HasPrefix(model, prefix+"-") || strings.HasPrefix(model, prefix+".") {
			return true
		}
	}
	return openAICompatibleIsOSeriesModel(model)
}

func openAICompatibleIsOSeriesModel(model string) bool {
	for _, prefix := range []string{"o1", "o3", "o4", "o5"} {
		if model == prefix || strings.HasPrefix(model, prefix+"-") {
			return true
		}
	}
	return false
}

func openAICompatibleBaseURLHasQuery(baseURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	return err == nil && parsed.RawQuery != ""
}

func appendProviderStatusEvidence(reason string, evidence ...string) string {
	reason = strings.TrimSpace(reason)
	for _, item := range evidence {
		item = strings.TrimSpace(item)
		if item == "" || strings.Contains(reason, item) {
			continue
		}
		if reason == "" {
			reason = item
			continue
		}
		reason += "; " + item
	}
	return reason
}

func (c *httpClient) recordTemperatureRetry(model string, err *HTTPError) {
	reason := providerRetryReason(err)
	c.mu.Lock()
	c.temperatureRetry = ProviderTemperatureRetryStatus{
		Attempts: 1,
		Stripped: true,
		Model:    model,
		Reason:   reason,
	}
	c.parameterRetry = ProviderUnsupportedParameterRetryStatus{
		Attempts:  1,
		Parameter: "temperature",
		Stripped:  true,
		Model:     model,
		Reason:    reason,
	}
	c.mu.Unlock()
}

func (c *httpClient) recordUnsupportedParameterRetry(model, param, replacement string, err *HTTPError) {
	reason := providerRetryReason(err)
	c.mu.Lock()
	c.parameterRetry = ProviderUnsupportedParameterRetryStatus{
		Attempts:    1,
		Parameter:   param,
		Replacement: replacement,
		Stripped:    true,
		Model:       model,
		Reason:      reason,
	}
	c.mu.Unlock()
}

func providerRetryReason(err *HTTPError) string {
	if err == nil {
		return ""
	}
	reason, _, _ := providerHTTPErrorText(err)
	if reason == "" {
		reason = err.Body
	}
	return strings.TrimSpace(reason)
}

func makeOpenAICompatibleMessages(messages []Message, provider, model, baseURL string) []orMessage {
	out := make([]orMessage, 0, len(messages))
	promptRole := string(ModelPromptRole(model))
	for idx, msg := range messages {
		role := msg.Role
		if idx == 0 && role == "system" && promptRole == string(PromptRoleDeveloper) {
			role = promptRole
		}
		wire := orMessage{
			Role:       role,
			Content:    openAICompatibleMessageContent(msg),
			ToolCallID: msg.ToolCallID,
			Name:       msg.Name,
		}
		if msg.Role == "assistant" {
			wire.ReasoningContent = openAICompatibleReasoningContent(msg, provider, model, baseURL)
		}
		if len(msg.ToolCalls) > 0 {
			wire.ToolCalls = make([]orToolCall, 0, len(msg.ToolCalls))
			for _, call := range msg.ToolCalls {
				args := string(call.Arguments)
				if args == "" {
					args = "{}"
				}
				wire.ToolCalls = append(wire.ToolCalls, orToolCall{
					ID:   call.ID,
					Type: "function",
					Function: orToolFunction{
						Name:      call.Name,
						Arguments: args,
					},
				})
			}
		}
		out = append(out, wire)
	}
	return out
}

func openAICompatibleMessageContent(msg Message) any {
	if len(msg.ContentParts) > 0 {
		parts := make([]map[string]any, 0, len(msg.ContentParts))
		for _, part := range msg.ContentParts {
			switch strings.ToLower(strings.TrimSpace(part.Type)) {
			case "text", "input_text", "output_text":
				if part.Text == "" {
					continue
				}
				parts = append(parts, map[string]any{
					"type": "text",
					"text": part.Text,
				})
			case "image_url", "input_image", "image":
				if part.ImageURL == "" {
					continue
				}
				image := map[string]any{"url": part.ImageURL}
				if strings.TrimSpace(part.Detail) != "" {
					image["detail"] = strings.TrimSpace(part.Detail)
				}
				parts = append(parts, map[string]any{
					"type":      "image_url",
					"image_url": image,
				})
			}
		}
		if len(parts) > 0 {
			return parts
		}
	}
	if msg.CacheControl == nil {
		return msg.Content
	}
	return []map[string]any{{
		"type":          "text",
		"text":          msg.Content,
		"cache_control": msg.CacheControl,
	}}
}

func openAICompatibleReasoningContent(msg Message, provider, model, baseURL string) *string {
	if !openAICompatibleRequiresReasoningEcho(provider, model, baseURL) {
		return nil
	}
	// Step 1: explicit reasoning_content (wire-level field from prior DeepSeek/Kimi turn).
	// Upgrade "" → " " for DeepSeek V4 Pro which rejects empty strings (Hermes #17341).
	if msg.ReasoningContent != nil {
		rc := *msg.ReasoningContent
		if rc == "" {
			rc = " "
		}
		return &rc
	}
	// No wire-level reasoning_content available. Gormes isolates generic internal Reasoning
	// (cross-provider storage from any prior provider) from the wire format — the msg.Reasoning
	// field is never promoted to reasoning_content to prevent cross-provider chain-of-thought
	// leakage. Pad with a single space to satisfy providers that require the field.
	// Hermes: "Space (not "") because DeepSeek V4 Pro tightened validation" (refs #17341).
	space := " "
	return &space
}

func openAICompatibleRequiresReasoningEcho(provider, model, baseURL string) bool {
	return openAICompatibleNeedsDeepSeekToolReasoning(provider, model, baseURL) ||
		openAICompatibleNeedsKimiToolReasoning(provider, baseURL)
}

func openAICompatibleNeedsDeepSeekToolReasoning(provider, model, baseURL string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	model = strings.ToLower(strings.TrimSpace(model))
	return provider == "deepseek" ||
		strings.Contains(model, "deepseek") ||
		baseURLHostMatches(baseURL, "api.deepseek.com")
}

func openAICompatibleNeedsKimiToolReasoning(provider, baseURL string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	return provider == "kimi-coding" ||
		provider == "kimi-coding-cn" ||
		baseURLHostMatches(baseURL, "api.kimi.com") ||
		baseURLHostMatches(baseURL, "moonshot.ai") ||
		baseURLHostMatches(baseURL, "moonshot.cn")
}

func baseURLHostMatches(rawBaseURL, domain string) bool {
	host := baseURLHostname(rawBaseURL)
	if host == "" {
		return false
	}
	domain = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(domain, ".")))
	if domain == "" {
		return false
	}
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func baseURLHostname(rawBaseURL string) string {
	rawBaseURL = strings.TrimSpace(rawBaseURL)
	if rawBaseURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawBaseURL)
	if err != nil || parsed.Host == "" {
		parsed, err = url.Parse("https://" + rawBaseURL)
		if err != nil {
			return ""
		}
	}
	return strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
}

func (c *httpClient) checkRateGuard() error {
	c.rateGuardMu.Lock()
	state := c.rateGuard
	c.rateGuardMu.Unlock()
	if state.LastKnownClass != RateLimitGenuineQuota {
		return nil
	}
	if time.Since(state.LastKnownAt) < 5*time.Minute {
		return fmt.Errorf("rate_guard_active: provider is rate-limited (class=%s since=%s)", state.LastKnownClass, state.LastKnownAt.Format(time.RFC3339))
	}
	return nil
}

func (c *httpClient) updateRateGuardFrom429(headers http.Header) {
	c.rateGuardMu.Lock()
	defer c.rateGuardMu.Unlock()
	c.rateGuard = ApplyClassification(c.rateGuard, time.Now(), Classify429(headers))
	classification := string(c.rateGuard.LastKnownClass)
	if c.onCredentialExhausted != nil && c.rateGuard.LastKnownClass != RateLimitInsufficientEvidence {
		c.onCredentialExhausted(http.StatusTooManyRequests, classification, headers)
	}
}

func (c *httpClient) updateRateLimitFromHeaders(headers http.Header) {
	state, ok := ParseRateLimitHeaders(headers, c.provider, time.Now())
	if !ok {
		return
	}
	c.rateLimitMu.Lock()
	c.rateLimit = state
	c.rateLimitMu.Unlock()
}

func (c *httpClient) handleUnauthorized(headers http.Header) {
	if c.onCredentialExhausted != nil {
		c.onCredentialExhausted(http.StatusUnauthorized, "auth_unauthorized", headers)
	}
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
		return nil, newHTTPError(resp.StatusCode, string(raw), resp.Header)
	}
	return newRunEventStream(resp.Body), nil
}
