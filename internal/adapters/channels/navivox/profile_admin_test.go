package navivox

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func newTestChannelWithOpts(t *testing.T, opts ...ChannelOption) *Channel {
	t.Helper()
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		BindHost:     config.NavivoxDefaultBindHost,
		Port:         config.NavivoxDefaultPort,
		ExposureMode: config.NavivoxExposureLocal,
		AuthMode:     config.NavivoxAuthPairingToken,
		Token:        "nvbx_test_token",
		AllowOrigins: []string{"*"},
	}, nil, opts...)
	if err != nil {
		t.Fatal(err)
	}
	ch.newID = func() string { return "generated-id" }
	return ch
}

func TestProfileAdminListEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_test_token")

	ch := newTestChannelWithOpts(t, WithProfileAdminHome(func() string { return home }))
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var got struct {
		Action   string              `json:"action"`
		Profiles []profileAdminEntry `json:"profiles"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/profile-admin", "", http.StatusOK, &got)
	if got.Action != "profile_admin.list" {
		t.Fatalf("action = %q, want profile_admin.list", got.Action)
	}
	if got.Profiles == nil {
		t.Fatal("profiles should be empty slice, not nil")
	}
	if len(got.Profiles) != 0 {
		t.Fatalf("profiles = %v, want empty", got.Profiles)
	}
}

func TestProfileAdminListWithProfiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_test_token")

	// Create two profile directories with config.toml
	for _, id := range []string{"dev-agent", "research"} {
		dir := filepath.Join(home, "profiles", id)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		cfg := "[hermes]\nprovider = \"openai\"\nmodel = \"gpt-4o\"\n"
		if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(cfg), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Directory without config.toml should be ignored
	if err := os.MkdirAll(filepath.Join(home, "profiles", "no-config"), 0o700); err != nil {
		t.Fatal(err)
	}

	ch := newTestChannelWithOpts(t, WithProfileAdminHome(func() string { return home }))
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var got struct {
		Action   string              `json:"action"`
		Profiles []profileAdminEntry `json:"profiles"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/profile-admin", "", http.StatusOK, &got)
	if got.Action != "profile_admin.list" {
		t.Fatalf("action = %q, want profile_admin.list", got.Action)
	}
	if len(got.Profiles) != 2 {
		t.Fatalf("profiles count = %d, want 2; got %v", len(got.Profiles), got.Profiles)
	}
	if got.Profiles[0].ProfileID != "dev-agent" || got.Profiles[1].ProfileID != "research" {
		t.Fatalf("profile IDs = %v, want [dev-agent research]", got.Profiles)
	}
}

func TestProfileAdminGetValues(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_test_token")

	dir := filepath.Join(home, "profiles", "my-agent")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := "[hermes]\nprovider = \"anthropic\"\nmodel = \"claude-sonnet-4-6\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	ch := newTestChannelWithOpts(t, WithProfileAdminHome(func() string { return home }))
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var got struct {
		Action    string                     `json:"action"`
		ProfileID string                     `json:"profile_id"`
		Values    []configAdminValueStateTest `json:"values"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/profile-admin/my-agent", "", http.StatusOK, &got)
	if got.Action != "profile_admin.get" {
		t.Fatalf("action = %q, want profile_admin.get", got.Action)
	}
	if got.ProfileID != "my-agent" {
		t.Fatalf("profile_id = %q, want my-agent", got.ProfileID)
	}
	provider := configAdminTestStringValueByKey(got.Values, "hermes.provider")
	if provider != "anthropic" {
		t.Fatalf("hermes.provider = %q, want anthropic", provider)
	}
	model := configAdminTestStringValueByKey(got.Values, "hermes.model")
	if model != "claude-sonnet-4-6" {
		t.Fatalf("hermes.model = %q, want claude-sonnet-4-6", model)
	}
}

func TestProfileAdminGetNotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_test_token")

	ch := newTestChannelWithOpts(t, WithProfileAdminHome(func() string { return home }))
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	httpc.Raw(http.MethodGet, "/v1/navivox/profile-admin/nonexistent", "", http.StatusNotFound)
}

func TestProfileAdminSchema(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_test_token")

	dir := filepath.Join(home, "profiles", "my-agent")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[hermes]\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ch := newTestChannelWithOpts(t, WithProfileAdminHome(func() string { return home }))
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var got struct {
		Action    string                      `json:"action"`
		ProfileID string                      `json:"profile_id"`
		Fields    []configAdminSchemaFieldTest `json:"fields"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/profile-admin/my-agent/schema", "", http.StatusOK, &got)
	if got.Action != "profile_admin.schema" {
		t.Fatalf("action = %q, want profile_admin.schema", got.Action)
	}
	if !configAdminTestFieldPresent(got.Fields, "hermes.provider") {
		t.Fatal("hermes.provider not in profile admin schema")
	}
	if !configAdminTestFieldPresent(got.Fields, "hermes.model") {
		t.Fatal("hermes.model not in profile admin schema")
	}
	if !configAdminTestFieldPresent(got.Fields, "agents.defaults.workspace") {
		t.Fatal("agents.defaults.workspace not in profile admin schema")
	}
}

func TestProfileAdminApplyUpdatesConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_test_token")

	dir := filepath.Join(home, "profiles", "my-agent")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[hermes]\nprovider = \"openai\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ch := newTestChannelWithOpts(t, WithProfileAdminHome(func() string { return home }))
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	body := `{"changes":[{"key":"hermes.provider","value":"anthropic"},{"key":"hermes.model","value":"claude-haiku-4-5"}]}`
	var got struct {
		Action    string `json:"action"`
		ProfileID string `json:"profile_id"`
		Applied   bool   `json:"applied"`
		Valid     bool   `json:"valid"`
	}
	httpc.JSON(http.MethodPost, "/v1/navivox/profile-admin/my-agent/apply", body, http.StatusOK, &got)
	if got.Action != "profile_admin.apply" {
		t.Fatalf("action = %q, want profile_admin.apply", got.Action)
	}
	if !got.Applied || !got.Valid {
		t.Fatalf("applied=%v valid=%v, want both true", got.Applied, got.Valid)
	}

	// Verify the config was actually updated by re-reading
	var check struct {
		Action string                     `json:"action"`
		Values []configAdminValueStateTest `json:"values"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/profile-admin/my-agent", "", http.StatusOK, &check)
	provider := configAdminTestStringValueByKey(check.Values, "hermes.provider")
	if provider != "anthropic" {
		t.Fatalf("after apply, hermes.provider = %q, want anthropic", provider)
	}
	model := configAdminTestStringValueByKey(check.Values, "hermes.model")
	if model != "claude-haiku-4-5" {
		t.Fatalf("after apply, hermes.model = %q, want claude-haiku-4-5", model)
	}
}

func TestProfileAdminApplyUnsupportedField(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_test_token")

	dir := filepath.Join(home, "profiles", "my-agent")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	ch := newTestChannelWithOpts(t, WithProfileAdminHome(func() string { return home }))
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	body := `{"changes":[{"key":"navivox.token","value":"should-not-be-allowed"}]}`
	httpc.Raw(http.MethodPost, "/v1/navivox/profile-admin/my-agent/apply", body, http.StatusUnprocessableEntity)
}

func TestProfileAdminRequiresAuth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_test_token")

	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	resp, err := http.Get(server.URL + "/v1/navivox/profile-admin")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated profile-admin status = %d, want 401", resp.StatusCode)
	}
}

func TestConfigAdminSchemaIncludesHermesFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_test_token")

	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var got struct {
		Action string                      `json:"action"`
		Fields []configAdminSchemaFieldTest `json:"fields"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/config-admin/schema", "", http.StatusOK, &got)
	for _, key := range []string{"hermes.provider", "hermes.model", "hermes.api_key", "hermes.endpoint"} {
		if !configAdminTestFieldPresent(got.Fields, key) {
			t.Errorf("global config-admin schema missing field %q", key)
		}
	}
	apiKeyField := configAdminTestFieldByKey(got.Fields, "hermes.api_key")
	if !apiKeyField.Secret {
		t.Fatal("hermes.api_key should be a secret field")
	}
}

func TestConfigAdminGetShowsHermesValues(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_test_token")

	cfg := "[hermes]\nprovider = \"openrouter\"\nmodel = \"anthropic/claude-sonnet-4-6\"\n[navivox]\nenabled = true\nbind_host = \"127.0.0.1\"\nport = 8765\nexposure_mode = \"local\"\nauth_mode = \"pairing_token\"\n"
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	fullCfg := config.Config{
		Navivox: config.NavivoxCfg{
			Enabled:      true,
			BindHost:     "127.0.0.1",
			Port:         8765,
			ExposureMode: config.NavivoxExposureLocal,
			AuthMode:     config.NavivoxAuthPairingToken,
			Token:        "nvbx_test_token",
			AllowOrigins: []string{"*"},
		},
		Hermes: config.HermesCfg{
			Provider: "openrouter",
			Model:    "anthropic/claude-sonnet-4-6",
		},
	}
	ch := newTestChannelWithOpts(t, WithRuntimeConfig(fullCfg))
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var got struct {
		Action string                     `json:"action"`
		Values []configAdminValueStateTest `json:"values"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/config-admin", "", http.StatusOK, &got)
	provider := configAdminTestStringValueByKey(got.Values, "hermes.provider")
	if provider != "openrouter" {
		t.Fatalf("hermes.provider = %q, want openrouter", provider)
	}
	model := configAdminTestStringValueByKey(got.Values, "hermes.model")
	if model != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("hermes.model = %q, want anthropic/claude-sonnet-4-6", model)
	}
}

// configAdminTestStringValueByKey finds a value state by key and returns its string value.
func configAdminTestStringValueByKey(values []configAdminValueStateTest, key string) string {
	for _, v := range values {
		if v.Key == key {
			if s, ok := v.Value.(string); ok {
				return s
			}
		}
	}
	return ""
}
