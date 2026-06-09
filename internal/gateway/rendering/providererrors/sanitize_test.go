package providererrors

import (
	"strings"
	"testing"
)

func TestSanitizeTextRedactsSecretLikeHTMLPrefix(t *testing.T) {
	got := SanitizeText("api_key=plain-secret-token: <html><body>bad</body></html>")
	for _, forbidden := range []string{"plain-secret-token", "api_key", "<html"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("SanitizeText leaked secret-like HTML prefix %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "[redacted]: provider returned HTML error body") {
		t.Fatalf("SanitizeText missing redacted HTML prefix in %q", got)
	}
}

func TestSanitizeTextRedactsSecretLikePlainText(t *testing.T) {
	got := SanitizeText("provider failed: api_key=plain-secret-token")
	for _, forbidden := range []string{"plain-secret-token", "api_key"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("SanitizeText leaked secret-like plain text %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("SanitizeText missing redaction marker in %q", got)
	}
}

func TestSanitizeTextRedactsSecretLikeJSONMessage(t *testing.T) {
	got := SanitizeText(`Bad Request: {"error":{"message":"api_key=plain-secret-token"}}`)
	for _, forbidden := range []string{"plain-secret-token", "api_key"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("SanitizeText leaked secret-like JSON message %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("SanitizeText missing redaction marker in %q", got)
	}
}
