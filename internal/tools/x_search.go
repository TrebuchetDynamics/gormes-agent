package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

const XSearchToolName = "x_search"

type XSearchConfig struct {
	AuthMode    string
	APIKey      string
	OAuthToken  string
	OAuthExpiry time.Time
	Fake        bool
	RateLimit   bool
}

type XSearchAuthStatus struct {
	Configured  bool
	AuthMode    string
	RedactedKey string
	Expired     bool
}

func (c XSearchConfig) AuthStatus() XSearchAuthStatus {
	switch c.AuthMode {
	case "api_key":
		if c.APIKey == "" {
			return XSearchAuthStatus{AuthMode: "api_key"}
		}
		return XSearchAuthStatus{
			Configured:  true,
			AuthMode:    "api_key",
			RedactedKey: redactKey(c.APIKey),
		}
	case "oauth":
		if c.OAuthToken == "" {
			return XSearchAuthStatus{AuthMode: "oauth"}
		}
		if c.OAuthExpiry.Before(time.Now()) {
			return XSearchAuthStatus{
				AuthMode: "oauth",
				Expired:  true,
			}
		}
		return XSearchAuthStatus{
			Configured: true,
			AuthMode:   "oauth",
		}
	default:
		return XSearchAuthStatus{}
	}
}

func redactKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

type XSearchTool struct {
	cfg XSearchConfig
}

func (t *XSearchTool) Name() string { return XSearchToolName }
func (t *XSearchTool) Description() string {
	return "Search X (Twitter) for recent posts, users, and trends. Returns bounded result envelopes with post text, author info, and engagement metrics."
}
func (t *XSearchTool) Timeout() time.Duration { return 30 * time.Second }

func (t *XSearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Search query string. Supports X search operators like from:, to:, since:, until:, min_faves:, etc."
			},
			"count": {
				"type": "integer",
				"description": "Maximum number of results to return. Default 10, max 50.",
				"default": 10
			},
			"result_type": {
				"type": "string",
				"enum": ["recent", "popular", "mixed"],
				"description": "Type of results to return.",
				"default": "recent"
			}
		},
		"required": ["query"]
	}`)
}

func (t *XSearchTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	status := t.cfg.AuthStatus()
	if !status.Configured {
		return nil, fmt.Errorf("x_search: credentials not configured (auth_mode=%q). Set X_API_KEY or OAuth token.", t.cfg.AuthMode)
	}
	if status.Expired {
		return nil, fmt.Errorf("x_search: OAuth token expired. Re-authenticate to continue.")
	}
	if t.cfg.RateLimit {
		return nil, fmt.Errorf("x_search: rate limit exceeded. Retry after cooldown period.")
	}

	var req struct {
		Query      string `json:"query"`
		Count      int    `json:"count"`
		ResultType string `json:"result_type"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("x_search: invalid args: %w", err)
	}
	if req.Query == "" {
		return nil, fmt.Errorf("x_search: query is required")
	}
	if req.Count <= 0 {
		req.Count = 10
	}
	if req.Count > 50 {
		req.Count = 50
	}

	if t.cfg.Fake {
		return t.fakeResults(req)
	}

	return nil, fmt.Errorf("x_search: live search not yet implemented (use fake mode for testing)")
}

func (t *XSearchTool) fakeResults(req struct {
	Query      string `json:"query"`
	Count      int    `json:"count"`
	ResultType string `json:"result_type"`
}) (json.RawMessage, error) {
	resp := XSearchResponse{
		Query:      req.Query,
		ResultType: req.ResultType,
		Count:      req.Count,
		Results: []XSearchResult{
			{
				ID:        "1",
				Text:      "Sample post matching query: " + req.Query,
				Author:    "@example_user",
				CreatedAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				Metrics:   XSearchMetrics{Likes: 42, Retweets: 12, Replies: 3},
			},
			{
				ID:        "2",
				Text:      "Another relevant result for: " + req.Query,
				Author:    "@dev_account",
				CreatedAt: time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
				Metrics:   XSearchMetrics{Likes: 128, Retweets: 34, Replies: 8},
			},
		},
	}
	if len(resp.Results) > req.Count {
		resp.Results = resp.Results[:req.Count]
	}
	return json.Marshal(resp)
}

type XSearchResponse struct {
	Query      string          `json:"query"`
	ResultType string          `json:"result_type"`
	Count      int             `json:"count"`
	Results    []XSearchResult `json:"results"`
}

type XSearchResult struct {
	ID        string         `json:"id"`
	Text      string         `json:"text"`
	Author    string         `json:"author"`
	CreatedAt string         `json:"created_at"`
	Metrics   XSearchMetrics `json:"metrics"`
}

type XSearchMetrics struct {
	Likes    int `json:"likes"`
	Retweets int `json:"retweets"`
	Replies  int `json:"replies"`
}

func NewXSearchTools(cfg XSearchConfig) []toolkit.Tool {
	return []toolkit.Tool{&XSearchTool{cfg: cfg}}
}

func RegisterXSearchTools(r *toolkit.Registry, cfg XSearchConfig) {
	if r == nil {
		return
	}
	status := cfg.AuthStatus()
	if !status.Configured && !cfg.Fake {
		return
	}
	for _, tool := range NewXSearchTools(cfg) {
		r.MustRegister(tool)
	}
}
