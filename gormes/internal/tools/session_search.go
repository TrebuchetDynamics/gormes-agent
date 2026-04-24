package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// SessionSearchBackend is the host-injected seam that runs the actual search
// across the canonical session catalog. Production wires it from
// internal/memory.SearchSessions over the user-bound metadata set; tests inject
// a fake to exercise the tool's argument plumbing without SQLite.
type SessionSearchBackend interface {
	Search(ctx context.Context, req SessionSearchRequest) ([]SessionSearchHit, error)
}

// SessionSearchRequest is the validated payload forwarded to the backend.
type SessionSearchRequest struct {
	Query            string
	Sources          []string
	Limit            int
	CurrentSessionID string
}

// SessionSearchHit is one session-level result row. Fields mirror the columns
// surfaced by internal/memory.SessionSearchHit so callers can convert without
// importing the memory package.
type SessionSearchHit struct {
	SessionID      string `json:"session_id"`
	Source         string `json:"source,omitempty"`
	ChatID         string `json:"chat_id,omitempty"`
	LatestTurnUnix int64  `json:"latest_turn_unix"`
}

// SessionSearchTool ports the Python session_search operator tool to the
// Go-native registry. It stays unavailable until Backend is injected.
type SessionSearchTool struct {
	Backend SessionSearchBackend
}

var _ Tool = (*SessionSearchTool)(nil)

const (
	sessionSearchDefaultLimit = 3
	sessionSearchMaxLimit     = 5
	sessionSearchMinLimit     = 1
)

func (*SessionSearchTool) Name() string { return "session_search" }

func (*SessionSearchTool) Description() string {
	return "Search past sessions for a topic, or list recent sessions when query is empty. Returns deterministic session-level hits ordered by the canonical catalog (latest turn first); excludes the current session from results."
}

func (*SessionSearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"query":{"type":"string","description":"Search keywords or phrase. Omit to list recent sessions."},
			"sources":{
				"type":"array",
				"description":"Optional allowlist of transport sources (e.g. telegram, discord). Case-insensitive.",
				"items":{"type":"string"}
			},
			"limit":{
				"type":"integer",
				"description":"Maximum number of session hits to return. Clamped to [1, 5]. Default 3.",
				"default":3
			},
			"current_session_id":{
				"type":"string",
				"description":"Active session id to exclude from results so the caller does not see itself."
			}
		},
		"required":[]
	}`)
}

func (*SessionSearchTool) Timeout() time.Duration { return 30 * time.Second }

func (t *SessionSearchTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	if t.Backend == nil {
		return nil, errors.New("session_search: no session search backend configured")
	}

	var in struct {
		Query            string   `json:"query"`
		Sources          []string `json:"sources"`
		Limit            *int     `json:"limit"`
		CurrentSessionID string   `json:"current_session_id"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("session_search: invalid args: %w", err)
	}

	req := SessionSearchRequest{
		Query:            strings.TrimSpace(in.Query),
		Sources:          normalizeSearchSources(in.Sources),
		Limit:            clampSessionSearchLimit(in.Limit),
		CurrentSessionID: strings.TrimSpace(in.CurrentSessionID),
	}

	hits, err := t.Backend.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("session_search: %w", err)
	}

	filtered := excludeCurrentSession(hits, req.CurrentSessionID)
	if filtered == nil {
		filtered = []SessionSearchHit{}
	}
	out := struct {
		Mode  string             `json:"mode"`
		Query string             `json:"query,omitempty"`
		Hits  []SessionSearchHit `json:"hits"`
		Count int                `json:"count"`
	}{
		Mode:  sessionSearchMode(req.Query),
		Query: req.Query,
		Hits:  filtered,
		Count: len(filtered),
	}
	return json.Marshal(out)
}

func sessionSearchMode(query string) string {
	if query == "" {
		return "recent"
	}
	return "search"
}

func clampSessionSearchLimit(raw *int) int {
	if raw == nil {
		return sessionSearchDefaultLimit
	}
	value := *raw
	if value < sessionSearchMinLimit {
		return sessionSearchMinLimit
	}
	if value > sessionSearchMaxLimit {
		return sessionSearchMaxLimit
	}
	return value
}

func normalizeSearchSources(sources []string) []string {
	if len(sources) == 0 {
		return nil
	}
	out := make([]string, 0, len(sources))
	for _, raw := range sources {
		src := strings.ToLower(strings.TrimSpace(raw))
		if src == "" || slices.Contains(out, src) {
			continue
		}
		out = append(out, src)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func excludeCurrentSession(hits []SessionSearchHit, currentSessionID string) []SessionSearchHit {
	if currentSessionID == "" {
		return hits
	}
	out := hits[:0:0]
	for _, hit := range hits {
		if hit.SessionID == currentSessionID {
			continue
		}
		out = append(out, hit)
	}
	return out
}
