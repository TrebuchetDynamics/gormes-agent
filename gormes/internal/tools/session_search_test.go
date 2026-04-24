package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type recordingSessionSearchBackend struct {
	got  SessionSearchRequest
	hits []SessionSearchHit
	err  error
}

func (b *recordingSessionSearchBackend) Search(_ context.Context, req SessionSearchRequest) ([]SessionSearchHit, error) {
	b.got = req
	if b.err != nil {
		return nil, b.err
	}
	return append([]SessionSearchHit(nil), b.hits...), nil
}

func TestSessionSearchTool_RejectsMissingBackend(t *testing.T) {
	tool := &SessionSearchTool{}

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"atlas"}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want missing backend failure")
	}
	if !strings.Contains(err.Error(), "no session search backend configured") {
		t.Fatalf("error = %q, want missing backend message", err)
	}
}

func TestSessionSearchTool_EmptyQueryReturnsRecentMode(t *testing.T) {
	backend := &recordingSessionSearchBackend{
		hits: []SessionSearchHit{
			{SessionID: "sess-newer", Source: "telegram", ChatID: "42", LatestTurnUnix: 100},
			{SessionID: "sess-older", Source: "discord", ChatID: "chan-9", LatestTurnUnix: 50},
		},
	}
	tool := &SessionSearchTool{Backend: backend}

	out, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if backend.got.Query != "" {
		t.Fatalf("backend.Query = %q, want empty", backend.got.Query)
	}
	if backend.got.Limit != 3 {
		t.Fatalf("backend.Limit = %d, want default 3", backend.got.Limit)
	}

	var payload struct {
		Mode  string             `json:"mode"`
		Query string             `json:"query"`
		Hits  []SessionSearchHit `json:"hits"`
		Count int                `json:"count"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if payload.Mode != "recent" {
		t.Fatalf("mode = %q, want recent", payload.Mode)
	}
	if payload.Query != "" {
		t.Fatalf("query = %q, want empty", payload.Query)
	}
	if payload.Count != 2 {
		t.Fatalf("count = %d, want 2", payload.Count)
	}
	if payload.Hits[0].SessionID != "sess-newer" {
		t.Fatalf("hits[0] = %+v, want sess-newer first", payload.Hits[0])
	}
}

func TestSessionSearchTool_QueryReturnsSearchModeAndForwardsSources(t *testing.T) {
	backend := &recordingSessionSearchBackend{
		hits: []SessionSearchHit{
			{SessionID: "sess-discord", Source: "discord", ChatID: "chan-9", LatestTurnUnix: 75},
		},
	}
	tool := &SessionSearchTool{Backend: backend}

	out, err := tool.Execute(context.Background(), json.RawMessage(`{
		"query": "  Atlas  ",
		"sources": ["Discord", "", "discord"],
		"limit": 2
	}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if backend.got.Query != "Atlas" {
		t.Fatalf("backend.Query = %q, want trimmed Atlas", backend.got.Query)
	}
	if backend.got.Limit != 2 {
		t.Fatalf("backend.Limit = %d, want 2", backend.got.Limit)
	}
	if len(backend.got.Sources) != 1 || backend.got.Sources[0] != "discord" {
		t.Fatalf("backend.Sources = %v, want [discord]", backend.got.Sources)
	}

	var payload struct {
		Mode  string             `json:"mode"`
		Query string             `json:"query"`
		Hits  []SessionSearchHit `json:"hits"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if payload.Mode != "search" {
		t.Fatalf("mode = %q, want search", payload.Mode)
	}
	if payload.Query != "Atlas" {
		t.Fatalf("query = %q, want Atlas", payload.Query)
	}
	if len(payload.Hits) != 1 || payload.Hits[0].SessionID != "sess-discord" {
		t.Fatalf("hits = %+v, want sess-discord", payload.Hits)
	}
}

func TestSessionSearchTool_ClampsLimitAndExcludesCurrentSession(t *testing.T) {
	backend := &recordingSessionSearchBackend{
		hits: []SessionSearchHit{
			{SessionID: "sess-current", Source: "telegram", ChatID: "42", LatestTurnUnix: 200},
			{SessionID: "sess-other", Source: "discord", ChatID: "chan-9", LatestTurnUnix: 100},
		},
	}
	tool := &SessionSearchTool{Backend: backend}

	out, err := tool.Execute(context.Background(), json.RawMessage(`{
		"query": "atlas",
		"limit": 99,
		"current_session_id": "sess-current"
	}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if backend.got.Limit != 5 {
		t.Fatalf("backend.Limit = %d, want clamped 5", backend.got.Limit)
	}
	if backend.got.CurrentSessionID != "sess-current" {
		t.Fatalf("backend.CurrentSessionID = %q, want sess-current", backend.got.CurrentSessionID)
	}

	var payload struct {
		Hits []SessionSearchHit `json:"hits"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if len(payload.Hits) != 1 || payload.Hits[0].SessionID != "sess-other" {
		t.Fatalf("hits = %+v, want only sess-other (current excluded)", payload.Hits)
	}
}

func TestSessionSearchTool_ClampsLimitFloor(t *testing.T) {
	backend := &recordingSessionSearchBackend{}
	tool := &SessionSearchTool{Backend: backend}

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"x","limit":0}`)); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if backend.got.Limit != 1 {
		t.Fatalf("backend.Limit = %d, want clamped floor 1", backend.got.Limit)
	}
}

func TestSessionSearchTool_PropagatesBackendError(t *testing.T) {
	backend := &recordingSessionSearchBackend{err: errors.New("boom")}
	tool := &SessionSearchTool{Backend: backend}

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"atlas"}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want backend failure")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %q, want backend boom message", err)
	}
}

func TestSessionSearchTool_DescriptorMetadata(t *testing.T) {
	tool := &SessionSearchTool{}
	if tool.Name() != "session_search" {
		t.Fatalf("Name() = %q, want session_search", tool.Name())
	}
	if tool.Description() == "" {
		t.Fatal("Description() empty, want non-empty")
	}
	if !json.Valid(tool.Schema()) {
		t.Fatal("Schema() returned invalid JSON")
	}
	if tool.Timeout() <= 0 {
		t.Fatalf("Timeout() = %v, want positive duration", tool.Timeout())
	}
}
