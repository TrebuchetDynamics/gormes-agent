package llm

import (
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/googlecodeassist"
)

type GoogleCodeAssistTokenProvider = googlecodeassist.TokenProvider

type GoogleCodeAssistResolver = googlecodeassist.Resolver

func NewGoogleCodeAssistResolver(project, tier string, provider GoogleCodeAssistTokenProvider) *GoogleCodeAssistResolver {
	return googlecodeassist.NewResolver(project, tier, provider)
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
