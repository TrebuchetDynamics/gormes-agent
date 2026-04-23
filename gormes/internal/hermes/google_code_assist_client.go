package hermes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultGoogleCodeAssistBaseURL = "https://cloudcode-pa.googleapis.com"
	googleCodeAssistMarkerEndpoint = "cloudcode-pa://google"
	googleCodeAssistStreamPath     = "/v1internal:streamGenerateContent?alt=sse"
	googleCodeAssistLoadPath       = "/v1internal:loadCodeAssist"
)

type googleCodeAssistClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

type googleCodeAssistStreamRequest struct {
	Project      string        `json:"project,omitempty"`
	Model        string        `json:"model"`
	UserPromptID string        `json:"user_prompt_id"`
	Request      geminiRequest `json:"request"`
}

type googleCodeAssistHealthRequest struct {
	Metadata struct {
		DuetProject string `json:"duetProject,omitempty"`
		IDEType     string `json:"ideType"`
		Platform    string `json:"platform"`
		PluginType  string `json:"pluginType"`
	} `json:"metadata"`
	CloudAICompanionProject string `json:"cloudaicompanionProject,omitempty"`
}

func newGoogleCodeAssistClient(baseURL, apiKey string) Client {
	return &googleCodeAssistClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    newStreamingHTTPClient(),
	}
}

func (c *googleCodeAssistClient) OpenStream(ctx context.Context, req ChatRequest) (Stream, error) {
	body, err := buildGoogleCodeAssistRequest(req)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+googleCodeAssistStreamPath, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
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
	return newWrappedGeminiStream(resp.Body), nil
}

func (c *googleCodeAssistClient) OpenRunEvents(context.Context, string) (RunEventStream, error) {
	return nil, ErrRunEventsNotSupported
}

func (c *googleCodeAssistClient) Health(ctx context.Context) error {
	body := googleCodeAssistHealthRequest{}
	body.Metadata.IDEType = "IDE_UNSPECIFIED"
	body.Metadata.Platform = "PLATFORM_UNSPECIFIED"
	body.Metadata.PluginType = "GEMINI"
	if projectID := googleCodeAssistProjectID(); projectID != "" {
		body.Metadata.DuetProject = projectID
		body.CloudAICompanionProject = projectID
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+googleCodeAssistLoadPath, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
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

func buildGoogleCodeAssistRequest(req ChatRequest) (googleCodeAssistStreamRequest, error) {
	inner, err := buildGeminiRequest(req)
	if err != nil {
		return googleCodeAssistStreamRequest{}, err
	}
	return googleCodeAssistStreamRequest{
		Project:      googleCodeAssistProjectID(),
		Model:        req.Model,
		UserPromptID: fmt.Sprintf("%d", time.Now().UnixNano()),
		Request:      inner,
	}, nil
}

func googleCodeAssistProjectID() string {
	for _, key := range []string{
		"GORMES_GEMINI_PROJECT_ID",
		"HERMES_GEMINI_PROJECT_ID",
		"GOOGLE_CLOUD_PROJECT",
		"GOOGLE_CLOUD_PROJECT_ID",
	} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}
