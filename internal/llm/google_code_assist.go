package llm

import (
	"fmt"
	"net/http"
	"strings"
)

type GoogleCodeAssistTokenProvider interface {
	Token() (string, error)
}

type GoogleCodeAssistResolver struct {
	project  string
	tier     string
	provider GoogleCodeAssistTokenProvider
}

func NewGoogleCodeAssistResolver(project, tier string, provider GoogleCodeAssistTokenProvider) *GoogleCodeAssistResolver {
	return &GoogleCodeAssistResolver{
		project:  strings.TrimSpace(project),
		tier:     strings.TrimSpace(tier),
		provider: provider,
	}
}

func (r *GoogleCodeAssistResolver) Headers() (http.Header, error) {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("User-Agent", "gormes-agent/0.0.0")
	if r.provider != nil {
		token, err := r.provider.Token()
		if err != nil {
			return nil, fmt.Errorf("code_assist_token_unavailable: %w", err)
		}
		if token != "" {
			header.Set("Authorization", "Bearer "+token)
		}
	}
	return header, nil
}

func (r *GoogleCodeAssistResolver) ProjectContext() string {
	if r.project != "" {
		return r.project
	}
	return "-"
}

func (r *GoogleCodeAssistResolver) Tier() string {
	if r.tier != "" {
		return r.tier
	}
	return "free"
}

func (r *GoogleCodeAssistResolver) RequiresExplicitProject() bool {
	return r.Tier() == "paid" && r.project == ""
}

func classifyGoogleCodeAssistError(status int, body string, header http.Header) ProviderErrorClassification {
	return classifyGeminiCloudCodeError(status, body, header)
}

func googleCodeAssistProviderStatus() ProviderStatus {
	status := openAICompatibleProviderStatus("google_code_assist", "")
	status.Provider = "google_code_assist"
	status.Runtime = "gemini_cloudcode"
	return status
}
