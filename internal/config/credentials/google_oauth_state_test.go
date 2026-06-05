package credentials

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGoogleOAuthPendingAuth_RoundTrip(t *testing.T) {
	store := NewGoogleOAuthStateStore(t.TempDir())
	pending := GoogleOAuthPendingAuth{
		State:           "state-123",
		CodeVerifier:    "verifier-456",
		RedirectURI:     "http://localhost:1/callback",
		RequestedScopes: []string{"email", "calendar.readonly"},
	}
	if err := store.SavePendingAuth(pending); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadPendingAuth()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, pending) {
		t.Fatalf("LoadPendingAuth() = %#v, want %#v", loaded, pending)
	}
}

func TestGoogleOAuthExtractCodeAndState(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantCode   string
		wantState  string
		wantScopes []string
	}{
		{name: "raw code", input: "  raw-code-123  ", wantCode: "raw-code-123"},
		{name: "callback URL", input: "http://localhost:1/?code=url-code&state=state-1&scope=email%20calendar.readonly", wantCode: "url-code", wantState: "state-1", wantScopes: []string{"email", "calendar.readonly"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtractGoogleOAuthCodeAndState(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if got.Code != tc.wantCode || got.State != tc.wantState || !reflect.DeepEqual(got.Scopes, tc.wantScopes) {
				t.Fatalf("ExtractGoogleOAuthCodeAndState() = %#v", got)
			}
		})
	}
}

func TestGoogleOAuthExchange_StoresGrantedScopes(t *testing.T) {
	store := NewGoogleOAuthStateStore(t.TempDir())
	pending := GoogleOAuthPendingAuth{State: "state-1", CodeVerifier: "verifier", RedirectURI: "http://localhost:1", RequestedScopes: []string{"email", "calendar.readonly", "drive.readonly"}}
	if err := store.SavePendingAuth(pending); err != nil {
		t.Fatal(err)
	}
	callback := GoogleOAuthCallback{Code: "code-1", State: "state-1", Scopes: []string{"email", "calendar.readonly"}}
	credential := []byte(`{"client_id":"client-id","client_secret":"plain-client-secret","refresh_token":"plain-refresh-token","token_uri":"https://oauth2.example/token"}`)
	status, err := store.ExchangeAuthCode(callback, credential)
	if err != nil {
		t.Fatal(err)
	}
	if status.Code != GoogleOAuthStatusAuthorized {
		t.Fatalf("status = %#v", status)
	}
	var token map[string]any
	readJSONFile(t, store.TokenPath(), &token)
	if token["type"] != "authorized_user" {
		t.Fatalf("type = %#v", token["type"])
	}
	if got := stringSliceFromAny(t, token["scopes"]); !reflect.DeepEqual(got, []string{"email", "calendar.readonly"}) {
		t.Fatalf("scopes = %#v", got)
	}
	if _, err := os.Stat(store.PendingPath()); !os.IsNotExist(err) {
		t.Fatalf("pending auth file should be removed after exchange, err=%v", err)
	}
}

func TestGoogleOAuthCheckAuth_PartialScopeEvidence(t *testing.T) {
	store := NewGoogleOAuthStateStore(t.TempDir())
	writeGoogleOAuthToken(t, store.TokenPath(), map[string]any{
		"type":          "authorized_user",
		"client_id":     "client-id",
		"client_secret": "plain-client-secret",
		"refresh_token": "plain-refresh-token",
		"scopes":        []string{"email"},
	})
	status, err := store.CheckAuth([]string{"email", "calendar.readonly"})
	if err != nil {
		t.Fatal(err)
	}
	if status.Code != GoogleOAuthStatusPartialScope || !status.Authenticated {
		t.Fatalf("status = %#v, want token_partial_scope while authenticated", status)
	}
	if !reflect.DeepEqual(status.MissingScopes, []string{"calendar.readonly"}) {
		t.Fatalf("MissingScopes = %#v", status.MissingScopes)
	}
	if status.ClientSecret != "" || status.AccessToken != "" || status.RefreshToken != "" {
		t.Fatalf("status leaked secret-bearing fields: %#v", status)
	}
}

func TestGoogleOAuthExchange_StateMismatch(t *testing.T) {
	dir := t.TempDir()
	store := NewGoogleOAuthStateStore(dir)
	pending := GoogleOAuthPendingAuth{State: "state-expected", CodeVerifier: "verifier", RedirectURI: "http://localhost:1", RequestedScopes: []string{"email"}}
	if err := store.SavePendingAuth(pending); err != nil {
		t.Fatal(err)
	}
	preexistingToken := filepath.Join(dir, "google-token.json")
	if err := os.WriteFile(preexistingToken, []byte(`{"type":"authorized_user","refresh_token":"existing-refresh"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	status, err := store.ExchangeAuthCode(GoogleOAuthCallback{Code: "code-1", State: "wrong-state", Scopes: []string{"email"}}, []byte(`{"refresh_token":"new-refresh"}`))
	if err != nil {
		t.Fatal(err)
	}
	if status.Code != GoogleOAuthStatusStateMismatch {
		t.Fatalf("status = %#v", status)
	}
	loaded, err := store.LoadPendingAuth()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, pending) {
		t.Fatalf("pending = %#v, want untouched %#v", loaded, pending)
	}
	gotToken, err := os.ReadFile(preexistingToken)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotToken) != `{"type":"authorized_user","refresh_token":"existing-refresh"}` {
		t.Fatalf("token file changed: %s", gotToken)
	}
}

func readJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func writeGoogleOAuthToken(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func stringSliceFromAny(t *testing.T, value any) []string {
	t.Helper()
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("value = %T %#v, want []any", value, value)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			t.Fatalf("item = %T %#v, want string", item, item)
		}
		out = append(out, text)
	}
	return out
}
