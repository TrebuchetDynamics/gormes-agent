package hermes

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

const codexCloudflareUserAgent = "codex_cli_rs/0.0.0 (Hermes Agent)"

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
	parsed, err := url.Parse(strings.TrimSpace(rawBaseURL))
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	path := "/" + strings.Trim(strings.ToLower(parsed.EscapedPath()), "/")
	return host == "chatgpt.com" && (path == "/backend-api/codex" || strings.HasPrefix(path, "/backend-api/codex/"))
}

func codexChatGPTAccountID(accessToken string) string {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return ""
	}
	parts := strings.Split(accessToken, ".")
	if len(parts) < 2 {
		return ""
	}
	payload := parts[1]
	if payload == "" {
		return ""
	}
	if rem := len(payload) % 4; rem != 0 {
		payload += strings.Repeat("=", 4-rem)
	}
	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return ""
	}
	var claims map[string]any
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return ""
	}
	auth, ok := claims["https://api.openai.com/auth"].(map[string]any)
	if !ok {
		return ""
	}
	accountID, _ := auth["chatgpt_account_id"].(string)
	return strings.TrimSpace(accountID)
}
