package msgraph

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMicrosoftGraphCredentialsFromEnvReportsMissingKeys(t *testing.T) {
	_, err := MicrosoftGraphCredentialsFromEnv(map[string]string{}, true)
	if err == nil {
		t.Fatal("MicrosoftGraphCredentialsFromEnv returned nil error for missing required keys")
	}
	for _, want := range []string{"MSGRAPH_TENANT_ID", "MSGRAPH_CLIENT_ID", "MSGRAPH_CLIENT_SECRET"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not mention %s", err, want)
		}
	}

	optional, err := MicrosoftGraphCredentialsFromEnv(map[string]string{}, false)
	if err != nil {
		t.Fatalf("optional credentials returned err: %v", err)
	}
	if optional != nil {
		t.Fatalf("optional credentials = %+v, want nil", optional)
	}
}

func TestMicrosoftGraphCredentialsFromEnvNormalizesDefaults(t *testing.T) {
	creds, err := MicrosoftGraphCredentialsFromEnv(map[string]string{
		"MSGRAPH_TENANT_ID":     " tenant-123 ",
		"MSGRAPH_CLIENT_ID":     " client-456 ",
		"MSGRAPH_CLIENT_SECRET": " secret-789 ",
	}, true)
	if err != nil {
		t.Fatalf("MicrosoftGraphCredentialsFromEnv: %v", err)
	}
	if creds.Scope != DefaultMicrosoftGraphScope {
		t.Fatalf("Scope = %q, want %q", creds.Scope, DefaultMicrosoftGraphScope)
	}
	if got := creds.TokenURL(); !strings.HasSuffix(got, "/tenant-123/oauth2/v2.0/token") {
		t.Fatalf("TokenURL = %q", got)
	}
}

func TestMicrosoftGraphTokenProviderCachesAndForceRefreshes(t *testing.T) {
	var calls atomic.Int64
	provider := NewMicrosoftGraphTokenProvider(MicrosoftGraphCredentials{
		TenantID:     "tenant",
		ClientID:     "client",
		ClientSecret: "secret",
	}, MicrosoftGraphTokenProviderOptions{
		HTTPClient: msgraphRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			body := `{"access_token":"token-` + string(rune('0'+calls.Load())) + `","expires_in":3600,"token_type":"Bearer"}`
			return msgraphStringResponse(http.StatusOK, body), nil
		}),
		Now: func() time.Time { return time.Unix(100, 0) },
	})

	first, err := provider.GetAccessToken(context.Background(), false)
	if err != nil {
		t.Fatalf("first token: %v", err)
	}
	second, err := provider.GetAccessToken(context.Background(), false)
	if err != nil {
		t.Fatalf("second token: %v", err)
	}
	third, err := provider.GetAccessToken(context.Background(), true)
	if err != nil {
		t.Fatalf("force refresh token: %v", err)
	}

	if first != "token-1" || second != "token-1" || third != "token-2" {
		t.Fatalf("tokens = %q %q %q, want cache then refresh", first, second, third)
	}
	if calls.Load() != 2 {
		t.Fatalf("HTTP calls = %d, want 2", calls.Load())
	}
}

func TestMicrosoftGraphTokenProviderCoalescesConcurrentFetch(t *testing.T) {
	var calls atomic.Int64
	provider := NewMicrosoftGraphTokenProvider(MicrosoftGraphCredentials{
		TenantID:     "tenant",
		ClientID:     "client",
		ClientSecret: "secret",
	}, MicrosoftGraphTokenProviderOptions{
		HTTPClient: msgraphRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			time.Sleep(10 * time.Millisecond)
			return msgraphStringResponse(http.StatusOK, `{"access_token":"token-1","expires_in":3600}`), nil
		}),
		Now: func() time.Time { return time.Unix(100, 0) },
	})

	var wg sync.WaitGroup
	got := make([]string, 2)
	errs := make([]error, 2)
	for i := range got {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			got[i], errs[i] = provider.GetAccessToken(context.Background(), false)
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			t.Fatalf("GetAccessToken err: %v", err)
		}
	}
	if got[0] != "token-1" || got[1] != "token-1" {
		t.Fatalf("tokens = %#v, want shared cached token", got)
	}
	if calls.Load() != 1 {
		t.Fatalf("HTTP calls = %d, want 1", calls.Load())
	}
}

func TestMicrosoftGraphTokenProviderRedactsSecretOnErrors(t *testing.T) {
	provider := NewMicrosoftGraphTokenProvider(MicrosoftGraphCredentials{
		TenantID:     "tenant",
		ClientID:     "client",
		ClientSecret: "super-secret",
	}, MicrosoftGraphTokenProviderOptions{
		HTTPClient: msgraphRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return msgraphStringResponse(http.StatusUnauthorized, `{"error":"invalid_client","error_description":"bad super-secret"}`), nil
		}),
	})

	_, err := provider.GetAccessToken(context.Background(), false)
	if err == nil {
		t.Fatal("GetAccessToken returned nil error for HTTP 401")
	}
	var graphErr *MicrosoftGraphError
	if !AsMicrosoftGraphError(err, &graphErr) {
		t.Fatalf("err type = %T, want MicrosoftGraphError", err)
	}
	if graphErr.Evidence != MicrosoftGraphTokenUnavailable {
		t.Fatalf("Evidence = %q, want %q", graphErr.Evidence, MicrosoftGraphTokenUnavailable)
	}
	if strings.Contains(err.Error(), "super-secret") {
		t.Fatalf("error leaked secret: %v", err)
	}
	if !strings.Contains(err.Error(), "[redacted]") {
		t.Fatalf("error %q should contain redaction marker", err)
	}
}

type msgraphRoundTripFunc func(*http.Request) (*http.Response, error)

func (f msgraphRoundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func msgraphStringResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
