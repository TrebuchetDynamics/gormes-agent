package codex

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
)

const CloudflareUserAgent = "codex_cli_rs/0.0.0 (Hermes Agent)"

func ChatGPTBackendBaseURL(rawBaseURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawBaseURL))
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	path := "/" + strings.Trim(strings.ToLower(parsed.EscapedPath()), "/")
	return host == "chatgpt.com" && (path == "/backend-api/codex" || strings.HasPrefix(path, "/backend-api/codex/"))
}

func ChatGPTAccountID(accessToken string) string {
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
