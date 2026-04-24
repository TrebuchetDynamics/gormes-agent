package site

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServer_ServesEmbeddedCSS(t *testing.T) {
	handler, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest("GET", "/static/site.css", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/css") {
		t.Fatalf("content-type = %q, want text/css", ct)
	}
	if !strings.Contains(rr.Body.String(), "--bg-0") {
		t.Fatalf("css is missing expected design variables")
	}
}

func TestServer_IndexLinksCSSAndScopesScripts(t *testing.T) {
	handler, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	body := rr.Body.String()
	if !strings.Contains(body, `href="/static/site.css"`) {
		t.Fatalf("index is missing stylesheet link\n%s", body)
	}
	// Exactly one inline <script> is allowed: the gormesCopy clipboard helper
	// for the install-step copy buttons. Anything more would be JS creep.
	scriptCount := strings.Count(strings.ToLower(body), "<script")
	if scriptCount != 1 {
		t.Fatalf("expected exactly 1 <script> tag (clipboard helper); found %d\n%s", scriptCount, body)
	}
	if !strings.Contains(body, "navigator.clipboard.writeText") {
		t.Fatalf("inline script is not the clipboard helper\n%s", body)
	}
	if strings.Contains(body, `<script src="`) {
		t.Fatalf("external script source not permitted\n%s", body)
	}
}

func TestServer_UnknownRoutesReturn404(t *testing.T) {
	handler, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest("GET", "/missing", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != 404 {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestServer_ServesInstallScript(t *testing.T) {
	handler, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest("GET", "/install.sh", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/x-shellscript") {
		t.Fatalf("content-type = %q, want text/x-shellscript", ct)
	}

	body := rr.Body.String()
	for _, want := range []string{
		"git@github.com:TrebuchetDynamics/gormes-agent.git",
		"https://github.com/TrebuchetDynamics/gormes-agent.git",
		`go build -o "$build_bin" ./cmd/gormes`,
		"GORMES_INSTALL_HOME",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("install.sh missing %q\n%s", want, body)
		}
	}
	for _, reject := range []string{
		"XelHaku/golang-hermes-agent",
		"XelHaku/gormes-agent",
		"releases/latest",
		`go install "${MODULE}@${VERSION}"`,
	} {
		if strings.Contains(body, reject) {
			t.Fatalf("install.sh contains stale installer path %q\n%s", reject, body)
		}
	}
}
