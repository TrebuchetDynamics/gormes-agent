package llm

import (
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/codex"
)

const codexCloudflareUserAgent = codex.CloudflareUserAgent

func (c *httpClient) applyCodexCloudflareHeaders(req *http.Request) {
	if req == nil || !c.usesCodexResponsesTransport() || !codexChatGPTBackendBaseURL(c.baseURL) {
		return
	}
	req.Header["User-Agent"] = []string{codexCloudflareUserAgent}
	req.Header["originator"] = []string{"codex_cli_rs"}
	if accountID := codexChatGPTAccountID(c.apiKey); accountID != "" {
		req.Header["ChatGPT-Account-ID"] = []string{accountID}
	}
}

func codexChatGPTBackendBaseURL(rawBaseURL string) bool {
	return codex.ChatGPTBackendBaseURL(rawBaseURL)
}

func codexChatGPTAccountID(accessToken string) string {
	return codex.ChatGPTAccountID(accessToken)
}
