package navivox

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

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestNavivoxConfigAdminRequiresAuthAndRedactsSchemaAndGet(t *testing.T) {
	home := filepath.Join(t.TempDir(), "gormes")
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "super-secret-navivox-token")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	configBody := `
[navivox]
enabled = true
bind_host = "127.0.0.1"
port = 8765
exposure_mode = "local"
auth_mode = "pairing_token"
token = "raw-config-token-should-not-leak"
allow_origins = ["http://localhost:3000"]
`
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(configBody), 0o600); err != nil {
		t.Fatal(err)
	}

	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	unauth, err := http.Get(server.URL + "/v1/navivox/config-admin/schema")
	if err != nil {
		t.Fatal(err)
	}
	defer unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized config-admin schema status = %d, want 401", unauth.StatusCode)
	}

	schemaResp := doNavivoxConfigAdminRequest(t, server.URL, http.MethodGet, "/v1/navivox/config-admin/schema", "")
	defer schemaResp.Body.Close()
	if schemaResp.StatusCode != http.StatusOK {
		t.Fatalf("schema status = %d, want 200", schemaResp.StatusCode)
	}
	var schema struct {
		Action string                       `json:"action"`
		Fields []configAdminSchemaFieldTest `json:"fields"`
	}
	if err := json.NewDecoder(schemaResp.Body).Decode(&schema); err != nil {
		t.Fatal(err)
	}
	if schema.Action != "config.schema" || !configAdminTestFieldPresent(schema.Fields, "navivox.port") {
		t.Fatalf("schema payload = %+v, want config.schema with navivox.port", schema)
	}
	secretField := configAdminTestFieldByKey(schema.Fields, "navivox.token")
	if !secretField.Secret || !containsString(secretField.Actions, "set") || !containsString(secretField.Actions, "delete") {
		t.Fatalf("navivox.token schema = %+v, want secret set/delete actions", secretField)
	}

	getResp := doNavivoxConfigAdminRequest(t, server.URL, http.MethodGet, "/v1/navivox/config-admin", "")
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", getResp.StatusCode)
	}
	var raw bytes.Buffer
	if _, err := raw.ReadFrom(getResp.Body); err != nil {
		t.Fatal(err)
	}
	body := raw.String()
	for _, forbidden := range []string{"super-secret-navivox-token", "raw-config-token-should-not-leak"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("config-admin get leaked secret %q in %s", forbidden, body)
		}
	}
	var got struct {
		Action string                      `json:"action"`
		Values []configAdminValueStateTest `json:"values"`
	}
	if err := json.Unmarshal(raw.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Action != "config.get" {
		t.Fatalf("get action = %q, want config.get", got.Action)
	}
	port := configAdminValueByKey(got.Values, "navivox.port")
	if port.Value == nil || port.Secret {
		t.Fatalf("navivox.port value = %+v, want non-secret value", port)
	}
	token := configAdminValueByKey(got.Values, "navivox.token")
	if !token.Secret || token.SecretStatus != "set" || token.Value != nil {
		t.Fatalf("navivox.token value = %+v, want redacted set secret with no value", token)
	}
}

func TestNavivoxConfigAdminValidateAndApplyAreNonMutatingOnError(t *testing.T) {
	home := filepath.Join(t.TempDir(), "gormes")
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "existing-secret-token")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(home, "config.toml")
	original := []byte(`
[navivox]
enabled = true
bind_host = "127.0.0.1"
port = 8765
exposure_mode = "local"
auth_mode = "pairing_token"
`)
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	validateResp := doNavivoxConfigAdminRequest(t, server.URL, http.MethodPost, "/v1/navivox/config-admin/validate", `{"changes":[{"key":"navivox.bind_host","value":"0.0.0.0"}]}`)
	defer validateResp.Body.Close()
	if validateResp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("validate status = %d, want 422", validateResp.StatusCode)
	}
	var validation struct {
		Action string `json:"action"`
		Valid  bool   `json:"valid"`
		Errors []struct {
			Key     string `json:"key"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(validateResp.Body).Decode(&validation); err != nil {
		t.Fatal(err)
	}
	if validation.Action != "config.validate" || validation.Valid || len(validation.Errors) == 0 || validation.Errors[0].Key != "navivox.bind_host" {
		t.Fatalf("validation payload = %+v, want field-scoped bind_host error", validation)
	}
	if body, err := os.ReadFile(configPath); err != nil || !bytes.Equal(body, original) {
		t.Fatalf("config changed after validate failure: err=%v body=%q", err, string(body))
	}

	applyResp := doNavivoxConfigAdminRequest(t, server.URL, http.MethodPost, "/v1/navivox/config-admin/apply", `{"changes":[{"key":"navivox.bind_host","value":"0.0.0.0"}]}`)
	defer applyResp.Body.Close()
	if applyResp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("apply-invalid status = %d, want 422", applyResp.StatusCode)
	}
	if body, err := os.ReadFile(configPath); err != nil || !bytes.Equal(body, original) {
		t.Fatalf("config changed after apply failure: err=%v body=%q", err, string(body))
	}
}

func TestNavivoxConfigAdminDiffAndApplyReportsReloadApplied(t *testing.T) {
	home := filepath.Join(t.TempDir(), "gormes")
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "existing-secret-token")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(home, "config.toml")
	original := []byte(`
[navivox]
enabled = true
bind_host = "127.0.0.1"
port = 8765
exposure_mode = "local"
auth_mode = "pairing_token"
`)
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	reloadCalls := 0
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		BindHost:     config.NavivoxDefaultBindHost,
		Port:         config.NavivoxDefaultPort,
		ExposureMode: config.NavivoxExposureLocal,
		AuthMode:     config.NavivoxAuthPairingToken,
		Token:        "nvbx_test_token",
		AllowOrigins: []string{"*"},
	}, nil, WithConfigAdminReloader(func(_ context.Context) error {
		reloadCalls++
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	payload := `{"changes":[{"key":"navivox.port","value":"8766"}]}`
	diffResp := doNavivoxConfigAdminRequest(t, server.URL, http.MethodPost, "/v1/navivox/config-admin/diff", payload)
	defer diffResp.Body.Close()
	if diffResp.StatusCode != http.StatusOK {
		t.Fatalf("diff status = %d, want 200", diffResp.StatusCode)
	}
	var diffed struct {
		Action  string                     `json:"action"`
		Valid   bool                       `json:"valid"`
		Changes []configAdminDiffEntryTest `json:"changes"`
	}
	if err := json.NewDecoder(diffResp.Body).Decode(&diffed); err != nil {
		t.Fatal(err)
	}
	portDiff := configAdminDiffByKey(diffed.Changes, "navivox.port")
	if diffed.Action != "config.diff" || !diffed.Valid || portDiff.Before == nil || portDiff.After == nil {
		t.Fatalf("diff payload = %+v, want port before/after", diffed)
	}
	if body, err := os.ReadFile(configPath); err != nil || !bytes.Equal(body, original) {
		t.Fatalf("config changed after diff: err=%v body=%q", err, string(body))
	}

	applyResp := doNavivoxConfigAdminRequest(t, server.URL, http.MethodPost, "/v1/navivox/config-admin/apply", payload)
	defer applyResp.Body.Close()
	if applyResp.StatusCode != http.StatusOK {
		t.Fatalf("apply status = %d, want 200", applyResp.StatusCode)
	}
	var applied struct {
		Action         string `json:"action"`
		Applied        bool   `json:"applied"`
		ReloadApplied  bool   `json:"reload_applied"`
		PendingRestart bool   `json:"pending_restart"`
	}
	if err := json.NewDecoder(applyResp.Body).Decode(&applied); err != nil {
		t.Fatal(err)
	}
	if applied.Action != "config.apply" || !applied.Applied || !applied.ReloadApplied || applied.PendingRestart || reloadCalls != 1 {
		t.Fatalf("apply payload = %+v reloadCalls=%d, want reload_applied", applied, reloadCalls)
	}
}

func TestNavivoxConfigAdminApplyWritesSafeValuesAndSecretRedacted(t *testing.T) {
	home := filepath.Join(t.TempDir(), "gormes")
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "existing-secret-token")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
[navivox]
enabled = true
bind_host = "127.0.0.1"
port = 8765
exposure_mode = "local"
auth_mode = "pairing_token"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	payload := `{"changes":[{"key":"navivox.port","value":"8766"},{"key":"navivox.token","value":"new-secret-token"}]}`
	applyResp := doNavivoxConfigAdminRequest(t, server.URL, http.MethodPost, "/v1/navivox/config-admin/apply", payload)
	defer applyResp.Body.Close()
	if applyResp.StatusCode != http.StatusOK {
		t.Fatalf("apply status = %d, want 200", applyResp.StatusCode)
	}
	var raw bytes.Buffer
	if _, err := raw.ReadFrom(applyResp.Body); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw.String(), "new-secret-token") {
		t.Fatalf("apply response leaked secret: %s", raw.String())
	}
	var applied struct {
		Action         string                     `json:"action"`
		Applied        bool                       `json:"applied"`
		ReloadApplied  bool                       `json:"reload_applied"`
		PendingRestart bool                       `json:"pending_restart"`
		Changes        []configAdminDiffEntryTest `json:"changes"`
	}
	if err := json.Unmarshal(raw.Bytes(), &applied); err != nil {
		t.Fatal(err)
	}
	if applied.Action != "config.apply" || !applied.Applied || applied.ReloadApplied || !applied.PendingRestart {
		t.Fatalf("apply payload = %+v, want applied with pending_restart when no reloader exists", applied)
	}
	tokenDiff := configAdminDiffByKey(applied.Changes, "navivox.token")
	if !tokenDiff.Secret || !tokenDiff.AfterRedacted {
		t.Fatalf("token diff = %+v, want redacted secret diff", tokenDiff)
	}
	configBody, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configBody), "port = 8766") || strings.Contains(string(configBody), "new-secret-token") {
		t.Fatalf("config.toml after apply = %q, want port update and no secret", string(configBody))
	}
	envBody, err := os.ReadFile(filepath.Join(home, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(envBody), "GORMES_NAVIVOX_TOKEN=new-secret-token") {
		t.Fatalf(".env after apply = %q, want navivox token", string(envBody))
	}
}

func doNavivoxConfigAdminRequest(t *testing.T, baseURL, method, path, body string) *http.Response {
	t.Helper()
	var reader *strings.Reader
	reader = strings.NewReader(body)
	req, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer nvbx_test_token")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

type configAdminSchemaFieldTest struct {
	Key     string   `json:"key"`
	Secret  bool     `json:"secret"`
	Actions []string `json:"actions"`
}

type configAdminValueStateTest struct {
	Key          string `json:"key"`
	Value        any    `json:"value,omitempty"`
	Secret       bool   `json:"secret"`
	SecretStatus string `json:"secret_status,omitempty"`
	Source       string `json:"source,omitempty"`
}

type configAdminDiffEntryTest struct {
	Key            string `json:"key"`
	Secret         bool   `json:"secret"`
	Before         any    `json:"before,omitempty"`
	After          any    `json:"after,omitempty"`
	BeforeRedacted bool   `json:"before_redacted"`
	AfterRedacted  bool   `json:"after_redacted"`
}

func configAdminTestFieldPresent(fields []configAdminSchemaFieldTest, key string) bool {
	for _, field := range fields {
		if field.Key == key {
			return true
		}
	}
	return false
}

func configAdminTestFieldByKey(fields []configAdminSchemaFieldTest, key string) configAdminSchemaFieldTest {
	for _, field := range fields {
		if field.Key == key {
			return field
		}
	}
	return configAdminSchemaFieldTest{}
}

func configAdminValueByKey(values []configAdminValueStateTest, key string) configAdminValueStateTest {
	for _, value := range values {
		if value.Key == key {
			return value
		}
	}
	return configAdminValueStateTest{}
}

func configAdminDiffByKey(values []configAdminDiffEntryTest, key string) configAdminDiffEntryTest {
	for _, value := range values {
		if value.Key == key {
			return value
		}
	}
	return configAdminDiffEntryTest{}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
