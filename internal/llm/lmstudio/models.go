package lmstudio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

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

type StatusErrorFunc func(statusCode int, message string, header http.Header) error

// ListModels fetches the available local models from LM Studio.
func ListModels(ctx context.Context, baseURL string) ([]ModelInfo, error) {
	return ListModelsWithStatusError(ctx, baseURL, nil)
}

// ListModelsWithStatusError fetches models and delegates non-200 status errors to statusError when provided.
func ListModelsWithStatusError(ctx context.Context, baseURL string, statusError StatusErrorFunc) ([]ModelInfo, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("lmstudio list models: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lmstudio list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if statusError != nil {
			return nil, statusError(resp.StatusCode, "lmstudio list models failed", resp.Header)
		}
		return nil, fmt.Errorf("lmstudio list models failed: HTTP %d", resp.StatusCode)
	}

	var envelope ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("lmstudio list models decode: %w", err)
	}
	return envelope.Data, nil
}
