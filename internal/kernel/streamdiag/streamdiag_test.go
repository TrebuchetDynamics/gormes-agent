package streamdiag

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

type customDropError struct{}

func (customDropError) Error() string { return " custom   drop " }

func TestFormatHeadersNormalizesSortsAndCompacts(t *testing.T) {
	got := FormatHeaders(map[string]string{
		" X-Request-ID ": " abc   def ",
		"Retry-After":    strings.Repeat("x", 130),
		" ":              "ignored",
		"Empty":          "   ",
	})
	if !strings.HasPrefix(got, "retry-after=") || !strings.Contains(got, " x-request-id=abc def") {
		t.Fatalf("FormatHeaders = %q, want sorted normalized headers", got)
	}
	if len(strings.Split(got, " ")[0]) > len("retry-after=")+120 {
		t.Fatalf("FormatHeaders did not compact long value: %q", got)
	}
}

func TestErrorChainUsesConcreteTypesAndCompactedMessages(t *testing.T) {
	err := fmt.Errorf("outer message: %w", customDropError{})
	got := ErrorChain(err)
	if !strings.Contains(got, "wrapError(outer message: custom drop)") || !strings.Contains(got, "customDropError(custom drop)") {
		t.Fatalf("ErrorChain = %q, want concrete wrapped error chain", got)
	}
}

func TestErrorTextFallbacks(t *testing.T) {
	if got := ErrorText(nil); got != "unknown stream drop" {
		t.Fatalf("ErrorText(nil) = %q", got)
	}
	if got := ErrorType(customDropError{}); got != "customDropError" {
		t.Fatalf("ErrorType(customDropError) = %q", got)
	}
	if got := ErrorChain(errors.New(strings.Repeat("x", 250))); len(got) > len("errorString(")+200+len(")") {
		t.Fatalf("ErrorChain did not compact: length %d", len(got))
	}
}
