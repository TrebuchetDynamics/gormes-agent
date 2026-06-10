package startup

import (
	"errors"
	"strings"
	"testing"
)

func TestNewErrorSanitizesMessageAndWrapsCause(t *testing.T) {
	cause := errors.New("dial tcp failed")
	err := NewError(CodeConnectError, " bearer abc.def\nfor bot123456:SECRET ", true, cause)
	if err.Code != CodeConnectError || !err.Retryable {
		t.Fatalf("NewError = %#v", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("NewError does not wrap cause")
	}
	if strings.Contains(err.Message, "SECRET") || strings.Contains(err.Message, "abc.def") || strings.Contains(err.Message, "\n") {
		t.Fatalf("message was not sanitized: %q", err.Message)
	}
}

func TestLooksLikePollingConflict(t *testing.T) {
	for _, msg := range []string{
		"terminated by other getUpdates request",
		"another bot instance is running",
		"Conflict: can't use getUpdates method while webhook is active",
	} {
		if !LooksLikePollingConflict(errors.New(msg)) {
			t.Fatalf("LooksLikePollingConflict(%q) = false", msg)
		}
	}
	if LooksLikePollingConflict(errors.New("connection reset")) {
		t.Fatal("network error classified as polling conflict")
	}
}

func TestLooksLikeNetworkError(t *testing.T) {
	for _, msg := range []string{
		"temporary failure in name resolution",
		"no such host",
		"connection refused",
		"connection reset",
		"network is unreachable",
		"read: i/o timeout",
		"Too Many Requests: retry after 5",
		"telegram: 429 Too Many Requests",
	} {
		if !LooksLikeNetworkError(errors.New(msg)) {
			t.Fatalf("LooksLikeNetworkError(%q) = false", msg)
		}
	}
	if LooksLikeNetworkError(errors.New("forbidden")) {
		t.Fatal("non-network error classified as network")
	}
}

func TestSanitizeText(t *testing.T) {
	got := SanitizeText("  bearer abc.def\nbot123456:SECRET and 123456:SECRET  ")
	if strings.Contains(got, "abc.def") || strings.Contains(got, "SECRET") || strings.Contains(got, "\n") {
		t.Fatalf("SanitizeText leaked secret or newline: %q", got)
	}
	if !strings.Contains(got, "Bearer <redacted>") || !strings.Contains(got, "bot<redacted-telegram-token>") {
		t.Fatalf("SanitizeText missing redaction markers: %q", got)
	}
}
