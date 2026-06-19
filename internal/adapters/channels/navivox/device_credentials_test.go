package navivox

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestNavivoxDeviceCredentialIssueListRevoke(t *testing.T) {
	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var issued struct {
		CredentialID string   `json:"credential_id"`
		Secret       string   `json:"secret"`
		AuthMethod   string   `json:"auth_method"`
		Interim      bool     `json:"interim"`
		Scopes       []string `json:"scopes"`
		AppInstallID string   `json:"app_install_id"`
	}
	httpc.JSON(http.MethodPost, "/v1/navivox/device-credentials",
		`{"app_install_id":"install-1"}`, http.StatusCreated, &issued)

	if issued.CredentialID == "" || !strings.HasPrefix(issued.Secret, "nvbxdc_") {
		t.Fatalf("issued credential = %+v, want id and nvbxdc_ secret", issued)
	}
	if issued.AuthMethod != "device_bearer" || !issued.Interim {
		t.Fatalf("issued auth_method=%q interim=%v, want device_bearer interim", issued.AuthMethod, issued.Interim)
	}
	if len(issued.Scopes) != 1 || issued.Scopes[0] != "navivox" {
		t.Fatalf("issued scopes = %v, want [navivox]", issued.Scopes)
	}

	listBody := httpc.Raw(http.MethodGet,
		"/v1/navivox/device-credentials?app_install_id=install-1", "", http.StatusOK)
	if bytes.Contains(bytes.ToLower(listBody), []byte("secret")) {
		t.Fatalf("list response leaked a secret field: %s", listBody)
	}
	if bytes.Contains(listBody, []byte(issued.Secret)) {
		t.Fatalf("list response leaked the raw secret value")
	}

	var list struct {
		Credentials []struct {
			CredentialID string `json:"credential_id"`
			AppInstallID string `json:"app_install_id"`
			Revoked      bool   `json:"revoked"`
		} `json:"credentials"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/device-credentials?app_install_id=install-1",
		"", http.StatusOK, &list)
	if len(list.Credentials) != 1 || list.Credentials[0].CredentialID != issued.CredentialID {
		t.Fatalf("list = %+v, want the issued credential", list.Credentials)
	}
	if list.Credentials[0].Revoked {
		t.Fatalf("freshly issued credential reported revoked")
	}

	var revoked struct {
		CredentialID string `json:"credential_id"`
		Revoked      bool   `json:"revoked"`
	}
	httpc.JSON(http.MethodPost, "/v1/navivox/device-credentials/revoke",
		`{"credential_id":"`+issued.CredentialID+`"}`, http.StatusOK, &revoked)
	if !revoked.Revoked || revoked.CredentialID != issued.CredentialID {
		t.Fatalf("revoke response = %+v, want revoked true", revoked)
	}

	var afterRevoke struct {
		Credentials []struct {
			Revoked bool `json:"revoked"`
		} `json:"credentials"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/device-credentials?app_install_id=install-1",
		"", http.StatusOK, &afterRevoke)
	if len(afterRevoke.Credentials) != 1 || !afterRevoke.Credentials[0].Revoked {
		t.Fatalf("after revoke = %+v, want the credential marked revoked", afterRevoke.Credentials)
	}
}

func TestNavivoxDeviceCredentialIssueRequiresAppInstallID(t *testing.T) {
	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	httpc.Raw(http.MethodPost, "/v1/navivox/device-credentials", `{}`, http.StatusBadRequest)
}

func TestNavivoxDeviceCredentialRevokeIsIdempotentForUnknownID(t *testing.T) {
	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var revoked struct {
		Revoked bool `json:"revoked"`
	}
	httpc.JSON(http.MethodPost, "/v1/navivox/device-credentials/revoke",
		`{"credential_id":"navivoxcred_unknown"}`, http.StatusOK, &revoked)
	if !revoked.Revoked {
		t.Fatalf("revoke of unknown id = %+v, want idempotent revoked true", revoked)
	}
}

func TestNavivoxDeviceCredentialPersistenceRoundtrip(t *testing.T) {
	dir := t.TempDir()
	credPath := filepath.Join(dir, "creds.json")

	newChannel := func() *Channel {
		ch, err := NewChannel(config.NavivoxCfg{
			Enabled:      true,
			BindHost:     "127.0.0.1",
			Port:         8765,
			ExposureMode: "local",
			AuthMode:     "pairing_token",
			Token:        "nvbx_test_token",
			AllowOrigins: []string{"*"},
		}, nil, WithCredentialsPath(credPath))
		if err != nil {
			t.Fatalf("NewChannel: %v", err)
		}
		ch.newID = func() string { return "generated-id" }
		return ch
	}

	// Issue a credential with persistence enabled.
	ch1 := newChannel()
	server1 := httptest.NewServer(ch1.Handler(make(chan gateway.InboundEvent, 1)))
	defer server1.Close()
	httpc1 := newNavivoxHTTPContract(t, server1.URL)

	var issued struct {
		CredentialID string `json:"credential_id"`
		Secret       string `json:"secret"`
	}
	httpc1.JSON(http.MethodPost, "/v1/navivox/device-credentials",
		`{"app_install_id":"install-persist"}`, http.StatusCreated, &issued)

	if issued.CredentialID == "" || issued.Secret == "" {
		t.Fatalf("issue failed: %+v", issued)
	}

	// Verify the file was created.
	if _, err := os.Stat(credPath); err != nil {
		t.Fatalf("credentials file not created: %v", err)
	}

	// Start a second channel loaded from the same file (simulates gormes restart).
	ch2 := newChannel()
	server2 := httptest.NewServer(ch2.Handler(make(chan gateway.InboundEvent, 1)))
	defer server2.Close()
	httpc2 := newNavivoxHTTPContract(t, server2.URL)

	// The credential issued by ch1 should authenticate on ch2.
	bearerToken := issued.CredentialID + ":" + issued.Secret
	req, _ := http.NewRequest(http.MethodGet, server2.URL+"/v1/navivox/status", nil)
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("reconnect request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatalf("credential not loaded from disk: got 401 after restart")
	}

	// Revocation on ch2 should persist and be reflected in the file.
	var revoked struct{ Revoked bool `json:"revoked"` }
	httpc2.JSON(http.MethodPost, "/v1/navivox/device-credentials/revoke",
		`{"credential_id":"`+issued.CredentialID+`"}`, http.StatusOK, &revoked)
	if !revoked.Revoked {
		t.Fatalf("revoke failed")
	}

	// Load the file and confirm the record is marked revoked.
	loaded, err := loadCredentialsFromDisk(credPath)
	if err != nil {
		t.Fatalf("load after revoke: %v", err)
	}
	rec, ok := loaded[issued.CredentialID]
	if !ok || !rec.Revoked {
		t.Fatalf("revocation not persisted to disk: record=%+v ok=%v", rec, ok)
	}
}

func TestNavivoxDeviceCredentialRequiresAuth(t *testing.T) {
	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	resp, err := http.Post(server.URL+"/v1/navivox/device-credentials", "application/json",
		strings.NewReader(`{"app_install_id":"install-1"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated issue status = %d, want 401", resp.StatusCode)
	}
}
