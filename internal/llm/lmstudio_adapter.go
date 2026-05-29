package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultLMStudioBaseURL = "http://localhost:1234/v1"
	lmstudioProviderName   = "lmstudio"
)

// LMStudioAdapter routes chat requests to a local LM Studio inference server
// using the OpenAI-compatible chat completions protocol.
type LMStudioAdapter struct {
	baseURL string
}

// NewLMStudioAdapter creates an adapter for the LM Studio local server.
// If baseURL is empty, it defaults to http://localhost:1234/v1.
func NewLMStudioAdapter(baseURL string) *LMStudioAdapter {
	if baseURL == "" {
		baseURL = defaultLMStudioBaseURL
	}
	return &LMStudioAdapter{baseURL: baseURL}
}

// Client returns an HTTP client configured for LM Studio.
func (a *LMStudioAdapter) Client() Client {
	return NewHTTPClientWithProvider(a.baseURL, "", lmstudioProviderName)
}

// Status probes the LM Studio server and returns a typed status.
func (a *LMStudioAdapter) Status() ProviderStatus {
	if !a.reachable() {
		return ProviderStatus{
			Provider: lmstudioProviderName,
			Runtime:  "chat_completions",
			Capabilities: ProviderCapabilities{
				PromptCache:     unavailableCapability("lmstudio_unreachable"),
				ReasoningEcho:   unavailableCapability("lmstudio_unreachable"),
				RateGuard:       unavailableCapability("lmstudio_unreachable"),
				BudgetTelemetry: unavailableCapability("lmstudio_unreachable"),
			},
		}
	}
	return openAICompatibleProviderStatus(lmstudioProviderName, a.baseURL)
}

// reachable performs a lightweight health check against the LM Studio server.
func (a *LMStudioAdapter) reachable() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(a.baseURL + "/models")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ModelInfo describes one model exposed by the LM Studio server.
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse is the envelope returned by LM Studio's /v1/models endpoint.
type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// ListModels fetches the available local models from LM Studio.
func (a *LMStudioAdapter) ListModels(ctx context.Context) ([]ModelInfo, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("lmstudio list models: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lmstudio list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, newHTTPError(resp.StatusCode, "lmstudio list models failed", resp.Header)
	}

	var envelope ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("lmstudio list models decode: %w", err)
	}
	return envelope.Data, nil
}
