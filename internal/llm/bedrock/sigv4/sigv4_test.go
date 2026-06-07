package sigv4

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/bedrock/testutil"
)

func TestSignBedrockRequest_DeterministicHeaders(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":[{"text":"hello"}]}]}`)
	req, err := http.NewRequest(http.MethodPost, "https://bedrock-runtime.us-east-1.amazonaws.com/model/x/converse", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	now := time.Date(2026, 4, 26, 22, 45, 56, 0, time.UTC)
	creds := StaticAWSCredentials{
		AccessKeyID:     "AKIA_TEST",
		SecretAccessKey: "secret-test-value",
		SessionToken:    "session-token-test-value",
		Region:          "us-east-1",
	}

	if err := SignBedrockRequest(req, creds, now); err != nil {
		t.Fatalf("SignBedrockRequest() error = %v", err)
	}

	wantHashBytes := sha256.Sum256(body)
	wantHash := hex.EncodeToString(wantHashBytes[:])
	if got := req.Header.Get("X-Amz-Date"); got != "20260426T224556Z" {
		t.Fatalf("X-Amz-Date = %q, want 20260426T224556Z", got)
	}
	if got := req.Header.Get("X-Amz-Content-Sha256"); got != wantHash {
		t.Fatalf("X-Amz-Content-Sha256 = %q, want %q", got, wantHash)
	}
	auth := req.Header.Get("Authorization")
	wantPrefix := "AWS4-HMAC-SHA256 Credential=AKIA_TEST/20260426/us-east-1/bedrock/aws4_request"
	if !strings.HasPrefix(auth, wantPrefix) {
		t.Fatalf("Authorization = %q, want prefix %q", auth, wantPrefix)
	}
	if !strings.Contains(auth, "SignedHeaders=host;x-amz-content-sha256;x-amz-date;x-amz-security-token") {
		t.Fatalf("Authorization missing deterministic signed headers: %q", auth)
	}
	if got := req.Header.Get("X-Amz-Security-Token"); got != "session-token-test-value" {
		t.Fatalf("X-Amz-Security-Token = %q, want session token", got)
	}
	afterSign, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read signed body: %v", err)
	}
	if !bytes.Equal(afterSign, body) {
		t.Fatalf("signed request body changed: got %q want %q", afterSign, body)
	}
}

func TestSignBedrockRequest_ErrorRedactsSecrets(t *testing.T) {
	const (
		accessKey    = "AKIA_TEST_REDACT_ME"
		secretKey    = "secret-test-value-redact-me"
		sessionToken = "session-token-test-value-redact-me"
	)
	signErr := SignBedrockRequest(nil, StaticAWSCredentials{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		SessionToken:    sessionToken,
		Region:          "us-east-1",
	}, time.Date(2026, 4, 26, 22, 45, 56, 0, time.UTC))
	if signErr == nil {
		t.Fatal("SignBedrockRequest(nil) error = nil, want redacted error")
	}
	if testutil.ContainsAnyString(signErr.Error(), accessKey, secretKey, sessionToken) {
		t.Fatalf("sign error leaked credential material: %q", signErr.Error())
	}
}
