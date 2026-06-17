package apiserver

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
)

type fakeCronReader struct{ jobs []cron.Job }

func (f fakeCronReader) List() ([]cron.Job, error)       { return f.jobs, nil }
func (f fakeCronReader) Get(id string) (cron.Job, error) { return cron.Job{}, nil }

func newUITestServer() *Server {
	return NewServer(Config{
		ModelName:          "gormes-agent",
		ProviderName:       "native",
		DashboardBoundHost: "127.0.0.1",
		ConfigSummary: func() []DashboardKeyValue {
			return []DashboardKeyValue{{Key: "workspace", Value: "/home/op/work"}, {Key: "model", Value: "gormes-agent"}}
		},
		EnvStatus: func() []DashboardEnvKey {
			return []DashboardEnvKey{{Name: "ANTHROPIC_API_KEY", Set: true, Source: "env"}, {Name: "OPENAI_API_KEY", Set: false, Source: "—"}}
		},
		SkillsList: func() []DashboardSkill {
			return []DashboardSkill{{Name: "gormes-builder", Source: "bundled", Enabled: true}}
		},
		CronJobs: fakeCronReader{jobs: []cron.Job{{Name: "daily-report", Schedule: "0 9 * * *", LastStatus: "success"}}},
	})
}

func getUI(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Host = "127.0.0.1"
	h.ServeHTTP(rec, req)
	return rec
}

func TestDashboardPagesRenderWithNav(t *testing.T) {
	h := newUITestServer().Handler()
	pages := []struct {
		path  string
		title string
	}{
		{"/", "Dashboard — Gormes"},
		{"/dashboard", "Dashboard — Gormes"},
		{"/chat", "Chat — Gormes"},
		{"/sessions", "Sessions — Gormes"},
		{"/skills", "Skills — Gormes"},
		{"/config", "Config — Gormes"},
		{"/cron", "Cron — Gormes"},
		{"/env", "Env — Gormes"},
		{"/models", "Models — Gormes"},
		{"/system", "System — Gormes"},
		{"/logs", "Logs — Gormes"},
	}
	for _, p := range pages {
		rec := getUI(t, h, p.path)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", p.path, rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "<title>"+p.title+"</title>") {
			t.Fatalf("GET %s missing title %q", p.path, p.title)
		}
		if !strings.Contains(body, `href="/sessions"`) {
			t.Fatalf("GET %s missing shared nav", p.path)
		}
	}
}

func TestDashboardLoadsAndServesSSEExtension(t *testing.T) {
	h := newUITestServer().Handler()
	// Pages must load both htmx core and the sse extension, or sse-swap is inert.
	page := getUI(t, h, "/chat")
	body := page.Body.String()
	if !strings.Contains(body, "/static/htmx.min.js") || !strings.Contains(body, "/static/sse.js") {
		t.Fatalf("chat page missing htmx/sse script tags; body head:\n%s", body[:min(len(body), 600)])
	}
	// The extension itself must be served from the embedded static FS.
	rec := getUI(t, h, "/static/sse.js")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /static/sse.js = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "defineExtension('sse'") {
		t.Fatalf("/static/sse.js did not serve the htmx sse extension")
	}
}

func TestDashboardUnknownPathIs404(t *testing.T) {
	h := newUITestServer().Handler()
	rec := getUI(t, h, "/definitely-not-a-page")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET unknown path = %d, want 404", rec.Code)
	}
}

func TestDashboardFragmentsRenderHTML(t *testing.T) {
	h := newUITestServer().Handler()
	cases := []struct {
		path string
		want string
	}{
		{"/ui/config", "workspace"},
		{"/ui/skills", "gormes-builder"},
		{"/ui/cron", "daily-report"},
		{"/ui/env", "ANTHROPIC_API_KEY"},
		{"/ui/models", "native"},
		{"/ui/system", "Goroutines"},
		{"/ui/logs", "No log entries yet."},
		{"/ui/sessions", "No sessions yet."},
		{"/dashboard/status", "Model"},
	}
	for _, c := range cases {
		rec := getUI(t, h, c.path)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200; body=%s", c.path, rec.Code, rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Fatalf("GET %s content-type = %q, want text/html", c.path, ct)
		}
		if !strings.Contains(rec.Body.String(), c.want) {
			t.Fatalf("GET %s missing %q; body=%s", c.path, c.want, rec.Body.String())
		}
	}
}

func TestUnwiredFragmentsDegradeGracefully(t *testing.T) {
	// No ConfigSummary/SkillsList/EnvStatus/CronJobs wired.
	h := NewServer(Config{ModelName: "gormes-agent", DashboardBoundHost: "127.0.0.1"}).Handler()
	for _, path := range []string{"/ui/config", "/ui/skills", "/ui/env", "/ui/cron"} {
		rec := getUI(t, h, path)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s unwired = %d, want 200 graceful", path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "not wired") {
			t.Fatalf("GET %s unwired missing graceful notice; body=%s", path, rec.Body.String())
		}
	}
}

func TestAgentExecuteEscapesUserInput(t *testing.T) {
	h := newUITestServer().Handler()
	rec := httptest.NewRecorder()
	form := url.Values{"prompt": {`<script>alert(1)</script>`}}
	req := httptest.NewRequest(http.MethodPost, "/agent/execute", strings.NewReader(form.Encode()))
	req.Host = "127.0.0.1"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /agent/execute = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Fatalf("agent execute did not escape user input: %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Fatalf("agent execute expected escaped script tag; body=%s", body)
	}
}

func TestFragmentRequiresAuthWhenNetworkExposed(t *testing.T) {
	// Network-exposed bind + API key: browser-style request without a bearer
	// must be rejected so fragment data is not leaked.
	srv := NewServer(Config{
		ModelName:          "gormes-agent",
		APIKey:             "sk-strong-key-1234567890",
		DashboardBoundHost: "dash.example.com",
		ConfigSummary:      func() []DashboardKeyValue { return []DashboardKeyValue{{Key: "secretish", Value: "x"}} },
	})
	h := srv.Handler()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ui/config", nil)
	req.Host = "dash.example.com"
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("network-exposed /ui/config without auth = %d, want 401", rec.Code)
	}
}
