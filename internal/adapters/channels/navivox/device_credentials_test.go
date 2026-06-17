package navivox

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
