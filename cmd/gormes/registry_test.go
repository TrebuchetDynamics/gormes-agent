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
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestBuildDefaultRegistryIncludesSessionSearch(t *testing.T) {
	reg := buildDefaultRegistry(context.Background(), config.Config{}, nil, "")
	if _, ok := reg.Get("session_search"); !ok {
		t.Fatal("session_search not registered in default tool registry")
	}
}

func TestBuildDefaultRegistryDelegationDisabled(t *testing.T) {
	reg := buildDefaultRegistry(context.Background(), config.Config{}, nil, "")
	if _, ok := reg.Get("delegate_task"); ok {
		t.Fatal("delegate_task unexpectedly registered")
	}
	if _, ok := reg.Get("execute_code"); !ok {
		t.Fatal("execute_code not registered")
	}
	_, hasTTS := reg.Get("text_to_speech")
	if audioToolsEnabled() && !hasTTS {
		t.Fatal("text_to_speech not registered")
	}
	if !audioToolsEnabled() && hasTTS {
		t.Fatal("text_to_speech registered in gormes_lite build")
	}
	if _, ok := reg.Get("memory"); !ok {
		t.Fatal("memory not registered")
	}
	if _, ok := reg.Get("clarify"); !ok {
		t.Fatal("clarify not registered")
	}
	if _, ok := reg.Get("image_generate"); !ok {
		t.Fatal("image_generate not registered")
	}
	if _, ok := reg.Get("video_analyze"); !ok {
		t.Fatal("video_analyze not registered")
	}
	if _, ok := reg.Get("vision_analyze"); !ok {
		t.Fatal("vision_analyze not registered")
	}
	for _, name := range []string{"read_file", "search_files", "write_file", "patch", "terminal"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("%s not registered", name)
		}
	}
	for _, name := range []string{"skills_list", "skill_view"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("%s not registered", name)
		}
	}
	for _, name := range []string{"browser_navigate", "browser_snapshot", "browser_click", "browser_type", "browser_cdp", "browser_dialog", "web_search", "web_extract", "web_crawl"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("%s not registered", name)
		}
	}
}

func TestBuildDefaultRegistryHomeAssistantRequiresToken(t *testing.T) {
	t.Setenv("HASS_TOKEN", "")
	reg := buildDefaultRegistry(context.Background(), config.Config{}, nil, "")
	for _, name := range []string{"ha_list_entities", "ha_get_state", "ha_list_services", "ha_call_service"} {
		if _, ok := reg.Get(name); ok {
			t.Fatalf("%s registered without HASS_TOKEN", name)
		}
	}

	t.Setenv("HASS_TOKEN", "test-token")
	t.Setenv("HASS_URL", "http://homeassistant.local:8123")
	reg = buildDefaultRegistry(context.Background(), config.Config{}, nil, "")
	for _, name := range []string{"ha_list_entities", "ha_get_state", "ha_list_services", "ha_call_service"} {
		tool, ok := reg.Get(name)
		if !ok {
			t.Fatalf("%s not registered with HASS_TOKEN", name)
		}
		if strings.Contains(string(tool.Schema()), "test-token") {
			t.Fatalf("%s schema leaked HASS_TOKEN: %s", name, tool.Schema())
		}
	}
}

func TestDefaultRegistryIncludesVisionAnalyze(t *testing.T) {
	reg := buildDefaultRegistry(context.Background(), config.Config{}, nil, "")
	if _, ok := reg.Get("vision_analyze"); !ok {
		t.Fatal("vision_analyze not registered")
	}
}

func TestBuildDefaultRegistryPassesTerminalCWDToTerminalTool(t *testing.T) {
	projectDir := t.TempDir()
	otherDir := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(otherDir); err != nil {
		t.Fatalf("Chdir(%q): %v", otherDir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	reg := buildDefaultRegistry(context.Background(), config.Config{
		Terminal: config.TerminalCfg{CWD: projectDir},
	}, nil, "")
	tool, ok := reg.Get("terminal")
	if !ok {
		t.Fatal("terminal not registered")
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"command":"pwd"}`))
	if err != nil {
		t.Fatalf("terminal Execute: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal terminal output %s: %v", raw, err)
	}
	if out["workdir"] != projectDir {
		t.Fatalf("terminal workdir = %v, want %q; output=%s", out["workdir"], projectDir, raw)
	}
	if strings.TrimSpace(asRegistryString(out["output"])) != projectDir {
		t.Fatalf("terminal output = %q, want pwd %q; raw=%s", out["output"], projectDir, raw)
	}
}

func TestBuildDefaultRegistryPassesExecuteCodeMode(t *testing.T) {
	reg := buildDefaultRegistry(context.Background(), config.Config{
		CodeExecution: config.CodeExecutionCfg{Mode: "project"},
	}, nil, "")
	tool, ok := reg.Get("execute_code")
	if !ok {
		t.Fatal("execute_code not registered")
	}
	execTool, ok := tool.(*tools.ExecuteCodeTool)
	if !ok {
		t.Fatalf("execute_code tool = %T, want *tools.ExecuteCodeTool", tool)
	}
	if execTool.Mode != tools.ExecuteCodeModeProject {
		t.Fatalf("execute_code mode = %q, want %q", execTool.Mode, tools.ExecuteCodeModeProject)
	}

	reg = buildDefaultRegistry(context.Background(), config.Config{
		CodeExecution: config.CodeExecutionCfg{Mode: "not-a-mode"},
	}, nil, "")
	tool, ok = reg.Get("execute_code")
	if !ok {
		t.Fatal("execute_code not registered after invalid mode")
	}
	execTool, ok = tool.(*tools.ExecuteCodeTool)
	if !ok {
		t.Fatalf("execute_code tool = %T, want *tools.ExecuteCodeTool", tool)
	}
	if execTool.Mode != tools.ExecuteCodeModeStrict {
		t.Fatalf("invalid execute_code mode fell back to %q, want %q", execTool.Mode, tools.ExecuteCodeModeStrict)
	}
	if !executeCodeRegistryHasEvidence(execTool.ModeEvidence, tools.ExecuteCodeModeEvidenceInvalid) {
		t.Fatalf("ModeEvidence = %#v, want invalid mode evidence", execTool.ModeEvidence)
	}
}

func executeCodeRegistryHasEvidence(evidence []tools.ExecuteCodeModeEvidence, code string) bool {
	for _, ev := range evidence {
		if ev.Code == code {
			return true
		}
	}
	return false
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

func asRegistryString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
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

func TestBuildDefaultRegistryUsesNousAuthStoreForManagedWebGateway(t *testing.T) {
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)
	if err := os.WriteFile(filepath.Join(gormesHome, "auth.json"), []byte(`{
  "providers": {
    "nous": {
      "access_token": "nous-auth-store-token",
      "expires_at": "2999-01-01T00:00:00Z"
    }
  }
}`), 0o600); err != nil {
		t.Fatalf("write auth store: %v", err)
	}

	var seenAuthorization string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuthorization = r.Header.Get("Authorization")
		if r.URL.Path != "/v2/search" {
			t.Errorf("path = %q, want /v2/search", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":[{"title":"Managed","url":"https://example.test","description":"ok"}]}`))
	}))
	defer srv.Close()

	t.Setenv("FIRECRAWL_API_KEY", "direct-firecrawl-key")
	t.Setenv("FIRECRAWL_GATEWAY_URL", srv.URL)

	reg := buildDefaultRegistry(context.Background(), config.Config{
		Web: config.WebCfg{Backend: "firecrawl", UseGateway: true},
	}, nil, "")
	tool, ok := reg.Get("web_search")
	if !ok {
		t.Fatal("web_search not registered")
	}
	out, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"managed gateway"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if seenAuthorization != "Bearer nous-auth-store-token" {
		t.Fatalf("Authorization = %q, want auth-store token", seenAuthorization)
	}
	if strings.Contains(string(out), "nous-auth-store-token") || strings.Contains(string(out), "direct-firecrawl-key") {
		t.Fatalf("output leaked token: %s", out)
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

func TestBuildDefaultRegistryPassesWebsiteBlocklistToWebCrawl(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		t.Errorf("provider should not be called for policy-blocked crawl URL")
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
	tool, ok := reg.Get("web_crawl")
	if !ok {
		t.Fatal("web_crawl not registered")
	}
	out, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://blocked.test"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if called {
		t.Fatal("provider was called for policy-blocked crawl URL")
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

func TestBuildDefaultRegistryWiresWebContentProcessorToWebCrawl(t *testing.T) {
	longContent := strings.Repeat("long content ", 600)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/crawl" {
			t.Errorf("path = %q, want /v2/crawl", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": []map[string]any{{
				"markdown": longContent,
				"metadata": map[string]any{
					"title":     "Long Crawl Page",
					"sourceURL": "https://example.test/long",
				},
			}},
		})
	}))
	defer srv.Close()

	t.Setenv("FIRECRAWL_API_KEY", "fire-secret")
	t.Setenv("FIRECRAWL_API_URL", srv.URL)

	mock := hermes.NewMockClient()
	mock.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "crawl registry summary"},
		{Kind: hermes.EventDone},
	}, "summary-session")

	reg := buildDefaultRegistry(context.Background(), config.Config{}, mock, "summary-model")
	tool, ok := reg.Get("web_crawl")
	if !ok {
		t.Fatal("web_crawl not registered")
	}
	out, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.test"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(string(out), "crawl registry summary") {
		t.Fatalf("output = %s, want processed crawl summary", out)
	}
	requests := mock.Requests()
	if len(requests) != 1 {
		t.Fatalf("summary requests = %d, want one", len(requests))
	}
	if requests[0].Model != "summary-model" || !strings.Contains(requests[0].Messages[len(requests[0].Messages)-1].Content, "Long Crawl Page") {
		t.Fatalf("summary request = %+v, want model and crawl page context", requests[0])
	}
}
