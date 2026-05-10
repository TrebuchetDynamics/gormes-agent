package hermes

import "testing"

// Live regression 2026-05-10: an operator wired up OpenRouter using the
// documented base URL `https://openrouter.ai/api/v1` and got
// "Not Found: provider returned HTML error body" with no further
// indication that the joined request URL had become
// `https://openrouter.ai/api/v1/v1/chat/completions` (double `/v1`,
// 404). Operators copy-pasting the documented OpenAI-compatible base
// URL is the natural intuition; the URL builder must collapse the
// duplicated `/v1` so both `endpoint = '.../api'` and
// `endpoint = '.../api/v1'` resolve to the same request URL.

func TestOpenAICompatibleURL_CollapsesDoubleV1Prefix(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		path    string
		want    string
	}{
		{
			name:    "openrouter base with /v1, chat completions path with /v1 — collapse",
			baseURL: "https://openrouter.ai/api/v1",
			path:    "/v1/chat/completions",
			want:    "https://openrouter.ai/api/v1/chat/completions",
		},
		{
			name:    "openrouter base with /v1/ trailing slash — collapse",
			baseURL: "https://openrouter.ai/api/v1/",
			path:    "/v1/chat/completions",
			want:    "https://openrouter.ai/api/v1/chat/completions",
		},
		{
			name:    "openrouter base WITHOUT /v1 — no collapse, still correct",
			baseURL: "https://openrouter.ai/api",
			path:    "/v1/chat/completions",
			want:    "https://openrouter.ai/api/v1/chat/completions",
		},
		{
			name:    "openai base with /v1 — collapse (same defect class)",
			baseURL: "https://api.openai.com/v1",
			path:    "/v1/chat/completions",
			want:    "https://api.openai.com/v1/chat/completions",
		},
		{
			name:    "anthropic base without /v1, anthropic path /v1/messages — no collapse, no double prefix",
			baseURL: "https://api.anthropic.com",
			path:    "/v1/messages",
			want:    "https://api.anthropic.com/v1/messages",
		},
		{
			name:    "azure-style base with /v1 ending and non-v1 path (e.g. /responses) — leave basePath alone",
			baseURL: "https://example.com/api/v1",
			path:    "/responses",
			want:    "https://example.com/api/v1/responses",
		},
		{
			name:    "base path with /v1 in the middle (not trailing) — no collapse",
			baseURL: "https://example.com/v1/proxy",
			path:    "/v1/chat/completions",
			want:    "https://example.com/v1/proxy/v1/chat/completions",
		},
		{
			name:    "no scheme baseURL falls through to raw concat (preserves prior behavior)",
			baseURL: "not-a-url",
			path:    "/v1/chat/completions",
			want:    "not-a-url/v1/chat/completions",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &httpClient{baseURL: tc.baseURL}
			got := c.openAICompatibleURL(tc.path)
			if got != tc.want {
				t.Fatalf("openAICompatibleURL(%q) with baseURL=%q = %q, want %q", tc.path, tc.baseURL, got, tc.want)
			}
		})
	}
}
