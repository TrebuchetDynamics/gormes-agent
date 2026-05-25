package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/doctor"
)

func TestDoctorCustomEndpointAllSet(t *testing.T) {
	cfg := config.Config{
		Hermes: config.HermesCfg{
			Endpoint: "https://example.invalid",
			APIKey:   "secret",
			Model:    "m",
		},
	}

	got := doctorCustomEndpointReadiness(cfg)

	if got.Name != "Custom endpoint" {
		t.Fatalf("Name = %q, want %q", got.Name, "Custom endpoint")
	}
	if got.Status != doctor.StatusPass {
		t.Fatalf("Status = %v, want %v", got.Status, doctor.StatusPass)
	}
	if !strings.Contains(got.Summary, "configured") {
		t.Fatalf("Summary = %q, want it to contain %q", got.Summary, "configured")
	}
	for _, item := range got.Items {
		if item.Status == doctor.StatusWarn {
			t.Fatalf("item %q has Status=Warn but expected none flagged: %+v", item.Name, item)
		}
	}
}

func TestDoctorCustomEndpointMissingAPIKey(t *testing.T) {
	cfg := config.Config{
		Hermes: config.HermesCfg{
			Endpoint: "https://example.invalid",
			APIKey:   "",
			Model:    "m",
		},
	}

	got := doctorCustomEndpointReadiness(cfg)

	if got.Status != doctor.StatusWarn {
		t.Fatalf("Status = %v, want %v", got.Status, doctor.StatusWarn)
	}
	apiKey, ok := findItem(got.Items, "api_key")
	if !ok {
		t.Fatalf("missing api_key item in: %+v", got.Items)
	}
	if apiKey.Status != doctor.StatusWarn {
		t.Fatalf("api_key item Status = %v, want %v", apiKey.Status, doctor.StatusWarn)
	}
	if apiKey.Note != "missing" {
		t.Fatalf("api_key item Note = %q, want %q", apiKey.Note, "missing")
	}
	if got.Summary != "configured endpoint=https://example.invalid missing=api_key" {
		t.Fatalf("Summary = %q, want missing field name", got.Summary)
	}
}

func TestDoctorCustomEndpointCodexReportsMissingOAuthAuthNotAPIKey(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	cfg := config.Config{
		Hermes: config.HermesCfg{
			Provider: config.CodexOAuthProvider,
			Endpoint: "https://chatgpt.com/backend-api/codex",
			Model:    "gpt-5.2",
		},
	}

	got := doctorCustomEndpointReadiness(cfg)

	if got.Status != doctor.StatusWarn {
		t.Fatalf("Status = %v, want %v", got.Status, doctor.StatusWarn)
	}
	auth, ok := findItem(got.Items, "auth")
	if !ok {
		t.Fatalf("missing auth item in: %+v", got.Items)
	}
	if auth.Status != doctor.StatusWarn || !strings.Contains(auth.Note, "gormes auth add openai-codex") {
		t.Fatalf("auth item = %+v, want OAuth setup guidance", auth)
	}
	if _, ok := findItem(got.Items, "api_key"); ok {
		t.Fatalf("Codex readiness should not ask for api_key: %+v", got.Items)
	}
	if got.Summary != "configured provider=openai-codex endpoint=https://chatgpt.com/backend-api/codex missing=auth" {
		t.Fatalf("Summary = %q, want missing OAuth auth", got.Summary)
	}
}

func TestDoctorCustomEndpointCodexCredentialPoolAuthPasses(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: config.CodexOAuthProvider}, []config.PooledCredential{{
		ID:           "codex-import",
		Label:        "Imported Codex CLI",
		AuthType:     config.CredentialAuthOAuth,
		Source:       config.CodexOAuthSourceCodexCLIImport,
		AccessToken:  "codex-access-secret",
		RefreshToken: "codex-refresh-secret",
		LastStatus:   config.CredentialStatusOK,
	}}); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}
	cfg := config.Config{
		Hermes: config.HermesCfg{
			Provider: config.CodexOAuthProvider,
			Endpoint: "https://chatgpt.com/backend-api/codex",
			Model:    "gpt-5.2",
		},
	}

	got := doctorCustomEndpointReadiness(cfg)

	if got.Status != doctor.StatusPass {
		t.Fatalf("Status = %v, want %v: %+v", got.Status, doctor.StatusPass, got)
	}
	auth, ok := findItem(got.Items, "auth")
	if !ok {
		t.Fatalf("missing auth item in: %+v", got.Items)
	}
	if auth.Status != doctor.StatusPass || auth.Note != "set" {
		t.Fatalf("auth item = %+v, want pass/set", auth)
	}
	if strings.Contains(got.Summary, "missing=auth") {
		t.Fatalf("Summary = %q, should not report missing auth", got.Summary)
	}
	formatted := got.Format()
	for _, leak := range []string{"codex-access-secret", "codex-refresh-secret"} {
		if strings.Contains(formatted, leak) {
			t.Fatalf("doctor readiness leaked credential %q:\n%s", leak, formatted)
		}
	}
}

func TestDoctorCustomEndpointCodexCredentialPoolMissingTokensWarns(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: config.CodexOAuthProvider}, []config.PooledCredential{{
		ID:         "codex-import",
		Label:      "Imported Codex CLI",
		AuthType:   config.CredentialAuthOAuth,
		Source:     config.CodexOAuthSourceCodexCLIImport,
		LastStatus: config.CredentialStatusOK,
	}}); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}
	cfg := config.Config{
		Hermes: config.HermesCfg{
			Provider: config.CodexOAuthProvider,
			Endpoint: "https://chatgpt.com/backend-api/codex",
			Model:    "gpt-5.2",
		},
	}

	got := doctorCustomEndpointReadiness(cfg)

	if got.Status != doctor.StatusWarn {
		t.Fatalf("Status = %v, want %v: %+v", got.Status, doctor.StatusWarn, got)
	}
	auth, ok := findItem(got.Items, "auth")
	if !ok {
		t.Fatalf("missing auth item in: %+v", got.Items)
	}
	if auth.Status != doctor.StatusWarn || !strings.Contains(auth.Note, "gormes auth add openai-codex") {
		t.Fatalf("auth item = %+v, want missing OAuth setup guidance", auth)
	}
}

func TestDoctorCustomEndpointDefaultModelMissingEndpointAndKey(t *testing.T) {
	cfg := config.Config{
		Hermes: config.HermesCfg{
			Model: "hermes-agent",
		},
	}

	got := doctorCustomEndpointReadiness(cfg)

	if got.Status != doctor.StatusWarn {
		t.Fatalf("Status = %v, want %v", got.Status, doctor.StatusWarn)
	}
	if got.Summary != "setup incomplete: missing endpoint, api_key" {
		t.Fatalf("Summary = %q, want setup guidance", got.Summary)
	}
	if strings.Contains(got.Summary, "endpoint=") || strings.Contains(got.Summary, "missing=2") {
		t.Fatalf("Summary kept ambiguous first-run wording: %q", got.Summary)
	}
}

func TestDoctorCustomEndpointMissingModel(t *testing.T) {
	cfg := config.Config{
		Hermes: config.HermesCfg{
			Endpoint: "https://example.invalid",
			APIKey:   "secret",
			Model:    "",
		},
	}

	got := doctorCustomEndpointReadiness(cfg)

	if got.Status != doctor.StatusFail {
		t.Fatalf("Status = %v, want %v", got.Status, doctor.StatusFail)
	}
	model, ok := findItem(got.Items, "model")
	if !ok {
		t.Fatalf("missing model item in: %+v", got.Items)
	}
	if model.Status != doctor.StatusFail {
		t.Fatalf("model item Status = %v, want %v", model.Status, doctor.StatusFail)
	}
	if model.Note != "missing" {
		t.Fatalf("model item Note = %q, want %q", model.Note, "missing")
	}
}

func TestDoctorCustomEndpointAllEmpty(t *testing.T) {
	cfg := config.Config{Hermes: config.HermesCfg{}}

	got := doctorCustomEndpointReadiness(cfg)

	if got.Status != doctor.StatusWarn {
		t.Fatalf("Status = %v, want %v", got.Status, doctor.StatusWarn)
	}
	if got.Summary != "disabled" {
		t.Fatalf("Summary = %q, want %q", got.Summary, "disabled")
	}
}

func TestDoctorCmdInvokesCustomEndpointReadiness(t *testing.T) {
	setupCustomEndpointDoctorEnv(t)
	t.Setenv("GORMES_ENDPOINT", "https://example.invalid")
	t.Setenv("GORMES_API_KEY", "secret")
	t.Setenv("GORMES_MODEL", "m")

	stdout, err := captureDoctorStdout(t, func() error {
		cmd := newRootCommand()
		cmd.SetArgs([]string{"doctor", "--offline"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s", err, stdout)
	}

	if !strings.Contains(stdout, "✓ Custom endpoint —") {
		t.Fatalf("stdout missing Custom endpoint PASS glyph line:\n%s", stdout)
	}
	if !strings.Contains(stdout, "configured") {
		t.Fatalf("stdout missing 'configured' summary:\n%s", stdout)
	}
}

func TestDoctorCodexProviderHealthUsesAuthReadinessNotGenericHealth(t *testing.T) {
	setupCustomEndpointDoctorEnv(t)
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		http.Error(w, "generic health endpoint should not be called for Codex", http.StatusNotFound)
	}))
	defer server.Close()
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]\nprovider = 'openai-codex'\nendpoint = '"+server.URL+"'\nmodel = 'gpt-5.2'\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: config.CodexOAuthProvider}, []config.PooledCredential{{
		ID:           "codex-import",
		Label:        "Imported Codex CLI",
		AuthType:     config.CredentialAuthOAuth,
		Source:       config.CodexOAuthSourceCodexCLIImport,
		AccessToken:  "codex-access-secret",
		RefreshToken: "codex-refresh-secret",
		LastStatus:   config.CredentialStatusOK,
	}}); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}

	stdout, err := captureDoctorStdout(t, func() error {
		cmd := newRootCommand()
		cmd.SetArgs([]string{"doctor", "--offline=false"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s", err, stdout)
	}
	if called {
		t.Fatalf("doctor called generic /health endpoint for Codex")
	}
	if !strings.Contains(stdout, "✓ Provider health — auth-ready") {
		t.Fatalf("stdout missing Codex auth-ready provider health:\n%s", stdout)
	}
	for _, leak := range []string{"codex-access-secret", "codex-refresh-secret"} {
		if strings.Contains(stdout, leak) {
			t.Fatalf("doctor leaked credential %q:\n%s", leak, stdout)
		}
	}
}

func TestDoctorOfflineOutputDoesNotMentionHermesAPIServer(t *testing.T) {
	setupCustomEndpointDoctorEnv(t)

	stdout, err := captureDoctorStdout(t, func() error {
		cmd := newRootCommand()
		cmd.SetArgs([]string{"doctor", "--offline"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s", err, stdout)
	}

	for _, forbidden := range []string{"api_server", "API_SERVER_ENABLED", "hermes gateway start"} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("doctor output contains obsolete Hermes API-server guidance %q:\n%s", forbidden, stdout)
		}
	}
}

func TestDoctorWebToolsStatusReportsManagedGateway(t *testing.T) {
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)
	if err := os.WriteFile(filepath.Join(gormesHome, "auth.json"), []byte(`{
  "providers": {
    "nous": {
      "access_token": "nous-doctor-token",
      "expires_at": "2999-01-01T00:00:00Z"
    }
  }
}`), 0o600); err != nil {
		t.Fatalf("write auth store: %v", err)
	}

	got := doctorWebToolsStatus(config.Config{
		Web: config.WebCfg{Backend: "firecrawl", UseGateway: true},
	})
	if got.Name != "Web tools" || got.Status != doctor.StatusPass {
		t.Fatalf("doctor web status = %+v, want Web tools PASS", got)
	}
	for _, want := range []string{"backend=firecrawl", "route=managed", "source=auth_store"} {
		if !strings.Contains(got.Summary, want) {
			t.Fatalf("summary = %q, want %q", got.Summary, want)
		}
	}
	toolset, ok := findItem(got.Items, "toolset")
	if !ok {
		t.Fatalf("missing toolset item in %+v", got.Items)
	}
	for _, name := range []string{"web_search", "web_extract", "web_crawl"} {
		if !strings.Contains(toolset.Note, name) {
			t.Fatalf("toolset note = %q, missing %s", toolset.Note, name)
		}
	}
	if strings.Contains(got.Format(), "nous-doctor-token") {
		t.Fatalf("doctor web status leaked token:\n%s", got.Format())
	}
}

func TestDoctorWebToolsStatusExplainsGoscraplingBrowserExtractOnly(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	got := doctorWebToolsStatus(config.Config{
		Web: config.WebCfg{Backend: "goscrapling_browser"},
	})
	if got.Name != "Web tools" || got.Status != doctor.StatusPass {
		t.Fatalf("doctor web status = %+v, want Web tools PASS", got)
	}
	for _, want := range []string{"backend=goscrapling_browser", "route=local", "source=config"} {
		if !strings.Contains(got.Summary, want) {
			t.Fatalf("summary = %q, want %q", got.Summary, want)
		}
	}
	toolset, ok := findItem(got.Items, "toolset")
	if !ok {
		t.Fatalf("missing toolset item in %+v", got.Items)
	}
	if strings.TrimSpace(toolset.Note) != "web_extract" {
		t.Fatalf("toolset note = %q, want only web_extract", toolset.Note)
	}
	requiresEnv, ok := findItem(got.Items, "requires_env")
	if !ok {
		t.Fatalf("missing requires_env item in %+v", got.Items)
	}
	if requiresEnv.Note != "none (browser runtime checked separately)" {
		t.Fatalf("requires_env note = %q, want no provider env for local browser backend", requiresEnv.Note)
	}
	formatted := got.Format()
	for _, want := range []string{"local browser extraction", "extract only", "web_extract"} {
		if !strings.Contains(formatted, want) {
			t.Fatalf("doctor web status missing %q:\n%s", want, formatted)
		}
	}
	for _, forbidden := range []string{"web_search", "web_crawl", "secret-token"} {
		if strings.Contains(formatted, forbidden) {
			t.Fatalf("doctor web status contains unsupported or secret text %q:\n%s", forbidden, formatted)
		}
	}
}

func findItem(items []doctor.ItemInfo, name string) (doctor.ItemInfo, bool) {
	for _, it := range items {
		if it.Name == name {
			return it, true
		}
	}
	return doctor.ItemInfo{}, false
}

func setupCustomEndpointDoctorEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes"))
}

func captureDoctorStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	orig := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe: %v", pipeErr)
	}
	os.Stdout = w

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = io.Copy(&buf, r)
	}()

	runErr := fn()
	_ = w.Close()
	<-done
	os.Stdout = orig
	return buf.String(), runErr
}
