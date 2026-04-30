package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

func TestBuildDefaultRegistryDelegationDisabled(t *testing.T) {
	reg := buildDefaultRegistry(context.Background(), config.Config{}, nil, "")
	if _, ok := reg.Get("delegate_task"); ok {
		t.Fatal("delegate_task unexpectedly registered")
	}
	if _, ok := reg.Get("execute_code"); !ok {
		t.Fatal("execute_code not registered")
	}
	if _, ok := reg.Get("text_to_speech"); !ok {
		t.Fatal("text_to_speech not registered")
	}
	if _, ok := reg.Get("memory"); !ok {
		t.Fatal("memory not registered")
	}
	for _, name := range []string{"browser_navigate", "browser_snapshot", "browser_click", "browser_type", "browser_cdp", "browser_dialog", "web_search", "web_extract"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("%s not registered", name)
		}
	}
}

func TestBuildDefaultRegistryDelegationEnabled(t *testing.T) {
	reg := buildDefaultRegistry(context.Background(), config.Config{Delegation: config.DelegationCfg{
		Enabled:               true,
		MaxDepth:              2,
		MaxConcurrentChildren: 4,
		DefaultMaxIterations:  9,
		DefaultTimeout:        time.Minute,
	}}, nil, "")
	if _, ok := reg.Get("delegate_task"); !ok {
		t.Fatal("delegate_task not registered")
	}
}

func TestBuildDefaultRegistryDelegationToolExecutes(t *testing.T) {
	reg := buildDefaultRegistry(context.Background(), config.Config{Delegation: config.DelegationCfg{
		Enabled:               true,
		MaxDepth:              2,
		MaxConcurrentChildren: 3,
		DefaultMaxIterations:  50,
		DefaultTimeout:        time.Second,
	}}, nil, "")

	tool, ok := reg.Get("delegate_task")
	if !ok {
		t.Fatal("delegate_task not registered")
	}

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"goal":"audit runtime"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !bytes.Contains(out, []byte(`"status":"completed"`)) {
		t.Fatalf("output = %s, want completed status", out)
	}
}

func TestBuildDefaultRegistryDelegationToolAppendsRunLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.jsonl")

	reg := buildDefaultRegistry(context.Background(), config.Config{Delegation: config.DelegationCfg{
		Enabled:               true,
		MaxDepth:              2,
		MaxConcurrentChildren: 3,
		DefaultMaxIterations:  50,
		DefaultTimeout:        time.Second,
		RunLogPath:            path,
	}}, nil, "")

	tool, ok := reg.Get("delegate_task")
	if !ok {
		t.Fatal("delegate_task not registered")
	}

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"goal":"audit runtime"}`)); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if !bytes.Contains(raw, []byte(`"goal":"audit runtime"`)) {
		t.Fatalf("run log = %s, want goal field", raw)
	}
}

func TestBuildDefaultRegistryDelegationToolDraftsCandidate(t *testing.T) {
	root := t.TempDir()
	reg := buildDefaultRegistry(context.Background(), config.Config{
		Delegation: config.DelegationCfg{
			Enabled:               true,
			MaxDepth:              2,
			MaxConcurrentChildren: 3,
			DefaultMaxIterations:  50,
			DefaultTimeout:        time.Second,
		},
		Skills: config.SkillsCfg{Root: root},
	}, nil, "")

	tool, ok := reg.Get("delegate_task")
	if !ok {
		t.Fatal("delegate_task not registered")
	}

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"goal":"audit runtime","draft_candidate_slug":"audit-runtime","allow_no_tool_draft":true}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !bytes.Contains(out, []byte(`"candidate_id":"`)) {
		t.Fatalf("output = %s, want candidate_id", out)
	}

	entries, err := os.ReadDir(filepath.Join(root, "candidates"))
	if err != nil {
		t.Fatalf("ReadDir(candidates): %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("candidate dir count = %d, want 1", len(entries))
	}
	if _, err := os.Stat(filepath.Join(root, "candidates", entries[0].Name(), "SKILL.md")); err != nil {
		t.Fatalf("candidate SKILL.md missing: %v", err)
	}
}

func TestBuildDefaultRegistryPassesWebConfigBackend(t *testing.T) {
	var seenPath string
	var seenBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&seenBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"Tavily","url":"https://example.test","content":"ok"}]}`))
	}))
	defer srv.Close()

	t.Setenv("FIRECRAWL_API_KEY", "fire-secret")
	t.Setenv("TAVILY_API_KEY", "tavily-secret")
	t.Setenv("TAVILY_BASE_URL", srv.URL)

	reg := buildDefaultRegistry(context.Background(), config.Config{
		Web: config.WebCfg{Backend: "tavily"},
	}, nil, "")
	tool, ok := reg.Get("web_search")
	if !ok {
		t.Fatal("web_search not registered")
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"configured backend"}`)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if seenPath != "/search" {
		t.Fatalf("seen path = %q, want Tavily /search from config backend", seenPath)
	}
	if seenBody["api_key"] != "tavily-secret" || seenBody["query"] != "configured backend" {
		t.Fatalf("seen body = %#v, want Tavily body", seenBody)
	}
}

func TestBuildDefaultRegistryPassesWebsiteBlocklistToWebExtract(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		t.Errorf("provider should not be called for policy-blocked URL")
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	t.Setenv("FIRECRAWL_API_KEY", "fire-secret")
	t.Setenv("FIRECRAWL_API_URL", srv.URL)

	reg := buildDefaultRegistry(context.Background(), config.Config{
		Security: config.SecurityCfg{WebsiteBlocklist: config.WebsiteBlocklistCfg{
			Enabled: true,
			Domains: []string{"blocked.test"},
		}},
	}, nil, "")
	tool, ok := reg.Get("web_extract")
	if !ok {
		t.Fatal("web_extract not registered")
	}
	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://blocked.test/page"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if called {
		t.Fatal("provider was called for policy-blocked URL")
	}
	if !strings.Contains(string(out), "Blocked by website policy") || !strings.Contains(string(out), `"rule":"blocked.test"`) {
		t.Fatalf("output = %s, want blocked policy result", out)
	}
}

func TestBuildDefaultRegistryWiresWebContentProcessor(t *testing.T) {
	longContent := strings.Repeat("long content ", 600)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"markdown": longContent,
				"metadata": map[string]any{
					"title":     "Long Page",
					"sourceURL": "https://example.test/long",
				},
			},
		})
	}))
	defer srv.Close()

	t.Setenv("FIRECRAWL_API_KEY", "fire-secret")
	t.Setenv("FIRECRAWL_API_URL", srv.URL)

	mock := hermes.NewMockClient()
	mock.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "registry summary"},
		{Kind: hermes.EventDone},
	}, "summary-session")

	reg := buildDefaultRegistry(context.Background(), config.Config{}, mock, "summary-model")
	tool, ok := reg.Get("web_extract")
	if !ok {
		t.Fatal("web_extract not registered")
	}
	out, err := tool.Execute(context.Background(), json.RawMessage(`{"urls":["https://example.test/long"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(string(out), "registry summary") {
		t.Fatalf("output = %s, want processed summary", out)
	}
	requests := mock.Requests()
	if len(requests) != 1 {
		t.Fatalf("summary requests = %d, want one", len(requests))
	}
	if requests[0].Model != "summary-model" || !strings.Contains(requests[0].Messages[len(requests[0].Messages)-1].Content, "Long Page") {
		t.Fatalf("summary request = %+v, want model and page context", requests[0])
	}
}
